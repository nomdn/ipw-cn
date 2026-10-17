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
| GET | `/v1/tcping/:ip` | `port`（默认 80）、`count`（默认 4，上限 20）、`version`（见下文「请求级参数」） | TCP 连通性与 RTT 统计 |
| GET | `/v1/dns/:type/*domain` | `type` ∈ `a`/`aaaa`/`cname`/`mx`/`ns`/`ptr`/`srv`/`txt`/`caa` | DNS 记录查询（`dns-server` 支持主从 failover 与 DoH） |
| GET | `/v1/dnssec/:domain` | 域名 | DNSSEC 链式验证（`dnssec-server`，留空沿用 `dns-server`） |
| GET | `/v1/whois/:domain` | 域名 | Whois 注册信息 |
| GET | `/v1/location/:ip` | IP | 多源 IP 归属地聚合（见 [功能总览](/info/features)） |
| GET | `/v1/location` | 无（取请求方 IP） | 查询访问者自身归属地；配置 `trusted-proxies` 后只信任这些代理转发的 `X-Forwarded-For` |
| GET | `/v1/asn/:ip` | IP | ASN 查询（三套 ASN 库 + ASN Whois 解析） |

**条件注册**：`/v1/location/*`、`/v1/asn/:ip` 仅在 `ipdb` 未关闭时注册——IP 库未加载时这些接口无意义。

**单栈模式**：`single-stack` 配为 `ipv4` / `ipv6` 时只跑对应栈，另一栈直接返回 `Skipped due to SINGLE_STACK=...`，不发起无意义的探测与超时。

**SSRF 防护**：`block-private-ips` 开启时，`detail` / `ssl` 对解析到内网/私有地址的目标直接返回伪造结果，出站客户端同时拒绝连接私有 IP、禁止跨跳重定向。`tcping` 是裸 TCP、不经过出站客户端，故在 `webtest.ResolveHostWith` 内单独把关：**字面 IP**、**DNS 解析结果**、**`resolve*` 改写后的目标**三者只要落在私网/回环/链路本地/组播网段一律拒绝，该栈返回 `Error: connection to private/internal address <IP> is not allowed`（`sent` 仍为请求次数、`success=0`、`loss_rate=100`）；关闭开关时三者一并放行。

### 请求级参数

除路径参数与常规 query 外，`/v1/tcping/:ip` 与所有走 DNS 的探测接口还接受一组**请求级参数**，用于按次覆盖解析行为。参数统一由 `webtest.ParseReqResolve` 解析，HTTP 拨测经 context、`tcping` 经形参透传到解析层；HTTP 与 WS 两条通道语义一致。

| 参数 | 取值 | 作用接口 | 说明 |
|------|------|----------|------|
| `dns` | 逗号分隔的 `ip:port` 或 `http(s)://` DoH URL | `detail` / `ssl` / `speed` / `tcping` / `whois` / `dns` | 本次探测改用指定 DNS 服务器，取代配置的 `dns-server` |
| `dns4` | 同上 | 同上 | 只覆盖 IPv4 族解析，**优先于 `dns`** |
| `dns6` | 同上 | 同上 | 只覆盖 IPv6 族解析，**优先于 `dns`** |
| `resolve` | 逗号分隔的 `域名:目标`，目标为 IP 或域名 | `detail` / `ssl` / `speed` / `tcping` | 强制改写某域名的解析结果（两栈通用） |
| `resolve4` | 同上 | 同上 | 只影响 IPv4 侧；非空时 IPv4 侧**只用这张表**（与 `dns4` 同口径） |
| `resolve6` | 同上 | 同上 | 只影响 IPv6 侧；非空时 IPv6 侧**只用这张表** |
| `version` | `ipv4` / `ipv6`（兼容 `v4` / `v6`，大小写与首尾空格不敏感） | `tcping` | 请求级指定单栈 |

`dns*` 与 `resolve*` 是两类正交的覆盖：前者「换服务器去问」，后者「直接改答案」，可同时使用。

**`resolve*` 的目标可以是 IP，也可以是域名**：

- 目标是 **IP**（`resolve4=example.com:1.2.3.4`，IPv6 可写 `[::1]` 或裸地址）→ 拨号阶段不再查 DNS，直接对该 IP 做私网校验后连接；
- 目标是 **域名**（`resolve=www.example.com:origin.internal`）→ 把它当作新的 host 继续走解析，解析服务器仍取 `dns*`/配置，且**不再查映射表**（因此不会绕成环）。

行为要点：

