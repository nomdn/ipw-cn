# 中间件部署

独立中间件（middleware-go）只负责**请求转发和 Key 注入**，不参与业务计算。前端将 `/middleware/*` 请求转发到中间件，由中间件转发到对应后端节点并注入 `Authorization: Bearer <key>`。

> [!IMPORTANT]
> 项目只提供配置示例，**使用前必须先复制为 `setting.json`**（GitHub Releases 中对应文件为 `middleware.setting.json.example`）：
>
> ```bash
> # 仓库内
> cp middleware-go/setting.json.example middleware-go/setting.json
>
> # 从 Release 下载后
> cp middleware.setting.json.example setting.json
> ```
>
> 复制后按需修改 `api-base-url` / `ip-location-api` / `api-keys` / `ws-keys` / `cors` 等配置项。

## 方案一：Docker

`middleware-go/` 目录提供 `Dockerfile`（多阶段构建，产物基于 Alpine，支持多架构）：

```bash
cd middleware-go
docker build -t middleware-go .
docker run -d -p 8091:8091 \
  -v $(pwd)/setting.json:/home/appuser/setting.json \
  middleware-go
```

> 配置挂载注意：容器工作目录是 `/home/appuser`（非 root 用户运行），`setting.json` 要挂到这个路径下。

**多架构**：Dockerfile 已适配 buildx 交叉编译，本机构建其他架构：

```bash
cd middleware-go
docker buildx build --platform linux/arm64 -t middleware-go:arm64 --load .
```

打 `v*` 版本标签发版时，CI 会自动构建 `linux/amd64` / `linux/arm64` / `linux/arm/v7` 三架构镜像（amd64 原生构建，arm64/armv7 在 GitHub 原生 ARM 实例上交叉构建，无 QEMU），推送 GHCR（`ghcr.io/<owner>/<repo>/lemon-ipw-middleware`；Docker Hub 推送在 workflow 中暂未启用，预留了注释可随时开启），Docker 按宿主机架构自动拉取对应变体。

配置通过挂载 `setting.json` 提供（`api-base-url` / `ip-location-api` / `api-keys` / `ws-keys` / `cors` / `trusted-proxies` / `rate-limit` / `remote-config-url` 等；`rate-limit` 为单 IP 每分钟限流次数，默认 120，0 表示不限流，可用环境变量 `RATE_LIMIT` 覆盖；`remote-config-url` 为远端配置地址，可用环境变量 `REMOTE_CONFIG_URL` 覆盖；**`api-keys` / `ws-keys` 为敏感凭据，不随远端配置覆盖**——`api-keys` 管 HTTP 转发鉴权，`ws-keys` 管 WS 注册校验，两者相互独立）。

## 方案二：二进制

```bash
cd middleware-go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o middleware-go-linux-amd64 .
./middleware-go-linux-amd64          # 运行（与 setting.json 同目录）
```

**OTA 自更新**（可选，默认关闭）：配置 `"ota": "true"`（或环境变量 `OTA=true`）后，中间件定期检查本仓库 Release，下载与当前平台匹配的 `middleware-go-*` 资产并替换自身重启。下载加速可配 `"gh-proxy": ""`（或 `GH_PROXY` 环境变量）。交接流程与后端节点一致：预检 → 原子替换 → 优雅停机 → 拉起新进程 → 健康检查确认；失败自动回滚 `.old` 备份。

