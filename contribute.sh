#!/usr/bin/bash
# ============================================================
# 节点贡献脚本（静默安装）
#
# 用途：把一台闲置 Linux 机器贡献为拨测节点。
#   - 全程无交互：端口自动挑一个空闲高位端口，节点标识随机生成
#   - 命名与官方 install.sh 保持一致：二进制 lemonipw / 目录 /opt/lemon-ipw / 服务 lemon-ipw
#   - 默认同时接入柠檬的两个中间件 WS 通道（多活，节点主动出站，无需公网放行端口）
#   - 安装完成后打印节点信息，提交给管理员登记即可上线
#
# 用法：
#   sudo bash contribute.sh                  # 全默认：自动端口 + 接 WS
#   sudo bash contribute.sh --ipdb           # 启用 IP 数据库（首启后台下载 ~450MB）
#   sudo bash contribute.sh --port 34567     # 指定端口
#   sudo bash contribute.sh --dry-run        # 只生成并打印信息，不下载、不落盘
# ============================================================

set -e

# 默认同时接入的两个中间件（逗号分隔 = 节点侧同时连接全部、多活；
# 任一断开只重连自己，不影响另一条连接）。顺序无关，收集中心放前面便于阅读。
WS_URL_DEFAULT="wss://boce-api.api-ipw.wsmdn.top/ws,wss://middleware-1.api-ipw.wsmdn.top/ws"
CONTACT_EMAIL="iduhih777@outlook.com"
PORT_MIN=20000
PORT_MAX=65000

OPT_PORT=""
OPT_WS_URL=""
OPT_VERSION=""
OPT_GH_PROXY=""
OPT_IPDB="false"
DRY_RUN="false"

usage() {
    cat <<'EOF'
用法: sudo bash contribute.sh [选项]

  --port <n>        监听端口（默认：20000-65000 中的随机空闲端口）
  --ws-url <url>    中间件 WS 地址，逗号分隔可多个（默认：柠檬两个中间件，同时连接）
  --version <tag>   指定版本（默认：取 GitHub 最新 release）
  --gh-proxy <p>    GitHub 下载加速前缀，如 https://ghfast.top/
  --ipdb            启用 IP 数据库（首次启动后台下载，约 450MB；默认关闭）
  --dry-run         只做检查与信息生成，不下载、不写 systemd
  -h, --help        显示本帮助
EOF
}

# ---------- 参数解析 ----------

while [ $# -gt 0 ]; do
    key="$1"
    case "$key" in
        --ipdb)     OPT_IPDB="true";  shift; continue ;;
        --dry-run)  DRY_RUN="true";  shift; continue ;;
        -h|--help)  usage; exit 0 ;;
        --port|--ws-url|--version|--gh-proxy) ;;
        *) echo "错误：未知参数 $key（用 --help 查看用法）" >&2; exit 1 ;;
    esac
    if [ $# -lt 2 ]; then
        echo "错误：$key 缺少参数值（用 --help 查看用法）" >&2
        exit 1
    fi
    case "$key" in
        --port)     OPT_PORT="$2" ;;
        --ws-url)   OPT_WS_URL="$2" ;;
        --version)  OPT_VERSION="$2" ;;
        --gh-proxy) OPT_GH_PROXY="$2" ;;
    esac
    shift 2
done

# ---------- 随机值生成 ----------

# 生成 UUID（优先 /proc，其次 uuidgen，最后 /dev/urandom）
gen_uuid() {
    if [ -r /proc/sys/kernel/random/uuid ]; then
        cat /proc/sys/kernel/random/uuid
    elif command -v uuidgen >/dev/null 2>&1; then
        uuidgen | tr 'A-Z' 'a-z'
    else
        od -An -N16 -tx1 /dev/urandom | tr -d ' \n' \
            | sed 's/\(........\)\(....\)\(....\)\(....\)\(............\)/\1-\2-\3-\4-\5/'
    fi
}

