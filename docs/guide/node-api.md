# 节点 API 与协议参考

后端节点（`src/`）对外暴露的全部接口与协议。配置项说明见 [配置文件](/guide/config)，部署方式见 [后端节点部署](/guide/deploy-node)。

> 本文描述 `src/` 主节点。边缘函数版本（`serverless/edgeone/`）是**精简形态**：只有 `/v1/*` 探测端点与 `GET /`（仅返回 `{"status":"ok"}`，不回报版本与能力），没有 WS 通道、管理接口与 OTA；但同样支持 HTTP 数据上报，见下文「数据上报协议」。

## 鉴权总则

| 通道 | 鉴权方式 | 未配置凭据时 |
|------|----------|--------------|
| 业务探测 `/v1/*` | `access-token` 非空时校验 `Authorization: Bearer <access-token>`，不符返回 `401` | 不启用鉴权，公开可访问 |
| 健康检查 `/`、节点信息 `/info` | 无 | — |
| 管理接口 `/v1/config`、`/v1/ota`（HTTP） | 同业务接口，复用 `tokenCheck` | **整组关闭**，一律返回 `403`（节点未配 `access-token` 时业务接口本身也无鉴权，暴露配置读写等于公网可改配置） |
| WS 通道 | 中间件 `ws-keys` 校验注册报文 | WS **常开**：节点是客户端，指令只来自已通过中间件校验的连接，与节点是否配 `access-token` 无关 |

> 结论：**HTTP 管理面是"有凭据才开"的**；只想用 WS 通道管理节点时，节点无需配置 `access-token`。

---

## 一、健康检查与节点信息

```
GET /      → {"status":"ok"}
GET /info  → {"version":"v1.2.3","capabilities":["probe","report","config","ota"]}
```

两个端点都无鉴权、不计入统计，且都注册在 `/v1` 组之外——因此既不受 `access-token` 约束，也不经过业务统计中间件。

- **健康检查 `/`**：只回答"进程是否活着"。给负载均衡、OTA 就绪探测（`otaWaitChildReady`）与运维 `curl` 用。**不回报版本**：它是免鉴权的对外端点，回版本号等于向公网公开节点版本。
- **节点信息 `/info`**：回报节点身份。`version` 为版本号，`capabilities` 为能力清单（与 WS `register` 报文同源，见 `ws.go nodeCapabilities`）。收集中心对纯 HTTP 节点（无 WS 连接）探活后单独取回，用于节点状态页展示版本、以及下发管理指令前判断该节点能否理解对应指令。

> 老版本节点没有 `/info`（返回 `404`），边缘函数版节点同样不提供；收集中心取不到就留空，不影响判活。

---

## 二、业务探测接口

全部挂在 `/v1` 组下，共用 CORS + 可选 `tokenCheck` + 统计中间件（`nodeReportMiddleware`，见下文「数据上报」）。

| 方法 | 路径 | 参数 | 说明 |
|------|------|------|------|
| GET | `/v1/detail/*url` | 目标 URL（路径传入，自动补全协议） | 网站探测明细：DNS/TCP/HTTP 各阶段耗时、状态码、页面体积、下载速度、可达性；双栈分别返回 |
| GET | `/v1/ssl/*url` | 目标 URL | SSL 证书检查：剩余有效天数、颁发/主体、TLS 版本、是否过期；双栈分别返回 |
| GET | `/v1/speed/:version/*url` | `version` = `v4` / `v6`，目标 URL | 下载测速 |
| GET | `/v1/tcping/:ip` | `port`（默认 80）、`count`（默认 4，上限 20） | TCP 连通性与 RTT 统计 |
| GET | `/v1/dns/:type/*domain` | `type` ∈ `a`/`aaaa`/`cname`/`mx`/`ns`/`ptr`/`srv`/`txt`/`caa` | DNS 记录查询（`dns-server` 支持主从 failover 与 DoH） |
| GET | `/v1/dnssec/:domain` | 域名 | DNSSEC 链式验证（`dnssec-server`，留空沿用 `dns-server`） |
| GET | `/v1/whois/:domain` | 域名 | Whois 注册信息 |
| GET | `/v1/location/:ip` | IP | 多源 IP 归属地聚合（见 [功能总览](/info/features)） |
| GET | `/v1/location` | 无（取请求方 IP） | 查询访问者自身归属地；配置 `trusted-proxies` 后只信任这些代理转发的 `X-Forwarded-For` |
| GET | `/v1/asn/:ip` | IP | ASN 查询（三套 ASN 库 + ASN Whois 解析） |

**条件注册**：`/v1/location/*`、`/v1/asn/:ip` 仅在 `ipdb` 未关闭时注册——IP 库未加载时这些接口无意义。

**单栈模式**：`single-stack` 配为 `ipv4` / `ipv6` 时只跑对应栈，另一栈直接返回 `Skipped due to SINGLE_STACK=...`，不发起无意义的探测与超时。

