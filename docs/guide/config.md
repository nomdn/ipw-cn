# 配置文件

各组件配置说明与完整字段参考。快速上手示例见 [快速开始](/guide/getting-started)。

## 后端（setting.json）

后端从当前目录读取 `setting.json`，不存在时使用默认值。最小配置：

```json
{
    "port": 8080,
    "dns-server": "119.28.28.28:53",
    "block-private-ips": true,
    "ipdb": true,
    "cors": "",
    "access_token": ""
}
```

- `port`：监听端口
- `dns-server`：DNS 解析服务器，支持**主从 failover**——逗号分隔多地址（如 `"119.28.28.28:53,223.5.5.5:53"`），第一个为主，主查询失败（超时/网络错误）自动切换下一个；每项支持 `ip:port`（UDP）或 `https://` URL（DoH）
- `dnssec-server`：**DNSSEC 专用** DNS 服务器（同样支持主从逗号分隔）；留空则沿用 `dns-server`
- `block-private-ips`：SSRF 防护开关
- `ipdb`：IP 数据库开关（首次启动自动下载约 200MB，之后每 24h 更新）
- `cors`：允许的请求来源（逗号分隔）
- `access_token`：API 访问令牌，留空则不启用鉴权

以上字段均可用环境变量覆盖（`PORTS` / `DNS_SERVER` / `DNSSEC_DNS_SERVER` / `BLOCK_PRIVATE_IPS` / `IPDB` / `CORS` / `ACCESS_TOKEN`）。需要从远端拉取配置时，设置 `remote-config-url` 或环境变量 `REMOTE_CONFIG_URL`（优先级：远端 > 环境变量 > setting.json）。

---

## 前端（config/index.ts）

前端配置在 `frontend-ssr/config/index.ts`，部署前按需修改。完整配置项如下：

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `siteUrl` | string | 站点对外地址 |
| `siteName` | string | 站点名称（页面标题 / 描述 / 页脚品牌） |
| `EnableInternalMiddleware` | boolean | 是否启用前端内置中间件（作为 `/middleware/*` 候选最后一位兜底），默认 `true` |
| `rateLimitPerMinute` | number | 内置中间件单 IP 限流次数（次/分钟），默认 `120`，`0` 表示不限流 |
| `Middleware` | string[] | 外部独立中间件 base URL 列表，`/middleware/*` 请求依次尝试、出错重试下一个 |
| `umamiHost` / `umamiScriptUrl` / `umamiWebsiteId` | string | Umami 统计配置 |
| `ICP` / `GongAn` | string | 网站备案号（页脚展示） |
| `noindex` | boolean | 全站是否禁止搜索引擎索引 |
| `v4OnlyAPI` / `v6OnlyAPI` / `DualStackAPI` | string | IPv4 / IPv6 / 双栈 IP 查询接口 |
| `apiBaseUrls` | 节点数组 | 基础 API 节点列表（detail / ssl / whois 等） |
| `IPLocationAPIs` | 节点数组 | IP 归属地 / ASN 查询节点列表 |
| `TCPing` | 节点分组 | TCPing 节点，按 `DualStack` / `IPv4` / `IPv6` 分组 |
| `SpeedTest` | 节点分组 | 测速节点，按 `DualStack` / `IPv4` / `IPv6` 分组 |
| `NSLookup` | 节点数组 | DNS 解析节点列表 |

完整模板（复制到 `frontend-ssr/config/index.ts`，替换占位值即可）：

