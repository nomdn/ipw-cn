# 后端节点部署

后端节点提供全部检测 API（IP 归属地 / ASN / SSL / DNS / DNSSEC / Whois / TCPing / 测速 / 截图等），可在任意地区部署，组成去中心化测试网络。

## 方案一：EdgeOne

腾讯云 EdgeOne 支持 Go 边缘函数，点击下方按钮通过 **EdgeOne Makers** 一键部署 `edgeone/` 版本（无需本地构建）：

[![使用 EdgeOne Makers 部署](https://cdnstatic.tencentcs.com/edgeone/pages/deploy.svg)](https://console.cloud.tencent.com/edgeone/makers/new?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=edgeone)


也可以使用 EdgeOne CLI 手动部署：

```bash
cd edgeone
npx edgeone pages deploy -n ipw-cn -t $EDGEONE_API_TOKEN
```

`edgeone/` 版本的配置来源为 环境变量 + 远端配置（`REMOTE_CONFIG_URL`）。

## 方案二：Vercel

点击下方按钮一键导入仓库（部署目标为 `edgeone/cloud-functions` 目录，即 Go 边缘函数版本）：

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=edgeone%2Fcloud-functions)

> [!IMPORTANT]
> **Serverless 节点均为 IPv4-only**（EdgeOne、Vercel 等平台的出站网络只提供 IPv4）。部署后需要：
>
> 1. 给节点配置 `SINGLE_STACK=ipv4`（声明节点为 IPv4 单栈），跳过 IPv6 相关测试，避免无意义的错误日志与超时
> 2. 在前端配置 `config/index.ts` 中，**该节点只能填入 `APIBaseURL.IPv4`**（拨测节点池的 IPv4 栈），不要放进 `DualStack` / `IPv6` 栈

## 方案三：Docker

仓库根目录提供 `Dockerfile`（多阶段构建，产物基于 Alpine）：

```bash
docker build -t lemon-ipw .
docker run -d -p 8080:8080 \
  -v $(pwd)/setting.json:/app/setting.json \
  lemon-ipw
```

配置通过挂载 `setting.json` 提供，也可用环境变量覆盖（`PORTS` / `CORS` / `ACCESS_TOKEN` 等）。

## 方案四：二进制

```bash
# 编译
go build -o lemonipw main.go

# 运行（与 setting.json 同目录）
./lemonipw
```

首次启动自动下载 IP 数据库（约 200MB），之后每 24 小时自动更新一次。需要守护运行时，可参考下方「方案五」生成的 systemd 服务，或手动创建 service 文件（`ExecStart` 指向二进制，`WorkingDirectory` 指向 `setting.json` 所在目录）。

## 方案五：一键安装（install.sh）

仓库根目录提供 `install.sh` 一键安装脚本：自动检测架构 → 下载最新 release 二进制 → **交互式输入配置**（无需准备 setting.json，配置以环境变量注入）→ 生成 systemd 服务并守护进程。

```bash
# 先下载脚本到本地（勿用 curl | bash 管道执行，避免下载中断导致语法解析错误）
git clone https://github.com/nomdn/ipw-cn && cd ipw-cn   # 或
curl -fsSL https://raw.githubusercontent.com/nomdn/ipw-cn/main/install.sh -o install.sh

sudo bash install.sh
```

**交互式配置项**（直接回车使用默认值）：

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| 安装目录 | `/opt/lemon-ipw` | 二进制与工作目录 |
| 监听端口 | `8080` | `PORTS` |
| access_token | 留空 | 留空 = 不启用鉴权 |
| DNS 服务器 | `119.28.28.28:53` | 主从逗号分隔（`119.28.28.28:53,223.5.5.5:53`） |
| DNSSEC 专用 DNS | 留空 | 留空 = 沿用 dns-server |
| IP 数据库 | `Y` | `IPDB`，首次启动下载约 200MB |
| CORS | 留空 | 逗号分隔允许来源 |
| 远端配置地址 | 留空 | `REMOTE_CONFIG_URL` |
| WS 通道接入 | `N` | 选 `y` 后继续输入 WS_URL / NODE_ID / NODE_KEY |
| 其他环境变量 | 无 | 每行一个 `K=V`，空行结束 |

**WS 接入特殊规则**：

- `WS_URL` 留空时自动使用默认值 `wss://middleware-1.api-ipw.wsmdn.top/ws`；填写自己的中间件地址时须为**完整路径**（含 `wss://` 前缀与 `/ws` 路径，如 `wss://host:8092/ws`），逗号分隔多备
- `NODE_ID` **强制自动生成 UUID**（无需输入，如 `4580ea9d-2a16-4b8e-8090-5f2e7f0a2229`），适合作为节点唯一标识
- `NODE_KEY` **必填**：不加 key 禁止启用 WS（中间件 `wsKeys` 必须包含该节点，否则注册被拒 401）；安装完成后脚本会提示把 `"<NODE_ID>": "<NODE_KEY>"` 加入中间件 setting.json 的 `wsKeys`

**安装完成后自动生成并启动 systemd 服务**（`lemon-ipw.service`），配置以 `Environment="K=V"` 注入，无需 setting.json。常用管理命令：

```bash
systemctl status lemon-ipw        # 状态
journalctl -u lemon-ipw -f        # 日志
sudo systemctl restart lemon-ipw  # 重启（改配置后）
```

修改配置：`sudo systemctl edit --full lemon-ipw` 编辑环境变量，保存后 `sudo systemctl daemon-reload && sudo systemctl restart lemon-ipw` 生效。

> [!WARNING]
> **版本兼容提醒**：v1.1.0 及更早的 release 二进制读的是旧环境变量名 `IPW_WS_URL` / `IPW_NODE_ID` / `IPW_NODE_KEY`（8-20 及之前构建）。install.sh 生成的 service 用的是新名 `WS_URL` / `NODE_ID` / `NODE_KEY`——**旧二进制 + 新 service 会因读不到配置而完全不启用 WS**（启动日志无任何 ws 输出）。升级到包含改名后的新 release 即可；急用可临时把 service 环境变量改回 `IPW_` 前缀。

## 配置

后端配置见 [配置文件](/guide/config)：`setting.json` 或环境变量，支持远端配置（`REMOTE_CONFIG_URL`，优先级：远端 > 环境变量 > setting.json）。

## WS 通道接入（可选）

后端节点可作为 WS 客户端接入 [独立中间件](/guide/deploy-middleware) 的 WS 通道，拨测请求经 WebSocket 转发，节点本地执行探针后回传。不配置则完全走原 HTTP 接口。

**配置三个环境变量**（或 setting.json 的 `ws-url` / `node-id` / `node-key`，env 优先；也支持远端配置覆盖，除非列入 `remote-ingore-config`）：

```bash
WS_URL=ws://<中间件IP>:8092/ws \
NODE_ID=<节点id，与中间件 APIBaseURL / IPLocationAPI 池的节点 id 一致> \
NODE_KEY=<注册key，与中间件 wsKeys[节点id] 一致；节点未配置 key 可留空> \
./lemonipw
```

- `WS_URL` 支持**逗号分隔多个中间件**（`ws://主:8092/ws,ws://备:8092/ws`），第一个为主，主故障自动切换下一个
- **双向心跳**：节点每 10s 发 ping，中间件回 pong；心跳发送连续失败 3 次判定中间件不可用，断开后 3s 重试（注册被拒 30s 重试）
- 收到 `probe` 后节点直调探针函数（带缓存）并回 `probe_result`，结果与 HTTP 通道一致
- 节点注册带 `NODE_KEY`：中间件 `wsKeys` 里配了 key 就必须传对，否则返回 401 并断开；未配置 key 的节点开放注册

**注意**：`NODE_ID` 必须与中间件 `APIBaseURL` / `IPLocationAPI` 池中的节点 `id` 一致，且该节点需配置 `"ws": true` 才会走 WS 通道。未连接中间件时，`ws:true` 节点的拨测会返回 502。
