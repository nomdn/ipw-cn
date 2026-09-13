# 公开 API（/api/v1）

收集中心（`ipw-boce` 中间件）对外提供的**程序化接口**，资源按 REST 规范组织：名词复数、裸资源 + HTTP 状态码、无信封。

- **面向任何持有凭据的调用方**：控制台用户生成一枚个人 API Token 即可用，适合脚本、巡检、看板、CI 取数。
- 与内部 `/admin/*` 共用**同一套鉴权与可见性**，只是对外形态更规范。
- 拨测**节点**自身的接口不在这里，见 [节点 API 与协议](/guide/node-api)。

> 下文 `<collector>` 指收集中心实例的 origin，例如 `https://collector.example.com`。

## 端点一览

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/api/v1/tasks` | 任务列表（分页 + 标签过滤） |
| GET | `/api/v1/tasks/:id` | 单任务详情 |
| GET | `/api/v1/tasks/:id/sla` | 任务 SLA 聚合（顶层汇总 + 按节点明细） |
| GET | `/api/v1/probes` | 拨测明细（分页 + 过滤） |
| POST | `/api/v1/probes` | 一键拨测（同步聚合；本层唯一的写操作） |
| GET | `/api/v1/nodes` | 节点池简表（脱敏） |
| GET | `/api/v1/usage` | 用量聚合 |

---

## 快速开始

```bash
# 1) 取一枚个人 API Token（控制台「个人资料 → 个人 API Token」同效）
curl -sX POST "https://<collector>/admin/me/token" -H "Authorization: Bearer <你的 JWT>"
# → {"token":"ipt_9f3c...e1a7","hint":"e1a7","createdAt":"2026-09-13T07:00:00Z"}

# 2) 调用公开 API
curl -s -H "Authorization: Bearer ipt_9f3c...e1a7" \
  "https://<collector>/api/v1/tasks?page=1&pageSize=20"