```ts
const config = {
    siteUrl: "https://your-domain.com/",        // 站点对外地址
    siteName: "你的站点名称",                     // 站点名称（页脚展示）
    // 是否启用前端内置中间件（本地转发，作为候选最后一位兜底）
    EnableInternalMiddleware: true,
    // 内置中间件单 IP 限流次数（次/分钟），0 表示不限流
    rateLimitPerMinute: 120,
    // 外部独立中间件 base URL 列表（可留空，仅用内置中间件）
    Middleware: <string[]>[],
    // Umami 统计（不需要留空）
    umamiHost: "",
    umamiScriptUrl: "",
    umamiWebsiteId: "",
    // 备案号（页脚展示，不需要留空）
    ICP: "",
    GongAn: "",
    // 是否禁止搜索引擎索引
    noindex: false,
    // 出站 IP 检测接口（页面直连需 CORS，建议 wsmdn.top 的三个）
    v4OnlyAPI: "https://4.wsmdn.top",
    v6OnlyAPI: "https://6.wsmdn.top",
    DualStackAPI: "https://test.wsmdn.top",
    // 基础 API 节点（detail / ssl / whois 等）
    apiBaseUrls: [
        { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
    ],
    // IP 归属地 / ASN 查询节点
    IPLocationAPIs: [
        { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
    ],
    // TCPing 节点：IPv4-only 节点只能放 IPv4 分组，不能放 DualStack / IPv6
    TCPing: {
        DualStack: [],
        IPv4: [
            { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
        ],
        IPv6: []
    },
    // 测速节点：同上，IPv4-only 节点只能放 IPv4 分组
    SpeedTest: {
        DualStack: [],
        IPv4: [
            { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
        ],
        IPv6: []
    },
    // DNS 解析节点
    NSLookup: [
        { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
    ]
}
export { config }
```

> 节点 `id` 需与后端节点 / 中间件 `apiKeys` 的键保持一致，中间件按 `id` 注入 `Authorization: Bearer <key>`。

节点数组项均为 `{ label, id, url }` 结构，例如：

```ts
apiBaseUrls: [
    { label: "中国 江苏 移动", id: "cn-jiangsu", url: "https://cn-jiangsu.api-ipw.wsmdn.top/" }
]
```

`label` 为节点展示名，`id` 为节点唯一标识（中间件 `apiKeys` 也按此 id 注入 key），`url` 为节点 base URL。`TCPing` / `SpeedTest` 节点再多一层 `DualStack` / `IPv4` / `IPv6` 分组，分别对应双栈 / 单栈测试时使用的节点列表。

### 出站 IP 检测接口（v4OnlyAPI / v6OnlyAPI / DualStackAPI）

`v4OnlyAPI` / `v6OnlyAPI` / `DualStackAPI` 用于获取请求方的出站 IP（分别对应 IPv4、IPv6、双栈），首页与部分页面会直接请求。可直接使用的接口：

| 接口 | 类型 |
|------|------|
| `https://4.wsmdn.top` | IPv4 |
| `https://6.wsmdn.top` | IPv6 |
| `https://test.wsmdn.top` | 双栈 |
| `https://ipv4.ip.sb` | IPv4 |
| `https://ipv6.ip.sb` | IPv6 |
| `https://ip.sb` | 双栈 |
| `https://4.itdog.cn` | IPv4 |
| `https://6.itdog.cn` | IPv6 |
| `https://v.itdog.cn` | 双栈 |

> **CORS 说明**：以上接口仅 `4.wsmdn.top` / `6.wsmdn.top` / `test.wsmdn.top` 返回 `Access-Control-Allow-Origin: *`，其余（ip.sb、itdog.cn）不带 CORS 头。前端页面在客户端导航时会从浏览器直接请求这些接口，**没有 CORS 会被浏览器拦截**，因此页面直连检测建议使用 wsmdn.top 的三个；ip.sb / itdog.cn 适合在服务端或命令行场景使用。

---

## 前端内置中间件（NUXT_APIKEYS）

前端内置中间件（`frontend-ssr/server/routes/middleware/[...slug].get.ts`）作为 `/middleware/*` 候选列表的最后一位兜底，转发时也会注入 key：

- 读取方式：`useRuntimeConfig(event).apiKeys`，按请求的 `backendID`（节点 `id`）查找
- 环境变量：**`NUXT_APIKEYS`**（Nuxt 约定 `NUXT_` + 配置键 `apiKeys`）
- 格式：JSON 字符串，例如 `NUXT_APIKEYS='{"cn-jiangsu":"your-key"}'`
- 注入方式：转发请求头 `Authorization: Bearer <key>`