# 32 位十六进制令牌（去横线的 UUID）
gen_token() { gen_uuid | tr -d '-'; }

# ---------- 端口 ----------

# 列出本机监听端口（ss 优先，netstat 次之）
listening_ports() {
    if command -v ss >/dev/null 2>&1; then
        ss -ltn 2>/dev/null | awk 'NR>1 {print $4}'
    elif command -v netstat >/dev/null 2>&1; then
        netstat -ltn 2>/dev/null | awk 'NR>2 {print $4}'
    else
        return 1
    fi
}

# 0 = 端口已被占用，1 = 空闲
port_in_use() {
    local p="$1" ports
    ports=$(listening_ports || true)
    if [ -n "$ports" ]; then
        if printf '%s\n' "$ports" | sed 's/.*[.:]//' | grep -qx "$p"; then
            return 0
        fi
        return 1
    fi
    # 兜底：连得上即视为被占用（无 ss / netstat 的极简系统）
    if (exec 3<>"/dev/tcp/127.0.0.1/$p") 2>/dev/null; then
        exec 3>&-
        return 0
    fi
    return 1
}

# 随机高位端口（20000-65000），连续试探直到落到空闲端口
pick_port() {
    local i p
    for i in $(seq 1 50); do
        p=$(( (RANDOM * 32768 + RANDOM) % (PORT_MAX - PORT_MIN + 1) + PORT_MIN ))
        if ! port_in_use "$p"; then
            printf '%s' "$p"
            return 0
        fi
    done
    echo "错误：连续 50 次未找到空闲的高位端口，请用 --port 指定" >&2
    exit 1
}

# ---------- 环境检查 ----------

if [ "$DRY_RUN" != "true" ]; then
    if [ "$(id -u)" -ne 0 ]; then
        echo "错误：需要 root 权限（写入 /etc/systemd/system 与安装目录），请用 sudo 运行" >&2
        exit 1
    fi
    if ! command -v systemctl >/dev/null 2>&1; then
        echo "错误：未检测到 systemd（systemctl 不存在），无法创建守护进程" >&2
        exit 1
    fi
    if ! command -v wget >/dev/null 2>&1 && ! command -v curl >/dev/null 2>&1; then
        echo "错误：未找到 wget 或 curl，无法下载" >&2
        exit 1
    fi
fi

# ---------- 架构检测 ----------
case "$(uname -m)" in
    x86_64)              ARCH_TAG="amd64" ;;
    i386|i486|i586|i686) ARCH_TAG="386" ;;
    aarch64|arm64)       ARCH_TAG="arm64" ;;
    armv7l|armv7hl)      ARCH_TAG="armv7" ;;
    armv6l)              ARCH_TAG="armv6" ;;
    loongarch64)         ARCH_TAG="loong64" ;;
    *)
        echo "错误：不支持的架构 '$(uname -m)'" >&2
        exit 1
        ;;
esac
RELEASE_ASSET="lemonipw-linux-${ARCH_TAG}"

# ---------- 命名（与 install.sh 保持一致） ----------

# 二进制 / 安装目录 / 服务名统一固定为 lemon 系列，便于运维识别与排查；
# 一台机器只跑一份，重跑本脚本 = 覆盖同名服务与目录。
BIN_BASE="lemonipw"
INSTALL_DIR="/opt/lemon-ipw"
SERVICE_NAME="lemon-ipw"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

# 同名服务已存在（例如这台机器已经跑着一个节点）：再跑一次会用新生成的
# 节点标识覆盖旧配置，旧节点在收集中心的记录就失效了，先提醒一句。
if [ "$DRY_RUN" != "true" ] && [ -f "$SERVICE_FILE" ]; then
    echo "提示：已存在服务 ${SERVICE_NAME}，本次会覆盖其配置并生成新的节点标识（旧标识在收集中心将失效）" >&2
fi

