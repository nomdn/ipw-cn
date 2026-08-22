#!/usr/bin/bash
# ============================================================
# 1IPW.CN 后端节点一键安装脚本
#   - 自动检测架构并下载最新 release 二进制
#   - 交互式输入配置（环境变量注入，无需 setting.json）
#   - 生成并启用 systemd 守护进程
# 用法：sudo bash install.sh
# ============================================================

text="
██╗██████╗ ██╗    ██╗      ███╗   ██╗ ██████╗ ██████╗ ███████╗
██║██╔══██╗██║    ██║      ████╗  ██║██╔═══██╗██╔══██╗██╔════╝
██║██████╔╝██║ █╗ ██║█████╗██╔██╗ ██║██║   ██║██║  ██║█████╗  
██║██╔═══╝ ██║███╗██║╚════╝██║╚██╗██║██║   ██║██║  ██║██╔══╝  
██║██║     ╚███╔███╔╝      ██║ ╚████║╚██████╔╝██████╔╝███████╗
╚═╝╚═╝      ╚══╝╚══╝       ╚═╝  ╚═══╝ ╚═════╝ ╚═════╝ ╚══════╝
                                                              
"
echo "$text"

set -e

# ---------- 前置检查 ----------
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

# ---------- 架构检测 ----------
systemArch=$(uname -m)
case "$systemArch" in
    x86_64)         BINARY="lemonipw-linux-amd64" ;;
    i386|i486|i586|i686) BINARY="lemonipw-linux-386" ;;
    aarch64|arm64)  BINARY="lemonipw-linux-arm64" ;;
    armv7l|armv7hl) BINARY="lemonipw-linux-armv7" ;;
    armv6l)         BINARY="lemonipw-linux-armv6" ;;
    loongarch64)    BINARY="lemonipw-linux-loong64" ;;
    *)
        echo "错误：不支持的架构 '$systemArch'" >&2
        exit 1
        ;;
esac
echo "检测到架构 $systemArch，将下载 $BINARY"

# ---------- 交互式配置输入 ----------

# 生成 UUID（优先 /proc，其次 uuidgen，最后 /dev/urandom）
gen_uuid() {
    if [ -r /proc/sys/kernel/random/uuid ]; then
        cat /proc/sys/kernel/random/uuid
    elif command -v uuidgen >/dev/null 2>&1; then
        uuidgen | tr 'A-Z' 'a-z'
    else
        od -An -N16 -tx1 /dev/urandom | tr -d ' \n' | sed 's/\(........\)\(....\)\(....\)\(....\)\(............\)/\1-\2-\3-\4-\5/'
    fi
}

echo ""
echo "========================================"
echo " 配置后端节点（直接回车使用默认值）"
echo "========================================"

read -r -p "安装目录 [/opt/lemon-ipw]: " INSTALL_DIR
INSTALL_DIR=${INSTALL_DIR:-/opt/lemon-ipw}

read -r -p "监听端口 [8080]: " PORTS
PORTS=${PORTS:-8080}

read -r -p "单栈模式 SINGLE_STACK（留空=双栈，ipv4 或 ipv6）: " SINGLE_STACK

read -r -p "访问令牌 access_token（留空=不启用鉴权）: " ACCESS_TOKEN

read -r -p "DNS 服务器 [119.28.28.28:53]（主从逗号分隔，如 119.28.28.28:53,223.5.5.5:53）: " DNS_SERVER
DNS_SERVER=${DNS_SERVER:-119.28.28.28:53}

read -r -p "DNSSEC 专用 DNS（留空=沿用上面 dns-server）: " DNSSEC_DNS_SERVER

read -r -p "启用 IP 数据库 ipdb（首次启动下载约 200MB）[Y/n]: " IPDB_CHOICE
case "${IPDB_CHOICE,,}" in
    n|no) IPDB="false" ;;
    *)    IPDB="true" ;;
esac

read -r -p "CORS 允许来源（逗号分隔，留空=不限）: " CORS