**SSRF 防护**：`block-private-ips` 开启时，`detail` / `ssl` 对解析到内网/私有地址的目标直接返回伪造结果，出站客户端同时拒绝连接私有 IP、禁止跨跳重定向。

---

## 三、管理接口

### 3.1 运行时配置 `/v1/config`

读写节点**当前生效**的配置（内存值），与收集中心的「托管配置」是两回事——后者是节点启动时拉的远端配置，见 [配置文件](/guide/config) 的「运行时配置管理」一节。

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/v1/config` | 返回当前生效配置快照 + `secretKeys` + `restartRequiredKeys` + `remoteProtectedKeys` |
| PATCH | `/v1/config` | 按下发 map 逐键应用；`?persist=1` 时同时写回本地 `setting.json` |
| POST | `/v1/config/refresh` | 重新拉取 `remote-config-url` 并应用（远端是持久来源，不写回本地文件） |

行为要点：

- **凭据遮蔽**：`access-token` / `node-key` / `report-token` 在 GET 快照里回显 `***`，不泄露明文；下发时原样提交会把真值覆盖成三个星号，因此调用方必须**剔除这些键**（收集中心控制台已自动剔除）。
- **`access-token` / `report-token` 不随远端配置覆盖**：这两项硬编码在保护名单 `configRemoteProtectedKeys` 里，`remote-config-url`（含收集中心托管配置）下发时会被跳过，并在 `refresh` 应答的 `protectedIgnored` 与节点日志里列出被跳过的键。**只约束远端下发**，本地 `PATCH` 照常可改。理由与清单见 [配置文件](/guide/config) 的「远端配置」一节。
- **未持久化的改动会被重启顶掉**：节点侧优先级是 远端 > 环境变量 > `setting.json`，只 PATCH 内存时重启即回到旧值。需要长期生效应 `refresh`（改远端配置）或把改动合并进托管配置——受保护凭据例外，只能落到本地 ENV / `setting.json`。
- **`restartRequired` 只如实回报**：`port` / `cors` / `ipdb` / `report-interval-seconds` / `trusted-proxies` / `node-id` / `node-key` / `access-token` 这些启动期固定的键，改动后节点**不会自行重启**，只在应答里列出键名，重启时机交给运维。
- **`ws-url` 例外，热生效**：WS 客户端控制器按新地址"多退少补"，无需重启。 `node-ota` 同样即时生效：该开关在每次下发 OTA 指令时才读取，不属需重启类键。

### 3.2 OTA 升级 `/v1/ota`

收集中心 HTTP 回退通道（WS 在线时优先走 WS `ota` 消息）。

```
POST /v1/ota
{
  "requestId": "可选，回执关联用",
  "version":   "v1.4.0",                    // 二选一：按版本号下发
  "url":       "https://.../lemonipw-linux-amd64",  // 二选一：直发下载地址
  "assetBase": "https://.../releases/download",     // 可选，配合 version 使用
  "sha256":    "可选，hex64（兼容 sha256: 前缀），提供则强校验"
}
→ 202 {"ok":true,"started":true}
```

响应仅为**受理回执**（下载可能数分钟，HTTP 无进度通道）。执行阶段见 [后端节点部署](/guide/deploy-node) 的「OTA 升级」一节。

---

## 四、WebSocket 协议

节点作为 **WS 客户端**连接收集中心（或独立中间件）的 `/ws` 端点：`ws(s)://<中间件>:<ws-port>/ws`。`ws-url` 支持逗号分隔多个地址，**同时连接全部（多活）**，由 per-URL 连接控制器统一收敛。

### 4.1 消息信封

```json
{ "type": "...", "nodeId": "...", "ts": 1710000000, "data": { } }
```

### 4.2 消息类型

**节点 → 中间件**

| type | data 字段 | 作用 |
|------|-----------|------|
| `register` | `nodeId` / `key`（配了 ws-keys 时必填）/ `version` / `capabilities[]` | 注册接入，建立身份 |
| `ping` | — | 心跳，每 10s |
| `pong` | — | 应答中间件 ping |
| `probe_result` | `requestId` / `status` / `body` | 拨测结果回传 |
| `report` | 统计 + 拨测明细（同 `/report` body） | 周期数据上报 |
| `config_result` | `requestId` / `action` / `ok` / `error` / 配置快照 / `applied` / `unknown` / `restartRequired` / `persisted` | 配置指令应答 |
| `ota_result` | `requestId` / `ok` / `stage` / `error` | OTA 进度与失败原因 |

**中间件 → 节点**