也可以在 `nuxt.config.ts` 的 `runtimeConfig.apiKeys` 里直接写死（不推荐，密钥不应进仓库）。

---

## 独立中间件（middleware-go）

中间件配置在 `middleware-go/setting.json`：

```json
{
    "port": "8091",
    "http-timeout-seconds": 30,
    "cors": "",
    "rate-limit": 120,
    "ws-port": 8092,
    "remote-config-url": "",
    "remote-ingore-config": [],
    "apiBaseUrls": [ { "label": "中国 江苏 移动", "id": "cn-jiangsu", "url": "https://cn-jiangsu.api-ipw.wsmdn.top/" } ],
    "apiKeys": { "cn-jiangsu": "" }
}
```

- `rate-limit`：单 IP 每分钟限流次数，默认 `120`，`0` 表示不限流（也可用环境变量 `RATE_LIMIT` 覆盖）
- `ws-port`：WS 服务端口，默认 `8092`，`0` 表示关闭 WS 通道（也可用环境变量 `WS_PORT` 覆盖；优先级：远端配置 > 环境变量 > setting.json）
- `remote-config-url`：远端配置地址，也可用环境变量 `REMOTE_CONFIG_URL`（环境变量优先）
- `remote-ingore-config`：**不被远端配置覆盖的配置项列表**（数组），如 `["port", "rate-limit"]`；也可用环境变量 `REMOTE_INGORE_CONFIG`（JSON 数组字符串，优先于 setting.json）
- `apiKeys`：用于向后端注入 `Authorization: Bearer <key>`，优先级：setting.json `apiKeys` > 环境变量 `APIKEYS`（JSON 字符串）。**敏感凭据，强制忽略，不随远端配置覆盖**
- 节点数组项（`apiBaseUrls` / `IPLocationAPIs` / `TCPing` / `SpeedTest` / `NSLookup`）与前端 `config/index.ts` 结构一致；节点加 `"ws": true` 可让该节点走 WS 通道通信，详见 [中间件部署 - WS 通道](/guide/deploy-middleware#ws-通道拨测数据经-websocket-传输)

---

## 远端配置（REMOTE_CONFIG_URL）

**什么是远端配置**：把配置文件放到一个可访问的 URL 上（如 Gist、对象存储、静态托管、任意 HTTP 服务），部署时只需设置一个环境变量，之后改配置**不用重新部署、不用重启**，下次启动自动拉取最新配置。适合多节点统一管理、批量更新节点配置的场景。

**支持的组件**（三个组件均已支持）：

| 组件 | 触发方式 | 远端配置格式 |
|------|----------|--------------|
| 后端（主线 `main.go`） | `REMOTE_CONFIG_URL` 或 setting.json 的 `remote-config-url` | 后端 `setting.json` 格式 |
| 后端（`edgeone/`） | `REMOTE_CONFIG_URL` | 后端 `setting.json` 格式 |
| 独立中间件（middleware-go） | `REMOTE_CONFIG_URL` 或 setting.json 的 `remote-config-url` | 中间件 `setting.json` 格式 |

**合并规则**（重要，逐字段生效）：

- **优先级：远端配置 > 环境变量 > setting.json**
- 远端 JSON 中**非空的字段才覆盖**本地值：字符串非空、数组非空、数字非 0（`rate-limit` 为指针语义，缺省=不覆盖，显式 0=不限流）
- 部分字段缺省不会冲掉本地配置——远端只需包含要覆盖的字段
- **敏感凭据强制忽略，永远不被远端覆盖**（代码写死，无需配置）：
  - 后端 `access_token`：不随远端覆盖（保持 环境变量 > setting.json），远端里的 `access_token` 会被忽略
  - 中间件 `apiKeys`：不随远端覆盖，远端里的 `apiKeys` 会被忽略
- `remote-ingore-config`（后端键名 `remote-ingore-config` / env `REMOTE_INGORE_CONFIG`）：额外指定**不被远端覆盖的配置项列表**，数组中的键即使远端下发也不生效。适用于 access_token / apiKeys 之外的敏感项（如 `ipw-node-key`、`dns-server` 等），也可覆盖"非空才覆盖"规则

**拉取行为**：

- 在**进程启动时的配置加载阶段**一次性拉取
- HTTP 超时 **10 秒**，仅接受 **200** 响应（404/5xx 视为失败）
- 拉取或 JSON 解析失败：打印警告日志 `WARN failed to fetch remote config, fallback to local: ...`，**自动回退本地配置**，服务正常启动，不会崩溃
- 拉取成功：启动日志打印 `remote config applied from <url>`
- **没有重试、没有定时刷新**：改远端配置后需要重启进程才会重新拉取

**示例一：后端**（主线，远端 JSON 放在 `https://example.com/ipw-config.json`）：

```json
{
    "port": 8080,
    "single-stack": "ipv4",
    "dns-server": "119.28.28.28:53",
    "cors": "https://ipw.wsmdn.top",
    "remote-config-url": ""
}
```

部署时：

```bash
REMOTE_CONFIG_URL=https://example.com/ipw-config.json ./lemonipw
```

**示例二：独立中间件**（middleware-go，远端 JSON 放在 `https://example.com/mw-config.json`）：

```json
{
    "port": "8091",
    "cors": "",
    "rate-limit": 120,
    "apiBaseUrls": [ { "label": "中国 江苏 移动", "id": "cn-jiangsu", "url": "https://cn-jiangsu.api-ipw.wsmdn.top/" } ]
}
```

> **⚠️ 敏感凭据不得上传远端**：`apiKeys`（及后端 `access_token`、节点 key 等密钥）**禁止**写入远端配置——即使写了也会被强制忽略。凭据只放本地 setting.json 或环境变量。

部署时：

```bash
REMOTE_CONFIG_URL=https://example.com/mw-config.json ./middleware-go-linux-amd64
```

systemd 中同样用 `Environment="REMOTE_CONFIG_URL=..."` 配置。

**如何验证生效**：

- 启动日志出现 `remote config applied from <url>` 即拉取成功
- 用 `curl -i <远端URL>` 确认远端返回 `200` 且是合法 JSON
- 实际操作验证：在远端把 `port` 改成另一个值，重启进程，观察监听端口是否变化

**安全与排查**：

- **⚠️ 敏感凭据不得上传远端**：远端配置（Gist / 对象存储 / 静态托管等）**禁止包含任何密钥**——`apiKeys`、`access_token`、节点 key（`ipw-node-key`）等。这些值会被强制忽略，但远端文件本身一旦泄露等于直接暴露凭据；凭据一律放本地 `setting.json` 或环境变量注入
- 远端 URL 必须能被节点访问；优先使用 **HTTPS** 地址
- 远端配置文件如果放在公开可访问的位置（公开 Gist、无鉴权对象存储），即便不含密钥，也等于把节点拓扑、节点 id 等敏感信息公之于众，建议放在私有仓库 / 带鉴权的对象存储 / 内部服务上
- 常见失败排查：

| 现象 | 原因 | 排查 |
|------|------|------|
| 日志 `WARN ... status 404` | URL 路径错误或远端文件不存在 | `curl -i <URL>` 检查返回码 |
| 日志 `WARN ... timeout` / 连接失败 | 网络不通或超时 | 确认节点可访问该 URL |
| 日志 `WARN ... invalid ... JSON` | 远端内容不是合法 JSON 或格式不符 | 检查 JSON 合法性、字段名与组件格式一致 |
| 启动日志无 remote 相关输出 | `REMOTE_CONFIG_URL` 未设置 | `env \| grep REMOTE` 确认环境变量已注入 |