read -r -p "远端配置地址 remote-config-url（留空=不启用）: " REMOTE_CONFIG_URL

echo ""
echo "--- WS 通道接入（可选，接入独立中间件）---"
read -r -p "接入中间件 WS 通道？[y/N]: " WS_CHOICE
WS_URL=""
NODE_ID=""
NODE_KEY=""
WS_USED_DEFAULT=""
case "${WS_CHOICE,,}" in
    y|yes)
        read -r -p "  中间件 WS 完整地址（含 wss:// 前缀与 /ws 路径，如 wss://host:8092/ws；留空=默认 wss://middleware-1.api-ipw.wsmdn.top/ws；逗号分隔可多备）: " WS_URL
        if [ -z "$WS_URL" ]; then
            WS_URL="wss://middleware-1.api-ipw.wsmdn.top/ws"
            WS_USED_DEFAULT="true"
        fi
        NODE_ID=$(gen_uuid)
        echo "  节点 id（自动生成 UUID）: $NODE_ID"
        while [ -z "$NODE_KEY" ]; do
            read -r -p "  注册 key（必填，中间件 apiKeys 必须包含此节点，否则注册被拒 401）: " NODE_KEY
            if [ -z "$NODE_KEY" ]; then
                echo "  错误：不加 key 禁止启用 WS，必须提供注册 key"
            fi
        done
        ;;
esac

echo ""
echo "--- 其他环境变量（可选）---"
echo "输入额外的 Environment= 项，每行一个（如 GH_PROXY=https://ghproxy.com/），空行结束："
EXTRA_ENVS=""
while true; do
    read -r -p "  > " EXTRA
    [ -z "$EXTRA" ] && break
    if [ -z "$EXTRA_ENVS" ]; then
        EXTRA_ENVS="$EXTRA"
    else
        EXTRA_ENVS="$EXTRA_ENVS
$EXTRA"
    fi
done

# ---------- 汇总展示 ----------
echo ""
echo "========================================"
echo " 配置汇总"
echo "========================================"
echo "安装目录:     $INSTALL_DIR"
echo "服务名:       lemon-ipw"
echo "监听端口:     $PORTS"
echo "单栈模式:     ${SINGLE_STACK:-双栈}"
echo "access_token: ${ACCESS_TOKEN:+已设置 (隐藏)}"
echo "DNS:          $DNS_SERVER"
echo "ipdb:         $IPDB"
echo "WS 接入:      ${WS_URL:+$WS_URL (id=$NODE_ID, key=${NODE_KEY:+已设置})}${WS_URL:-未启用}"
echo "========================================"
read -r -p "确认安装？[Y/n]: " CONFIRM
case "${CONFIRM,,}" in
    n|no) echo "已取消" ; exit 0 ;;
    *) : ;;
esac

