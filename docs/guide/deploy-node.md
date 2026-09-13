# 后端节点部署

后端节点提供全部检测 API（IP 归属地 / ASN / SSL / DNS / DNSSEC / Whois / TCPing / 测速 / 截图等），可在任意地区部署，组成去中心化测试网络。

## 方案一：EdgeOne

腾讯云 EdgeOne 支持 Go 边缘函数，点击下方按钮通过 **EdgeOne Makers** 一键部署 `serverless/edgeone/` 版本（无需本地构建）：

[![使用 EdgeOne Makers 部署](https://cdnstatic.tencentcs.com/edgeone/pages/deploy.svg)](https://console.cloud.tencent.com/edgeone/makers/new?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=serverless%2Fedgeone)


也可以使用 EdgeOne CLI 手动部署：

```bash
cd serverless/edgeone
npx edgeone pages deploy -n ipw-cn -t $EDGEONE_API_TOKEN
```

`serverless/edgeone/` 版本的配置来源为 环境变量 + 远端配置（`REMOTE_CONFIG_URL`）。

### 数据上报（可选）

`serverless/edgeone/` 版本**没有 WS 客户端**（边缘函数不维持长连接），因此上报只走 HTTP 一条路：按周期把统计与拨测明细 `POST <report-url>/report` 到收集中心。下列键同样支持环境变量与远端配置两种来源（远端配置优先）：

| 配置键 | 环境变量 | 说明 |
| --- | --- | --- |
| `node-id` | `NODE_ID` | 上报身份（写入 `probe_results.origin`）；留空回退 hostname |
| `report-url` | `REPORT_URL` | 收集中心基址；留空则不上报 |
| `report-token` | `REPORT_TOKEN` | `/report` 鉴权，请求头 `Authorization: Bearer <token>`；收集中心未配 token 时可留空 |
| `report-interval-seconds` | `REPORT_INTERVAL_SECONDS` | 上报间隔秒，缺省 15 |

记账口径与主节点完全一致（节点是唯一记录者、`stats` 为增量累加、上报 at-most-once），报文格式见 [节点 API 与协议](/guide/node-api) 的「数据上报协议」一节。

> [!NOTE]
> 上报依赖进程存活期内的内存计数。若所用平台会把实例缩容到零，缓冲区会随实例回收丢失——上报只在实例存活期内累积。长驻部署（Docker / 二进制）不受此限。

## 方案二：Vercel

点击下方按钮一键导入仓库（部署目标为 `serverless/edgeone/cloud-functions` 目录，即 Go 边缘函数版本）：

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=serverless%2Fedgeone%2Fcloud-functions)

> [!IMPORTANT]
> **Serverless 节点均为 IPv4-only**（EdgeOne、Vercel 等平台的出站网络只提供 IPv4）。部署后需要：
>
> 1. 给节点配置 `SINGLE_STACK=ipv4`（声明节点为 IPv4 单栈），跳过 IPv6 相关测试，避免无意义的错误日志与超时
> 2. 在前端配置 `config/index.ts` 中，**该节点只能填入 `APIBaseURL.IPv4`**（拨测节点池的 IPv4 栈），不要放进 `DualStack` / `IPv6` 栈

## 方案三：Docker

`src/` 目录提供 `Dockerfile`（多阶段构建，产物基于 Alpine，支持多架构）：

```bash
cd src
docker build -t lemon-ipw .
docker run -d --restart unless-stopped -p 8080:8080 \
  -v $(pwd)/setting.json:/home/appuser/setting.json \
  lemon-ipw
```

> `--restart unless-stopped`：主进程意外退出时 Docker 自动拉起容器；手动 `docker stop` 不会触发重启。

配置通过挂载 `setting.json` 提供（注意挂载到容器工作目录 `/home/appuser`），也可用环境变量覆盖（`PORTS` / `CORS` / `TRUSTED_PROXIES` / `ACCESS_TOKEN` 等）。

> [!TIP]
> 容器内 OTA 替换二进制意义有限（重建容器即回滚到镜像版本），如需彻底关闭可设 `NODE_OTA=false`；镜像**不含 IP 库数据**，首次启动若 `ipdb` 未关闭会现场拉取约 450MB 到容器可写层，轻量部署建议直接 `IPDB=false`。

**多架构**：Dockerfile 已适配 buildx——构建阶段固定在构建机原生平台交叉编译（`TARGETOS`/`TARGETARCH`），目标架构无需 QEMU 模拟。本机为其他架构构建：

```bash
cd src
docker buildx build --platform linux/arm64 -t lemon-ipw:arm64 --load .
```

打 `v*` 版本标签发版时，CI（`.github/workflows/docker.yml`）会自动构建 `linux/amd64` / `linux/arm64` / `linux/arm/v7` 三架构镜像（amd64 在 x64 实例上原生构建，arm64/armv7 在 GitHub 原生 ARM 实例 `ubuntu-24.04-arm` 上交叉构建，全程无 QEMU），推送 GHCR 与 Docker Hub：

- GHCR：`ghcr.io/<owner>/<repo>/lemon-ipw`（自动）
- Docker Hub：`docker.io/<namespace>/lemon-ipw`（**workflow 中暂未启用**——`.github/workflows/docker.yml` 已预留注释，取消注释并配置 `DOCKERHUB_USERNAME` / `DOCKERHUB_TOKEN` Secrets 即可开启）

均带 `版本` 与 `major.minor` 标签。部署直接：

```bash
docker run -d -p 8080:8080 \
  -v $(pwd)/setting.json:/home/appuser/setting.json \
  ghcr.io/nomdn/ipw-cn/lemon-ipw:v1.2.3
```

Docker 会按宿主机架构自动拉取 manifest 中对应的镜像变体。

## 方案四：二进制

```bash
# 编译（Go 后端在 src/ 目录）
cd src && go build -o lemonipw .

# 运行（与 setting.json 同目录）
./lemonipw
```

首次启动自动下载 IP 数据库（约 450MB），之后每 24 小时自动更新一次。需要守护运行时，可参考下方「方案五」生成的 systemd 服务，或手动创建 service 文件（`ExecStart` 指向二进制，`WorkingDirectory` 指向 `setting.json` 所在目录）。

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
| 单栈模式 | 双栈 | `SINGLE_STACK`，`ipv4` / `ipv6` / 留空 |
| access-token | 留空 | 留空 = 不启用鉴权 |
| DNS 服务器 | 留空 | 主从逗号分隔（`119.28.28.28:53,223.5.5.5:53`）；留空 = 启动时自动探测系统 DNS |
| DNSSEC 专用 DNS | 留空 | 留空 = 沿用 dns-server；都留空 = 自动探测系统 DNS |
| IP 数据库 | `Y` | `IPDB`，首次启动下载约 450MB |
| CORS | 留空 | 逗号分隔允许来源 |
| 远端配置地址 | 留空 | `REMOTE_CONFIG_URL` |
| WS 通道接入 | `N` | 选 `y` 后继续输入 WS_URL / NODE_ID / NODE_KEY |
| 其他环境变量 | 无 | 每行一个 `K=V`，空行结束。如需可信代理在此输入 `TRUSTED_PROXIES=<IP/CIDR 逗号分隔>`（配置后归属地接口只信任这些代理转发的 X-Forwarded-For，防伪造） |

**WS 接入特殊规则**：

- `WS_URL` 留空时自动使用默认值 `wss://middleware-1.api-ipw.wsmdn.top/ws`；填写自己的中间件地址时须为**完整路径**（含 `wss://` 前缀与 `/ws` 路径，如 `wss://host:8092/ws`），逗号分隔多个则**同时连接全部（多活）**
- `NODE_ID` **强制自动生成 UUID**（无需输入，如 `4580ea9d-2a16-4b8e-8090-5f2e7f0a2229`），适合作为节点唯一标识
- `NODE_KEY` **必填**：不加 key 禁止启用 WS（中间件 `ws-keys` 必须包含该节点，否则注册被拒 401）；安装完成后脚本会提示把 `"<NODE_ID>": "<NODE_KEY>"` 加入中间件 setting.json 的 `ws-keys`

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

**配置三个环境变量**（或 setting.json 的 `ws-url` / `node-id` / `node-key`，env 优先；也支持远端配置覆盖，除非列入 `remote-ignore-config`）：

```bash
WS_URL=ws://<中间件IP>:8092/ws \
NODE_ID=<节点id，与中间件 api-base-url / ip-location-api 池的节点 id 一致> \
NODE_KEY=<注册key，与中间件 ws-keys[节点id] 一致；节点未配置 key 可留空> \
./lemonipw
```

- `WS_URL` 支持**逗号分隔多个中间件**（`ws://中间件1:8092/ws,ws://中间件2:8092/ws`），**同时连接全部（多活）**：每个地址独立注册并保持连接，任一断开只重连自己，不影响其他中间件；注意多个地址指向同一中间件实例时，同 `nodeId` 后注册的连接会顶掉先前的
- **双向心跳**：节点每 10s 发 ping，中间件回 pong；心跳发送连续失败 3 次判定中间件不可用，断开后 3s 重试（注册被拒 30s 重试）
- 收到 `probe` 后节点直调探针函数（带缓存）并回 `probe_result`，结果与 HTTP 通道一致
- 节点注册带 `NODE_KEY`：中间件 `ws-keys` 里配了 key 就必须传对，否则返回 401 并断开；未配置 key 的节点开放注册

**注意**：`NODE_ID` 必须与中间件 `APIBaseURL` / `IPLocationAPI` 池中的节点 `id` 一致，且该节点需配置 `"ws": true` 才会走 WS 通道。未连接中间件时，`ws:true` 节点的拨测会返回 502。

## 运行时配置管理（可选）

节点暴露 `/v1/config`，供收集中心读写该节点**当前生效**的配置（内存值），是「控制台改配置不用登机器」的基础：

| 动作 | HTTP | WS | 说明 |
|------|------|----|------|
| 读取 | `GET /v1/config` | `config` + `action=get` | 返回生效配置快照 + `secretKeys` + `restartRequiredKeys` |
| 修改 | `PATCH /v1/config`（`?persist=1` 写回本地文件） | `config` + `action=patch` | 只应用传入的键，其余不动 |
| 刷新远端 | `POST /v1/config/refresh` | `config` + `action=refresh` | 重新拉 `remote-config-url` 并应用 |

三条重要约定：

- **凭据类键不回显明文**：`access-token` / `node-key` / `report-token` 在快照中为 `***`。调用方下发配置时必须剔除这些键，否则会把真值覆盖成三个星号。收集中心控制台已自动剔除。
- **节点不自行重启**：`port` / `cors` / `ipdb` / `report-interval-seconds` / `trusted-proxies` / `node-id` / `node-key` / `access-token` 这些启动期固定的键，改动后在应答的 `restartRequired` 中如实列出，但**不会自动重启进程**——重启会中断在途服务，时机由运维 / 编排层掌握。`ws-url` 是例外：热生效，控制器按新地址多退少补。
- **只 PATCH 内存的改动会被重启顶掉**：节点侧优先级是 远端 > 环境变量 > `setting.json`，长期生效需写回托管配置或本地文件，详见 [配置文件](/guide/config) 的「运行时配置管理」一节。

> [!NOTE]
> **HTTP 管理面需要凭据**：节点未配置 `access-token` 时，`/v1/config` 与 `/v1/ota` 整组返回 `403`（此时业务接口也无鉴权，开放配置读写等于公网可改配置）。**WS 通道不受此限制**——节点是 WS 客户端，指令只来自已通过中间件 `ws-keys` 校验的连接。

## OTA 升级（收集中心下发，可选）

节点支持由收集中心下发**一次性升级任务**：下载新二进制 → 校验 → 预检 → 原子替换 → 重启，全程无需登录节点主机。

**触发方式**：收集中心经 WS `ota` 消息下发（HTTP 回退 `POST /v1/ota`）。请求体三选一指定下载源：

| 字段 | 说明 |
|------|------|
| `version` | 按版本号下发。节点按自身平台拼资产名 `lemonipw-{goos}-{goarch}[.exe]`，到资产基址 `{assetBase}/{tag}/` 下取；`assetBase` 缺省为 GitHub Releases，自建镜像源可覆盖 |
| `url` | 直发下载地址（完全自定义） |
| `sha256` | 可选，提供即强校验（hex64，兼容 `sha256:` 前缀） |

**执行流程**：下载到与目标同分区的临时文件（保证 `rename` 原子）→ 体积下限 1MB（防错误页）→ sha256 校验 → **预检**（试运行新二进制 `-v`，确认架构匹配、可执行，在停机前拦下损坏文件）→ 当前二进制改名 `.old` → 新文件就位 → 重启。任一步失败都会回滚 `.old`。

**进度回传**：经 WS `ota_result` 依次回报 `accepted` → `downloading` → `verifying` → `installing` → `restarting`，失败时带 `error`。重启后连接必然断开，**最终结果由收集中心以重连注册上报的新版本号判定**（15 分钟未观测到新版本即判超时失败）。

**并发与开关**：

- 同一时刻只允许一个 OTA 任务，重复下发直接拒绝。
- `node-ota: false`（env `NODE_OTA`）可完全禁用：节点拒绝一切 OTA 指令并回传原因。**只读文件系统 / 编排托管**（Docker Swarm、K8s 等自行管理镜像）的部署建议关闭。
- 部署方式受限导致替换失败（如容器只读层）时，升级会失败并回滚，此时仍按传统方式人工升级。

手动更新（OTA 不可用时）：

```bash
# Docker：拉取新镜像后重建容器
docker pull lemon-ipw && docker stop lemon-ipw && docker run ... lemon-ipw
# systemd：替换二进制后重启服务
systemctl restart lemon-ipw
```

## 版本与能力上报

节点无论是否接入 WS，都会上报自身版本与支持的管理能力：

- **版本号**来自构建时注入的 `VERSION`，随 WS `register` 报文上报（字段 `version`）；纯 HTTP 节点（无 WS 连接）由收集中心探活 `GET /` 时一并取回（响应体含 `version`）。
- **能力清单**（`capabilities`）取值为 `probe` / `report` / `config` / `ota`，用于让收集中心在下发管理指令前判断该节点能否理解——老版本节点的消息循环没有对应分支，收到管理消息会**静默忽略**，只能靠超时暴露。
- 收集中心「节点状态」页展示每个节点的版本号，并据此判断哪些节点需要升级；节点升级部署后重新注册即为新版本。

能力清单的取值与三态判定规则见 [节点 API 与协议参考](/guide/node-api) 的「能力清单」一节。