PORTS="${OPT_PORT:-$(pick_port)}"
case "$PORTS" in
    ""|*[!0-9]*)
        echo "错误：端口必须是数字（当前值：$PORTS）" >&2
        exit 1
        ;;
esac
# 10# 前缀强制十进制：避免 "0800" 这类带前导 0 的值被当成八进制解析
PORTS=$((10#$PORTS))
if [ "$PORTS" -lt 1 ] || [ "$PORTS" -gt 65535 ]; then
    echo "错误：端口超出范围 1-65535（当前值：$PORTS）" >&2
    exit 1
fi
if [ -n "$OPT_PORT" ] && port_in_use "$PORTS"; then
    echo "警告：端口 $PORTS 当前已被占用，节点可能启动失败" >&2
fi

WS_URL="${OPT_WS_URL:-$WS_URL_DEFAULT}"
# 逐段校验（逗号分隔可多个），并去掉段内空格：节点侧虽会 trim，
# 但空格会原样进 systemd 的 Environment 值，还是在这里清掉更干净。
WS_URL_SEGMENTS=""
_old_ifs="$IFS"; IFS=','
for _seg in $WS_URL; do
    # 去前导/尾随空白（IFS 只含逗号，空白不会被自动去掉）
    _seg="${_seg#"${_seg%%[![:space:]]*}"}"
    _seg="${_seg%"${_seg##*[![:space:]]}"}"
    [ -n "$_seg" ] || continue
    case "$_seg" in
        ws://*|wss://*) ;;
        *)
            IFS="$_old_ifs"
            echo "错误：WS 地址的每一段都需以 ws:// 或 wss:// 开头（问题段：${_seg}）" >&2
            exit 1
            ;;
    esac
    case "$_seg" in
        */ws|*/ws/) ;;
        *) echo "警告：WS 地址 ${_seg} 看起来不含 /ws 路径，节点可能连不上（节点默认连 <地址>/ws）" >&2 ;;
    esac
    WS_URL_SEGMENTS="${WS_URL_SEGMENTS:+${WS_URL_SEGMENTS},}${_seg}"
done
IFS="$_old_ifs"
if [ -z "$WS_URL_SEGMENTS" ]; then
    echo "错误：WS 地址为空（当前值：${WS_URL}）" >&2
    exit 1
fi
WS_URL="$WS_URL_SEGMENTS"

NODE_ID=$(gen_uuid)
NODE_KEY=$(gen_token)
ACCESS_TOKEN=$(gen_token)

# ---------- 公网地址探测（失败不阻断） ----------

detect_ip() {
    local url="$1" out=""
    if command -v curl >/dev/null 2>&1; then
        out=$(curl -fsS -m 5 "$url" 2>/dev/null || true)
    elif command -v wget >/dev/null 2>&1; then
        out=$(wget -qO- --timeout=5 "$url" 2>/dev/null || true)
    fi
    printf '%s' "$out" | tr -d ' \t\r\n'
}

is_ipv4() { printf '%s' "$1" | grep -qE '^([0-9]{1,3}\.){3}[0-9]{1,3}$'; }
is_ipv6() {
    printf '%s' "$1" | grep -qE '^[0-9A-Fa-f:]+$' || return 1
    printf '%s' "$1" | grep -q ':' || return 1
    return 0
}

IPV4=$(detect_ip "https://4.wsmdn.top/")
is_ipv4 "$IPV4" || IPV4=""
IPV6=$(detect_ip "https://6.wsmdn.top/")
is_ipv6 "$IPV6" || IPV6=""

# 单栈判定：只有一侧能出网就按单栈跑，避免另一协议的拨测全失败刷错误日志
SINGLE_STACK=""
STACK_LABEL="双栈"
if [ -n "$IPV4" ] && [ -z "$IPV6" ]; then
    SINGLE_STACK="ipv4"
    STACK_LABEL="IPv4"