| type | data 字段 | 作用 |
|------|-----------|------|
| `register_ok` | `heartbeatSeconds` | 注册通过，告知心跳周期 |
| `register_error` | `code` / `command` | 注册被拒（如 key 不匹配），30s 后重试 |
| `probe` | `requestId` / `apiType` / `raw` / `query` / `scheduler` | 下发拨测 |
| `config` | `requestId` / `action` / `config` / `persist` | 下发配置指令 |
| `ota` | `requestId` / `url` 或 `version`+`assetBase` / `sha256` | 下发升级任务 |
| `ping` | — | 中间件心跳（节点回 `pong`） |
| `status` | 统计字段 | 中间件状态，节点仅记录日志 |

### 4.3 心跳与重连

- 节点每 **10s** 发 `ping`；发送连续失败 **3 次**判定通道不可用，主动断开并在 **3s** 后重连。
- 注册被拒（key 错误等）时 **30s** 后重试。
- 中间件侧每 20s 发 `ping` + `status`，空闲 **75s** 未收到任何报文即剔除该节点并记离线事件。

### 4.4 OTA 进度阶段

节点经 `ota_result` 依次回报：`accepted` → `downloading` → `verifying` → `installing` → `restarting`；任一阶段失败即终止并带 `error`。重启后连接必然断开，"最终结果"由收集中心以**重连注册上报的新版本号**判定。

---

## 五、数据上报协议

统计与拨测明细**由节点自己上报**，收集中心不在转发路径上采集——节点是唯一记录者，不存在双算。

| 通道 | 条件 | 形式 |
|------|------|------|
| WS `report` 消息 | 已连接中间件 | 向所有在线中间件广播 |
| HTTP `POST <report-url>/report` | WS 未启用或全部掉线 | 携带 `Authorization: Bearer <report-token>` |

```json
{
  "instance": "cn-jiangsu",
  "stats": [
    { "nodeId": "cn-jiangsu", "apiType": "tcping",
      "total": 1, "errors": 0, "latencySumMs": 12, "latencyMaxMs": 12, "minute": 29821676 }
  ],
  "probes": [
    { "nodeId": "cn-jiangsu", "apiType": "tcping", "raw": "qq.com",
      "status": 200, "latencyMs": 33, "body": { } }
  ]
}
```

- 上报周期由 `report-interval-seconds` 控制（缺省 15s）。
- `stats` 是**增量**，收集器按 `(分钟 × 节点 × apiType)` 累加；上报为 at-most-once（失败丢弃并记日志，不重试——重试会双算）。
- 单次报文上限 500 条 stats / 500 条 probes / 4MB；超出或队列（缓冲 2048）打满时丢弃并计数。
- **只上报第一方观测**：节点自己处理的请求才记账，不转播从别处收到的数据。
- `minute` 是 unix 分钟桶（按节点时钟）；上报方传 `0` 表示"由收集器按自身时钟入桶"，收集端另有"不接受未来 1 小时以外"的漂移容忍。
- **部署形态差异**：`serverless/edgeone/` 版本不维持长连接（无 WS 客户端），只走 HTTP `POST /report` 一条通道，配置方式见 [后端节点部署](/guide/deploy-node) 的 EdgeOne 一节。

---

## 六、能力清单（capabilities）

节点在 `register` 报文与 `GET /` 中上报自身支持的管理能力，取值为 `probe` / `report` / `config` / `ota` 的数组。

收集中心据此做**三态判定**，绝不能把"未知"当作"不支持"：

| 上报情况 | 收集中心行为 |
|----------|--------------|
| 清单含该能力 | 正常下发对应指令 |
| 清单明确不含 | 该能力确实没有：立刻拒绝，不发起指令、不等超时 |
| 清单为空（未上报） | 老版本节点，能力**未知**：仍尝试下发；超时后主动探活，区分"节点活着但静默忽略（程序过旧）"与"连接已僵死" |

新增一类管理指令时应在此追加能力标识，并保持取值稳定（收集中心按字符串比对）。

---

## 七、缓存与并发

业务探测结果在节点侧缓存，避免同 key 洪打把上游打崩：

| 缓存 | 键 | TTL |
|------|-----|-----|
| `detail` | 归一化 URL | 5 分钟 |
| `ssl` | URL | 5 分钟 |
| `speed` | `url:version` | 5 分钟 |
| `tcping` | `host:port:count` | 5 分钟 |
| `whois` | 域名 | 5 分钟 |
| `asn whois` | `AS<asn>` | 5 分钟 |
| Bilibili 归属地（在线 API） | IP | 24 小时 |

- **失败结果 30s 后删除**：探测失败（如目标不可达）不缓存整 5 分钟，避免故障恢复后长时间返回旧结论。
- **singleflight 合并**：`detail` / `ssl` / `tcping` / `speed` 的并发同 key 请求合并为一次真实探测。
- **定时清扫**：每 10 分钟清理过期缓存条目与 Bilibili 缓存，防止只靠"同 key 重访才淘汰"时内存无限增长。