```

- Token **明文只在生成时返回一次**（`token` 字段）；库里只存 bcrypt 哈希，`hint` 是末 4 位供列表辨识。
- 再次 `POST /admin/me/token` 会**立即作废旧 Token**；`DELETE /admin/me/token` 吊销。
- 静态 `admin-token` 没有账号，不能生成个人 Token（返回 400）。

---

## 鉴权

三轨，与 `/admin/*` 完全一致：

| 凭据 | 特征 | 身份 | 适用场景 |
|------|------|------|----------|
| 个人 API Token | `ipt_` 前缀 | 其账号（角色随账号） | 程序化访问的推荐方式，可随时吊销 |
| JWT | 无前缀 | 其账号（角色随账号） | `POST /admin/login` 签发，有过期时间 |
| 静态 `admin-token` | 无前缀 | 恒为 admin（`uid=0`） | 向后兼容；无归属，脚本慎用 |

请求头固定 `Authorization: Bearer <凭据>`。失败一律 `401`：

| message | 场景 |
|---------|------|
| `missing bearer token` | 未携带 `Authorization: Bearer ` 前缀 |
| `invalid or revoked token` | `ipt_` 开头但库中查不到（未生成 / 已吊销 / 被轮换） |
| `invalid credentials` | 其它凭据无效 |

> **API Token 等价于其账号登录**：权限随账号角色。普通用户只看得见自己的资源。
> 若收集中心既没配 `admin-token` 也没启用 JWT，则管理面**不鉴权**（仅限内网/联调部署，勿用于公网）。

---

## 通用约定

### 错误体

非 2xx 统一为：

```json
{ "error": { "code": "not_found", "message": "task not found" } }
```

| code | HTTP | 含义 |
|------|------|------|
| `unauthorized` | 401 | 缺凭据 / 凭据无效或已吊销 |
| `forbidden` | 403 | 凭据有效但无权访问该资源（如别人的任务） |
| `not_found` | 404 | 资源不存在（任务、分享令牌） |
| `bad_request` | 400 | 参数非法 |
| `rate_limited` | 429 | 请求过于频繁（仅免登录的公开状态接口） |
| `internal` | 500 | 服务端错误 |

### 分页

两套参数，**互不通用**：

| 端点 | 参数 | 缺省 | 边界处理 | 返回字段 |
|------|------|------|----------|----------|
| `/tasks` | `page`（1 起）、`pageSize`（别名 `limit`） | 1 / 50 | `page<1` → 1；`pageSize` 夹在 1~200 | `{items,total,page,pageSize}` |
| `/probes` | `limit`、`offset` | 50 / 0 | `limit` 夹在 1~500；`offset<0` → 0 | `{items,total,limit,offset}` |

越界值是**夹到边界**而非报错（`pageSize=9999` → 200，`page=0` → 1）。`total` 为符合过滤条件的总数，与当前页无关。两个列表均按 `id` 倒序（最新在前）。

### 可见性

| 角色 | 任务 | 拨测明细 |
|------|------|----------|
| admin / 静态 token | 全量 | 全量 |
| 普通用户 | 仅 `ownerId` = 本人 | 定时样本按「本人任务」归属、业务拨测按发起者 |

访问**单个**他人的任务返回 `403`（而非 404），便于区分"不存在"与"不是你的"。

### 限流

`/api/v1` **不挂限流**——限流只作用于转发口 `/v1` 与 `/middleware`。唯一例外是免登录的公开状态接口（见文末）。

---

## 一、任务

### 1.1 `GET /api/v1/tasks`

| 参数 | 必填 | 说明 |
|------|------|------|
| `page` | 否 | 页码，1 起，缺省 1 |
| `pageSize` | 否 | 每页条数，缺省 50，上限 200（`limit` 为别名） |
| `tag` | 否 | 按标签**子串**过滤（标签以逗号分隔存储） |

```json
{
  "items": [
    {
      "id": 5, "name": "官网可用性", "ownerId": 2, "ownerUsername": "alice",
      "enabled": true, "apiType": "detail", "target": "example.com", "stack": "",
      "recordType": "", "nodeScope": "all", "nodeIds": "", "intervalSec": 60,
      "slowMs": 800, "expectStatus": "2xx", "bothProtocols": false,
      "requireAllStacks": true, "certExpiredDown": false, "notifyRecover": true,
      "quietHours": "23:00-07:00", "hideTarget": false, "tags": "prod,官网",
      "updatedAt": "2026-09-13T06:12:00Z"
    }
  ],
  "total": 1, "page": 1, "pageSize": 50
}
```

任务对象字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | number | 任务 id |
| `name` | string | 任务名 |
| `ownerId` | number | 所有者用户 id；`0` = 无归属（静态 token 建的旧任务） |
| `ownerUsername` | string | 派生字段：所有者用户名（无归属为空串） |
| `enabled` | bool | 是否启用调度 |
| `apiType` | string | `tcping` / `speed` / `ssl` / `detail` / `dns` |
| `target` | string | `detail`/`ssl`/`dns` = 域名；`tcping` = `host[:port]`；`speed` = `v4/`或`v6/`+URL |
| `stack` | string | `""`(按类型) / `v4` / `v6`，仅 `speed` 想固定单栈时用 |
| `recordType` | string | `dns` 专用：`a`/`aaaa`/`cname`/`mx`/`ns`/`ptr`/`srv`/`txt`/`caa`（缺省 `a`） |
| `nodeScope` | string | `all`（全池）/ `custom`（指定） |
| `nodeIds` | string | `custom` 时的逗号分隔节点 id；`all` 忽略 |
| `intervalSec` | number | 调度间隔（秒），最小 10 |
| `slowMs` | number | 慢阈值（ms），超过计一次"未达标"；0 = 不启用 |
| `expectStatus` | string | 期望状态码：`"2xx"` 或 `"200"` / `"200,301,302"` |
| `bothProtocols` | bool | `detail`：http 与 https 都要命中才算成功 |
| `requireAllStacks` | bool | 双栈全通才算该节点可用 |
| `certExpiredDown` | bool | `ssl`：证书过期视为不可用 |
| `notifyRecover` | bool | 从 down 恢复时也通知一次 |
| `quietHours` | string | 免打扰时段 `"HH:MM-HH:MM"`（服务器本地时区，支持跨零点） |
| `hideTarget` | bool | 公开状态页隐藏探测目标 |
| `tags` | string | 逗号分隔标签 |
| `updatedAt` | string | 最后更新时间（RFC3339） |

> 分享令牌 `shareToken` **不在 JSON 里**（`json:"-"`），只经管理接口读写。

### 1.2 `GET /api/v1/tasks/:id`

返回单个任务对象（字段同 1.1）。

| 状态 | 场景 |
|------|------|
| `404 not_found` | `task not found` |
| `403 forbidden` | `not your task`（普通用户访问他人任务） |

### 1.3 `GET /api/v1/tasks/:id/sla?hours=24`

| 参数 | 必填 | 说明 |
|------|------|------|
| `hours` | 否 | 统计窗口（小时），缺省 24，夹在 1 ~ 2160（90 天） |

```json
{
  "taskId": 5, "hours": 24,
  "window": { "from": "2026-09-12T07:00:00Z", "to": "2026-09-13T07:00:00Z" },
  "samples": 1440, "up": 1432, "down": 8,
  "availability": 99.44, "avgMs": 128, "maxMs": 2103, "p95Ms": 402,
  "byNode": [
    { "nodeId": "cn-jiangsu", "up": 720, "down": 0, "availability": 100,
      "avgMs": 96, "p95Ms": 180, "latestMs": 88 }
  ]
}
```

- **只统计 `source=sched` 的定时样本**：节点自主上报与手动一键拨测不进 SLA（详见 [节点 API 与协议](/guide/node-api) 的「数据上报协议」一节）。
- 判定配置（`expectStatus` / `bothProtocols` / `requireAllStacks` / `certExpiredDown`）在**读取时**生效，改配置立刻作用于历史样本，无需回填重算。
- `availability` 是**百分数**（0~100，两位小数）。两处口径略有差异：
  - 顶层 `availability` = `up/(up+down)`；
  - `byNode[].availability` = `up/samples`（分母含无法判定的 `invalid` 样本，故通常略低）。
- `byNode[].latestMs` 是该节点窗口内最新一条可达样本的延迟。

---

## 二、拨测明细

### 2.1 `GET /api/v1/probes`

| 参数 | 必填 | 说明 |
|------|------|------|
| `node` | 否 | 节点 id 精确匹配 |
| `type` | 否 | `apiType` 精确匹配（`detail`/`ssl`/`dns`/`tcping`/`speed` …） |
| `source` | 否 | `sched`（定时）或 `biz`（业务拨测）；**其它取值返回 400** |
| `since` | 否 | 起始时间，**RFC3339**（如 `2026-09-13T00:00:00Z`），非法返回 400 |
| `limit` | 否 | 缺省 50，上限 500 |
| `offset` | 否 | 缺省 0 |

```json
{
  "items": [
    { "nodeId": "cn-jiangsu", "apiType": "tcping", "raw": "example.com:443",
      "status": 200, "latencyMs": 33, "source": "sched", "error": "",
      "createdAt": "2026-09-13T07:01:12Z" }
  ],
  "total": 1440, "limit": 50, "offset": 0
}
```

- 为控制体积，`items` **精简且不含 `body`**（原始响应体）。需要看原始 body 请用控制台「拨测明细」页。
- 不传 `source` 时按可见性返回全部来源：admin 含节点上报的 `ws`/`http` 样本，普通用户仅自己的 `sched` + `biz`。
- **`source=biz` 的边界**：admin 身份下等价于「非 `sched` 的全部来源」（含节点上报的 `ws`/`http`）；普通用户下严格等于「自己发起的 `biz`」。要精确取节点上报样本，请按 `node` / `type` 过滤。

### 2.2 `POST /api/v1/probes`

一键拨测：对节点池（或指定节点）并发发起一轮探测，**同步**聚合后一次性返回。这是公开 API 上唯一的写操作——结果会落库（`source=biz`）并归属 Token 主人。

请求体：

```json
{
  "apiType": "detail",
  "raw": "example.com",
  "query": { "port": "443" },
  "nodes": ["cn-jiangsu", "cn2-sichuan"]
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `apiType` | 是 | 探测类型；`location`/`asn` 走 IP 定位池，其余走 API 节点池 |
| `raw` | 是 | 探测目标；前导 `/` 会被去掉 |
| `query` | 否 | 透传给节点的查询参数对象（空值项被忽略） |
| `nodes` | 否 | 限定节点 id 列表；缺省 = 池内全部节点 |

```json
{
  "apiType": "detail", "raw": "example.com",
  "targeted": 2, "ok": 2, "failed": 0, "unknown": [], "persisted": 2,
  "results": [
    { "nodeId": "cn-jiangsu", "label": "中国 江苏 移动", "channel": "http",
      "status": 200, "latencyMs": 412, "body": { "ipv4": { "http_status_code": 200 } } },
    { "nodeId": "cn2-sichuan", "label": "中国 四川 电信", "channel": "ws",
      "status": 0, "latencyMs": 15000, "error": "probe timeout" }
  ]
}
```

- **HTTP 200 即便部分节点失败**：单节点成败看 `results[i].status` 与 `results[i].error`。`results[i].channel` 为 `ws` / `http`，表示该节点实际走的通道。
- `unknown`：`nodes` 里指定了但不在池中的节点 id。
- `persisted`：实际落库条数。仅 `detail`/`ssl`/`dns`/`tcping`/`speed` 这类拨测型入库；`whois`/`dnssec`/`location`/`asn` 等诊断型不入库。
- **400** 的几种情况：池为空、`nodes` 全部不在池中（错误信息里会列出已知节点 id）、`apiType` 与 `raw` 缺失。
- 这是**并发同步**调用，耗时取决于最慢的节点（WS 通道有独立超时）。客户端超时应放宽到 **30s 以上**。

---

## 三、节点与用量

### 3.1 `GET /api/v1/nodes`

节点池简表，**脱敏**——只有启用中的节点，不含上游地址。

```json
{
  "items": [
    { "nodeId": "cn-jiangsu", "label": "中国 江苏 移动",
      "ws": false, "pools": "api,location", "stack": "", "online": true, "version": "v1.4.0" }
  ],
  "total": 1
}
```

| 字段 | 说明 |
|------|------|
| `ws` | 该节点拨测是否走 WS 通道 |
| `pools` | 归属池，逗号分隔：`api` / `location` / `api,location` |
| `stack` | 仅 `api` 池有意义：`DualStack` / `IPv4` / `IPv6` |
| `online` | 是否在线（来自节点在线快照） |
| `version` | 节点上报的版本号；从未上线则为 `""` |

### 3.2 `GET /api/v1/usage?hours=24`

| 参数 | 必填 | 说明 |
|------|------|------|
| `hours` | 否 | 统计窗口（小时），缺省 24，夹在 1 ~ 2160 |

```json
{
  "hours": 24, "total": 1440, "up": 1432, "down": 8, "invalid": 0,
  "byType": [
    { "apiType": "detail", "total": 1200, "up": 1194, "down": 6, "invalid": 0, "avgMs": 128 }
  ]
}
```

- 口径：admin = 全站（`sched` + `biz` 样本）；普通用户 = 自己任务的定时样本 + 自己发起的业务拨测。
- `byType` 按 `total` 倒序；`invalid` = 无法判定的样本，不计入成败。

---

## 附：免登录的公开状态接口

SLA 任务的**分享组**对外开放状态页，无需任何凭据。它不属于 `/api/v1`（不挂鉴权），但因同样面向公众而并列在此。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/s/:token` | 只读状态页（Vue SPA，30s 自刷新；需托管侧提供 SPA fallback） |
| GET | `/api/public/status/:token` | 同数据的 JSON（页面用），参数 `hours` 同上 |

```json
{
  "token": "a1b2c3", "hours": 24,
  "tasks": [
    { "taskId": 5, "name": "官网可用性", "apiType": "detail", "target": "example.com",
      "hours": 24, "samples": 1440, "up": 1432, "down": 8, "availability": 99.44,
      "avgMs": 128, "maxMs": 2103, "p95Ms": 402,
      "byNode": [ { "nodeId": "cn-jiangsu", "up": 720, "down": 0,
                    "availability": 100, "avgMs": 96, "p95Ms": 180, "latestMs": 88 } ],
      "series": [ { "time": "2026-09-13T07:00:00Z", "minute": 29758260, "samples": 10,
                    "up": 10, "down": 0, "availability": 100, "avgMs": 120.5 } ] }
  ]
}
```

- 同一令牌的任务**同页展示**（令牌即"分享组"）；令牌为 3~32 位小写字母/数字/连字符，可自定义，管理接口见控制台「SLA 监控 → 分享」。
- **暴露面刻意收窄**：只有任务名/类型/目标 + 窗口聚合与延迟时序（`series` 为每轮多节点平均；多节点时另给 `nodeSeries` 分节点曲线）；不含 owner、上游地址、内部字段。任务勾选 `hideTarget` 时，`target` 字段**整体不出现在 JSON 里**。
- 令牌未知或已关闭分享 → `404 not_found`（`invalid share link`）。
- **防枚举限流**：每 IP 60 次/分，超限 `429 rate_limited`（随机令牌仅 24-bit 熵，必须有这层保护）；响应带 `Cache-Control: no-store`。

---

## 排错

| 现象 | 原因与处理 |
|------|------------|
| `401 missing bearer token` | 没带 `Authorization: Bearer ` 前缀（注意 `Bearer` 后有一个空格） |
| `401 invalid or revoked token` | Token 被重新生成过或已吊销 → 重新生成一枚 |
| `403 not your task` | 用普通用户的 Token 去读别人的任务 |
| `400 source 只能是 sched / biz` | `?source=` 传了其它取值 |
| `400 since 须为 RFC3339 时间` | 时间必须带时区，如 `2026-09-13T00:00:00Z` |
| `POST /probes` 400 且提示 known nodes | `nodes` 里的 id 都不在节点池 → 先用 `GET /api/v1/nodes` 取真实 id |
| `POST /probes` 耗时很久 | 同步聚合要等最慢节点 → 客户端超时设到 30s 以上 |
| 列表查不到自己刚建的任务 | 列表按 `id` 倒序；确认用的是同一个账号的 Token（普通用户只见自己的） |