elif [ -z "$IPV4" ] && [ -n "$IPV6" ]; then
    SINGLE_STACK="ipv6"
    STACK_LABEL="IPv6"
fi

# ---------- 版本与下载地址 ----------

fetch_latest_version() {
    local api="https://api.github.com/repos/nomdn/ipw-cn/releases/latest" raw=""
    if command -v curl >/dev/null 2>&1; then
        raw=$(curl -fsS -m 20 "$api" 2>/dev/null || true)
    else
        raw=$(wget -qO- --timeout=20 "$api" 2>/dev/null || true)
    fi
    printf '%s' "$raw" | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4
}

VERSION="$OPT_VERSION"
if [ -z "$VERSION" ]; then
    VERSION=$(fetch_latest_version || true)
fi
if [ -z "$VERSION" ]; then
    echo "错误：无法获取最新版本号（GitHub API 不可达或限流）。可用 --version <tag> 指定，例如 --version v3.7.0" >&2
    exit 1
fi

DOWNLOAD_URL="https://github.com/nomdn/ipw-cn/releases/download/${VERSION}/${RELEASE_ASSET}"
if [ -n "$OPT_GH_PROXY" ]; then
    DOWNLOAD_URL="${OPT_GH_PROXY%/}/${DOWNLOAD_URL}"
fi

# ---------- 输出节点信息 ----------

print_info() {
    local dry_note=""
    if [ "$DRY_RUN" = "true" ]; then
        dry_note="（dry-run，未实际安装）"
    fi
    local ipv4_disp="${IPV4:-检测失败}"
    local ipv6_disp="${IPV6:-检测失败}"
    local ipdb_disp="未启用（仅影响 IP 归属地/ASN 类拨测，需要时用 --ipdb 重装启用）"
    if [ "$OPT_IPDB" = "true" ]; then
        ipdb_disp="已启用（首次启动后台下载，约 450MB）"
    fi
    # WS 地址逐个列出（可能接多个中间件），首个带标签、其余对齐缩进
    local ws_block=""
    local _first="true" _u=""
    local _old_ifs="$IFS"; IFS=','
    for _u in $WS_URL; do
        if [ "$_first" = "true" ]; then
            ws_block="  WS 地址:        ${_u}"
            _first="false"
        else
            ws_block="${ws_block}
                  ${_u}"
        fi
    done
    IFS="$_old_ifs"
    cat << EOF

========================================
 贡献节点${dry_note}已就绪，请把以下信息提交给管理员
========================================
  节点 id:        ${NODE_ID}
  注册 key:       ${NODE_KEY}
${ws_block}
  监听端口:       ${PORTS}
  单栈模式:       ${STACK_LABEL}
  access-token:   ${ACCESS_TOKEN}
  CORS:           不限
  公网 IPv4:      ${ipv4_disp}
  公网 IPv6:      ${ipv6_disp}
  地区-运营商:     请自行填写
----------------------------------------
 中间件 ws-keys（**每个 WS 地址对应的中间件都要登记**，缺一个该条连接会被拒 401）:
   "${NODE_ID}": "${NODE_KEY}"
 中心 / 中间件 api-keys（HTTP 直连本节点时的鉴权）:
   "${NODE_ID}": "${ACCESS_TOKEN}"
----------------------------------------
 提交邮箱: ${CONTACT_EMAIL}
 说明:
   1. 本节点默认以 WS 方式同时接入上面的中间件（节点主动出站连接），无需公网放行 ${PORTS}/TCP。
   2. 若接收方希望以 HTTP 方式直连本节点，需放行 ${PORTS}/TCP 且该端口公网可达。
   3. IP 数据库: ${ipdb_disp}
----------------------------------------
 运维命令:
   状态: systemctl status ${SERVICE_NAME}
   日志: journalctl -u ${SERVICE_NAME} -f
   重启: systemctl restart ${SERVICE_NAME}
   卸载: systemctl disable --now ${SERVICE_NAME} && rm -f ${SERVICE_FILE} && rm -rf ${INSTALL_DIR}
 节点信息备份: ${INSTALL_DIR}/node-info.txt
========================================
EOF
}