# ---------- 下载二进制 ----------
echo ""
echo "正在获取最新版本..."
if command -v curl >/dev/null 2>&1; then
    LATEST_TAG=$(curl -s https://api.github.com/repos/nomdn/ipw-cn/releases/latest | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)
else
    LATEST_TAG=$(wget -qO- https://api.github.com/repos/nomdn/ipw-cn/releases/latest | grep -o '"tag_name": *"[^"]*"' | head -1 | cut -d'"' -f4)
fi
if [ -z "$LATEST_TAG" ]; then
    echo "错误：无法获取最新版本号（检查网络或 GitHub API 速率限制）" >&2
    exit 1
fi
echo "最新版本：$LATEST_TAG"
DOWNLOAD_URL="https://github.com/nomdn/ipw-cn/releases/download/$LATEST_TAG/$BINARY"

mkdir -p "$INSTALL_DIR"
echo "下载 $BINARY ..."
if command -v wget >/dev/null 2>&1; then
    wget -q -O "$INSTALL_DIR/lemonipw" "$DOWNLOAD_URL"
else
    curl -sL -o "$INSTALL_DIR/lemonipw" "$DOWNLOAD_URL"
fi
chmod +x "$INSTALL_DIR/lemonipw"
echo "二进制已安装到 $INSTALL_DIR/lemonipw"

# ---------- 生成 systemd 服务 ----------
SERVICE_FILE="/etc/systemd/system/lemon-ipw.service"

# 收集环境变量（仅非空的写入，避免空值覆盖默认）
ENV_LINES=""
[ -n "$PORTS" ]             && ENV_LINES="${ENV_LINES}Environment=\"PORTS=$PORTS\"
"
[ -n "$SINGLE_STACK" ]      && ENV_LINES="${ENV_LINES}Environment=\"SINGLE_STACK=$SINGLE_STACK\"
"
[ -n "$ACCESS_TOKEN" ]      && ENV_LINES="${ENV_LINES}Environment=\"ACCESS_TOKEN=$ACCESS_TOKEN\"
"
[ -n "$DNS_SERVER" ]        && ENV_LINES="${ENV_LINES}Environment=\"DNS_SERVER=$DNS_SERVER\"
"
[ -n "$DNSSEC_DNS_SERVER" ] && ENV_LINES="${ENV_LINES}Environment=\"DNSSEC_DNS_SERVER=$DNSSEC_DNS_SERVER\"
"
[ -n "$IPDB" ]              && ENV_LINES="${ENV_LINES}Environment=\"IPDB=$IPDB\"
"
[ -n "$CORS" ]              && ENV_LINES="${ENV_LINES}Environment=\"CORS=$CORS\"
"
[ -n "$REMOTE_CONFIG_URL" ] && ENV_LINES="${ENV_LINES}Environment=\"REMOTE_CONFIG_URL=$REMOTE_CONFIG_URL\"
"
[ -n "$WS_URL" ]            && ENV_LINES="${ENV_LINES}Environment=\"WS_URL=$WS_URL\"
"
[ -n "$NODE_ID" ]           && ENV_LINES="${ENV_LINES}Environment=\"NODE_ID=$NODE_ID\"
"
[ -n "$NODE_KEY" ]          && ENV_LINES="${ENV_LINES}Environment=\"NODE_KEY=$NODE_KEY\"
"
if [ -n "$EXTRA_ENVS" ]; then
    while IFS= read -r line; do
        [ -n "$line" ] && ENV_LINES="${ENV_LINES}Environment=\"$line\"
"
    done <<< "$EXTRA_ENVS"
fi

cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Lemon IPW Backend Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/lemonipw
${ENV_LINES}Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

echo "已生成服务文件 $SERVICE_FILE"

# ---------- 启动服务 ----------
systemctl daemon-reload
systemctl enable --now lemon-ipw
systemctl status lemon-ipw --no-pager | head -15

echo ""
echo "========================================"
echo " 安装完成"
echo "========================================"
echo "常用命令："
echo "  systemctl status lemon-ipw      # 查看状态"
echo "  journalctl -u lemon-ipw -f      # 查看日志"
echo "  systemctl restart lemon-ipw     # 重启（改配置后）"
echo "验证：curl http://127.0.0.1:$PORTS/ 应返回 {\"status\":\"ok\"}"
if [ -n "$WS_URL" ]; then
    echo ""
    echo "WS 通道已启用，请把节点注册信息加入中间件 setting.json 的 apiKeys："
    echo "  \"$NODE_ID\": \"$NODE_KEY\""
fi
if [ -n "$WS_USED_DEFAULT" ]; then
    echo ""
    echo "========================================"
    echo "感谢您的贡献，请向 iduhih777@outlook.com 提交您的节点信息，谢谢！"
    echo "  节点 id:      $NODE_ID"
    echo "  注册 key:     $NODE_KEY"
    echo "  WS 地址:      $WS_URL"
    echo "  监听端口:     $PORTS"
    echo "  单栈模式:     ${SINGLE_STACK:-双栈}"
    echo "  access_token: ${ACCESS_TOKEN:+已设置 (隐藏)}"
    echo "  CORS:         ${CORS:-不限}"
    echo "  地区-运营商    请自行填写"
    echo "========================================"
fi
echo "========================================"