所有配置可用环境变量覆盖（`API_BASE_URLS` / `IP_LOCATION_APIS` / `CORS` / `TRUSTED_PROXIES` / `API_KEYS` / `WS_KEYS` / `RATE_LIMIT` 等，数组/对象用 JSON 字符串），优先级：**环境变量 > setting.json > 默认值**（远端配置 `REMOTE_CONFIG_URL` 高于两者，详见 [配置文件 - 远端配置](/guide/config#远端配置remoteconfigurl)）。

需要守护运行时，参考 [后端节点部署 - 方案五：一键安装](/guide/deploy-node#方案五一键安装installsh) 的 systemd 管理方式（`ExecStart` 指向中间件二进制，`WorkingDirectory` 指向 `middleware-go/setting.json` 所在目录）。

## WS 通道（拨测数据经 WebSocket 传输）

middleware-go 内置 WS 服务端，后端节点可作为 **WS 客户端**连入，拨测数据上传改用 WS 通道传输（HTTP 转发保留，双通道并存）。**现有 HTTP 接口（`/v1/*` / `/middleware/*`）完全不变**。

### 架构

```
前端（HTTP） → middleware-go（8091，HTTP 转发）→ 后端节点（ws:false 时）
              ↘ middleware-go（8092，WS 服务端）→ 后端节点（ws:true 时，WS 客户端连入）
```

- WS 服务端跑在**独立端口**（配置 `ws-port`，默认 `8092`，路径 `/ws`；`ws-port: 0` 或环境变量 `WS_PORT=0` 关闭），与 fiber HTTP 服务完全分离；端口走统一配置管道（远端配置 > 环境变量 > setting.json）
- 节点配置加 `"ws": true`（兼容 `"ws": "true"` 字符串写法）→ 该节点的拨测请求改走 WS 通道；缺省 `false` → 原 HTTP 转发不变
- ws:true 节点未连接或拨测超时 → 返回 502，不影响其他节点

### 配置示例

```json
{
    "port": "8091",
    "APIBaseURL": {
        "DualStack": [ { "label": "上海 腾讯云 BGP", "id": "tencent-sh", "url": "", "ws": true } ],
        "IPv4": [],
        "IPv6": [ { "label": "河北 秦皇岛 联通", "id": "cn-hebei-qinhuangdao", "url": "https://cn-hebei-qinhuangdao.api-ipw.wsmdn.top/", "ws": true } ]
    }
}
```

启动后日志会打印 `ws channel enabled on port 8092`。

### 消息协议

所有消息为 JSON 文本帧，信封：`{ "type": "...", "nodeId": "...", "ts": <unix秒>, "data": {...} }`

| 方向 | type | data 说明 |
|------|------|-----------|
| 节点 → middleware | `register` | `{ "nodeId": "...", "key": "..." }`，连接后首条消息，同 id 新连接顶掉旧连接；`key` 为注册凭证，与 setting.json `ws-keys` 配置比对 |
| middleware → 节点 | `register_ok` | `{ "heartbeatSeconds": 20 }` |
| middleware → 节点 | `register_error` | `{ "code": 401, "command": "invalid key" }`（注册凭证错误时返回并断开连接） |
| middleware → 节点 | `probe` | `{ "requestId": "...", "apiType": "tcping", "raw": "qq.com", "query": {"port":"443"} }`，拨测请求（HTTP/WS 双通道时前端请求仍走 HTTP，middleware 内部转 WS） |
| 节点 → middleware | `probe_result` | `{ "requestId": "...", "status": 200, "body": <JSON 值> }`（body 为 JSON 字符串时按原文透传） |
| middleware → 节点 | `ping` | 心跳，节点回 `pong` |
| middleware → 节点 | `status` | 状态/统计上报：`{ "uptimeSeconds": ..., "totalRequests": ..., "errorRequests": ... }`（每 20s） |
| 节点 → middleware | `command` | 指令下发，如 `{ "command": "reload_config" }`（重读配置；与并发请求存在竞态，生产建议重启生效） |

### 节点侧接入（WS 客户端）

1. 连接 `ws://<中间件IP>:8092/ws`（本仓库后端节点接入方式见 [后端节点部署 - WS 通道接入](/guide/deploy-node#ws-通道接入可选)，配置 `WS_URL` / `NODE_ID` / `NODE_KEY` 即可自动连接注册；`WS_URL` 支持逗号分隔多个中间件地址，节点会**同时连接全部（多活）**，任一断开只重连自己）
2. 首条消息发 `register { "nodeId": "<与中间件配置一致的节点 id>", "key": "<注册凭证>" }`（节点在 setting.json `ws-keys` 里配了 key 就必须传对；未配置 key 的开放节点可不传）
3. 收到 `probe` → 执行拨测 → 回 `probe_result`（必须携带原 `requestId`，支持乱序返回，同一连接可并发多个拨测）
4. 收到 `ping` → 回 `pong`；`status` 可忽略或记录
5. 断线后重连并重新 `register` 即可

### 注册 key 校验

节点注册时中间件按 `ws-keys` 配置校验 key（与 `api-keys` HTTP 转发鉴权相互独立）：

- 节点在 `ws-keys` 配置了 key（非空）→ `register` 必须携带正确 key，不传或传错返回 `register_error { "code": 401, "command": "invalid key" }` 并**断开连接**，不进入数据阶段
- 节点未配置 key（`ws-keys` 无该项或为空）→ 开放注册，不传 key 即可

与 HTTP 通道的鉴权语义一致（配置了 key 就必须带凭据，留空 = 开放）。

### 说明

- 注册阶段按节点 `ws-keys` 配置校验 key（配了 key 必须传对，否则 401 断开；未配置则开放，详见上文「注册 key 校验」）；拨测请求经 WS 转发时不携带 `Authorization` 头（后续由节点侧自行决定是否需要额外鉴权）
- `ws:true` 且 WS 未启用（`WS_PORT=0`）时，自动回退原 HTTP 转发