# ---------- dry-run：只展示 ----------

if [ "$DRY_RUN" = "true" ]; then
    echo "架构:     $(uname -m) → ${RELEASE_ASSET}"
    echo "版本:     ${VERSION}"
    echo "下载地址: ${DOWNLOAD_URL}"
    echo "安装目录: ${INSTALL_DIR}"
    echo "服务名:   ${SERVICE_NAME}"
    echo "随机:     端口=${PORTS} 节点id=${NODE_ID}"
    print_info
    exit 0
fi

# ---------- 下载二进制 ----------

echo "正在下载 ${VERSION} / ${RELEASE_ASSET} ..."
mkdir -p "$INSTALL_DIR"
if command -v wget >/dev/null 2>&1; then
    wget -q -O "${INSTALL_DIR}/${BIN_BASE}" "$DOWNLOAD_URL"
else
    # -f 让 4xx 直接失败，避免把 GitHub 的 Not Found 页面存成二进制
    curl -fsSL -o "${INSTALL_DIR}/${BIN_BASE}" "$DOWNLOAD_URL"
fi
chmod +x "${INSTALL_DIR}/${BIN_BASE}"

# ---------- 生成 systemd 服务 ----------

# 仅非空项写入，避免空值覆盖节点自身默认值。
# 值内的双引号 / 反斜杠 / % 必须转义：前两者会破坏 systemd 的引号包裹，
# % 是 unit 说明符（要写成 %% 才是字面量）。
ENV_LINES=""
add_env() {
    local name="$1" value="$2"
    [ -n "$value" ] || return 0
    value="${value//\\/\\\\}"
    value="${value//\"/\\\"}"
    value="${value//%/%%}"
    ENV_LINES="${ENV_LINES}Environment=\"${name}=${value}\"
"
}
add_env PORTS "$PORTS"
add_env SINGLE_STACK "$SINGLE_STACK"
add_env WS_URL "$WS_URL"
add_env NODE_ID "$NODE_ID"
add_env NODE_KEY "$NODE_KEY"
add_env ACCESS_TOKEN "$ACCESS_TOKEN"
add_env IPDB "$OPT_IPDB"

cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Lemon IPW Backend Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${BIN_BASE}
${ENV_LINES}Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# ---------- 启动服务 ----------

systemctl daemon-reload
systemctl enable --now "$SERVICE_NAME" >/dev/null 2>&1 || true
sleep 2
if ! systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "错误：服务未能启动，最近日志：" >&2
    journalctl -u "$SERVICE_NAME" --no-pager -n 20 >&2 || true
    exit 1
fi

# ---------- 落一份信息备份（节点标识随机，记不住） ----------

{
    echo "node_id=${NODE_ID}"
    echo "node_key=${NODE_KEY}"
    echo "access_token=${ACCESS_TOKEN}"
    echo "ws_url=${WS_URL}"
    echo "port=${PORTS}"
    echo "single_stack=${SINGLE_STACK:-双栈}"
    echo "service=${SERVICE_NAME}"
    echo "install_dir=${INSTALL_DIR}"
    echo "version=${VERSION}"
} > "${INSTALL_DIR}/node-info.txt"
chmod 600 "${INSTALL_DIR}/node-info.txt"

# 本地存活自检（只打本机回环，不依赖接收方是否已登记）
LOCAL_CHECK="失败"
if command -v curl >/dev/null 2>&1; then
    if curl -fsS -m 5 "http://127.0.0.1:${PORTS}/" >/dev/null 2>&1; then
        LOCAL_CHECK='正常（{"status":"ok"}）'
    fi
fi
echo "本地自检: ${LOCAL_CHECK}"

print_info