- **取值与配置侧同一套校验**（`NormalizeServers`）：合法项为 `ip:port`（UDP）或 DoH URL，非法项被丢弃；**参数存在但所有项都非法时返回 `400`**，不静默忽略——避免「看起来指定了、实际按默认跑」。`resolve*` 同理：每组必须形如 `域名:目标`，目标为 IP 或形如域名（不含端口、空格），非法项丢弃、全非法返回 `400`。
- **`version` 不越权单栈配置**：节点 `single-stack` 非空时一律以配置为准（单栈机指定另一栈也跑不起来）；仅当配置为空（双栈机）时 `version` 才生效。跳过文案借此区分来源：配置造成记 `SINGLE_STACK`，参数造成记 `version`。
- **作用止于「域名 → IP」这一步**：`dns*` 只更换解析服务器、`resolve*` 只改写解析结果，都不改变探测目标与端口；`dnssec` / `location` / `asn` 三类接口不接受上述参数，`whois` / `dns` 只取 `dns*`、忽略 `resolve*`（`dns` 拨测问的就是真实解析，改写会自相矛盾）。
- **SSRF 预检同源**：开启 `block-private-ips` 时，预检校验的是**拨号真正会连的地址**（即 `resolve*` 改写后的目标），不会拿原域名的解析结果去判定。`tcping` 没有独立的预检步骤，同一判断落在 `ResolveHostWith` 进入拨号之前，口径与上述一致。
- **带覆盖时按独立缓存键缓存**：规范化后的参数会拼进缓存键（见「七、缓存与并发」），不同服务器/不同映射的结果各占一条，既不串味、也不牺牲命中率。

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

节点在 `register` 报文与主节点的 `GET /info` 中上报自身支持的管理能力，取值为 `probe` / `report` / `config` / `ota` 的数组。边缘函数版本不提供 `/info`，也不回报能力清单。

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

| 缓存 | 键（末尾拼请求级解析片段） | TTL（成功 / 失败） |
|------|-----|--------------------|
| `detail` | 归一化 URL | 30s / 5s |
| `ssl` | URL | 30s / 5s |
| `speed` | `url:version` | 30s / 5s |
| `tcping` | `host:port:count:stack` | 30s / 5s |
| `whois` | 域名 | 30s / 5s |
| `asn whois` | `AS<asn>` | 30s / 5s |
| Bilibili 归属地（在线 API） | IP | 24 小时（不分成败） |

- **TTL 按结果分档**（`cacheTTL`）：成功 30s、失败 5s。失败结果不缓存太久，避免故障恢复后长时间返回旧结论；TTL 记在条目里、读时判断，无需后台协程唤醒。
- **缓存键含生效栈**：`tcping` 的键为 `host:port:count:stack`，`version=ipv6` 的结果与默认双栈不会互相污染。
- **singleflight 合并**：`detail` / `ssl` / `tcping` / `speed` 的并发同 key 请求合并为一次真实探测（合并键同缓存键）。
- **请求级解析参数进键**：`dns*` 与 `resolve*` 一并拼入，详见下一节。
- **定时清扫**：每 1 分钟清理过期缓存条目，防止只靠「同 key 重访才淘汰」时内存无限增长。

### 7.1 请求级解析参数如何进入缓存键

缓存键的基础部分只含**目标**维度（URL / host），但探测结果依赖**怎么解析**——同一域名经不同服务器、或经 `resolve*` 改写后可能指向完全不同的 IP（企业内网、CDN 调度、分线路权威、上线前的强制指向）。若把两者混在一条键里，会双向串味且无从察觉：

- 带 `dns=1.1.1.1:53` 的结果写进 `example.com` 这条 → 随后的普通请求（用配置的 `dns-server`）读出来，用户以为是自己默认 DNS 的结果；
- 反过来，普通请求的结果也会被带 `dns=` 的请求读走——用户明明指定了服务器，却拿到别人用另一台服务器查的结果；
- `resolve*` 更直接：强制指向 A 与强制指向 B 是两次不同探测，共用一条键等于互相覆盖。

因此把**规范化后的参数片段拼在键末尾**（`ReqResolve.CacheKey()`），两件事同时成立：不同解析配置各占一条，相同配置仍能命中。

```
https://example.com                                      ← 无请求级参数
https://example.com;dns=1.1.1.1:53                       ← dns=
https://example.com:v4;dns=8.8.8.8:53;dns6=[2400::1]:53   ← speed，按栈覆盖
qq.com:80:4:ipv4;dns=1.1.1.1:53                           ← tcping
https://example.com;resolve4=example.com:127.0.0.1        ← resolve4，强制解析
```

片段规则：

- **字段顺序固定**为 `dns` → `dns4` → `dns6` → `resolve` → `resolve4` → `resolve6`。所以参数在 URL 里谁先谁后不影响键（`?dns6=…&dns=…` 与 `?dns=…&dns6=…` 是同一个键）。
- **`dns*` 内部按输入顺序，`resolve*` 内部按域名排序**：映射表本质是 map，迭代顺序随机；不排序的话同一请求每次算出的键都不同，缓存永远命不中。
- **无参数时片段为空串** ⇒ 键与未引入该特性时**逐字节一致**，升级后旧缓存条目照常命中。
- **按规范化结果拼**：非法项在校验阶段已被丢弃，故 `dns=1.1.1.1:53,garbage` 与 `dns=1.1.1.1:53` 落在同一个键上；`resolve4=Example.COM.:1.2.3.4` 与 `resolve4=example.com:1.2.3.4` 同理（域名统一小写、去尾点）。
- **键空间随组合增长**：每条仍受 30s / 5s 的 TTL 约束，由每分钟的清扫兜住，不会无限堆积。
- `asn` / `location` / `dnssec` 不接这些参数，键与以前相同。

> 已知残留（与本次改动无关）：`detail` 与 `ssl` 的 singleflight 合并键都是「URL + 片段」，两种探测**并发**打同一 URL 时会撞在一起，后到者按另一类型断言会 panic（被 `recover` 接住 → 500）。修法是给合并键加探测类型前缀，暂未处理。
