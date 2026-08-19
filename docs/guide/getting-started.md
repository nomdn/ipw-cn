# 快速开始

> 本章节将指导您以最低成本部署一个功能完整的测试站
## 尝试一下
我们的开发团队已经部署了一个功能完整的预览站：   
[预览站点](https://ipw.wsmdn.top/)

## 前置准备
1. Git
2. Node.js 24+
3. Go1.26
4. 一台双栈网络环境的服务器(Linux+Systemd)
## 开始部署
### 构建后端
1. 拉取代码
```bash
git clone https://github.com/nomdn/ipw-cn && cd ipw-cn
```
2. 安装依赖
```bash
go mod tidy
```
3. 构建程序
```bash
go build
```
### 构建中间层(可选)
> [!TIP]
> 如果你的服务器到测速节点的链接延迟不大,可以跳过这一步
1. 切换目录
```bash
cd middleware-go
```
2. 安装依赖
```bash
go mod tidy
```
3. 构建程序
```bash
go build
```
### 构建前端
1. 切换目录
```bash
cd frontend-ssr
```
2. 安装依赖
```bash
pnpm install
```
3. 本地开发(可选)
```bash
pnpm dev
```
4. 生产构建
```bash
pnpm build
```
> 前端部署方式（Cloudflare Workers / EdgeOne Makers 等）见 [前端部署](/guide/deploy-frontend)。

### 配置文件

#### 后端（setting.json）

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
- `dns-server`：`ip:port` 走 UDP，`https://` URL 走 DoH
- `block-private-ips`：SSRF 防护开关
- `ipdb`：IP 数据库开关（首次启动自动下载约 200MB，之后每 24h 更新）
- `cors`：允许的请求来源（逗号分隔）
- `access_token`：API 访问令牌，留空则不启用鉴权

以上字段均可用环境变量覆盖（`PORTS` / `DNS_SERVER` / `BLOCK_PRIVATE_IPS` / `IPDB` / `CORS` / `ACCESS_TOKEN`）。需要从远端拉取配置时，设置 `remote-config-url` 或环境变量 `REMOTE_CONFIG_URL`（优先级：远端 > 环境变量 > setting.json）。

---

#### 前端（config/index.ts）

前端配置在 `frontend-ssr/config/index.ts`，部署前按需修改。完整配置项如下：

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `siteUrl` | string | 站点对外地址 |
| `siteName` | string | 站点名称（页面标题 / 描述 / 页脚品牌） |
| `EnableInternalMiddleware` | boolean | 是否启用前端内置中间件（作为 `/middleware/*` 候选最后一位兜底），默认 `true` |
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

##### 出站 IP 检测接口（v4OnlyAPI / v6OnlyAPI / DualStackAPI）

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

#### 前端内置中间件（NUXT_APIKEYS）

前端内置中间件（`frontend-ssr/server/routes/middleware/[...slug].get.ts`）作为 `/middleware/*` 候选列表的最后一位兜底，转发时也会注入 key：

- 读取方式：`useRuntimeConfig(event).apiKeys`，按请求的 `backendID`（节点 `id`）查找
- 环境变量：**`NUXT_APIKEYS`**（Nuxt 约定 `NUXT_` + 配置键 `apiKeys`）
- 格式：JSON 字符串，例如 `NUXT_APIKEYS='{"cn-jiangsu":"your-key"}'`
- 注入方式：转发请求头 `Authorization: Bearer <key>`

也可以在 `nuxt.config.ts` 的 `runtimeConfig.apiKeys` 里直接写死（不推荐，密钥不应进仓库）。

---

#### 独立中间件（middleware-go）

中间件配置在 `middleware-go/setting.json`：

```json
{
    "port": "8091",
    "cors": "",
    "apiBaseUrls": [ { "label": "中国 江苏 移动", "id": "cn-jiangsu", "url": "https://cn-jiangsu.api-ipw.wsmdn.top/" } ],
    "apiKeys": { "cn-jiangsu": "" }
}
```

`apiKeys` 用于向后端注入 `Authorization: Bearer <key>`，优先级：setting.json `apiKeys` > 环境变量 `APIKEYS`（JSON 字符串）。

### systemd 守护进程

以 `/opt/lemon-ipw` 为安装目录（二进制与 `setting.json` 放同一目录），创建服务文件：

```ini
# /etc/systemd/system/lemon-ipw.service
[Unit]
Description=Lemon IPW Backend
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/lemon-ipw
ExecStart=/opt/lemon-ipw/lemonipw
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

> `WorkingDirectory` 必须指向 `setting.json` 所在目录，后端从当前目录读取配置。

启动与管理：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now lemon-ipw
sudo systemctl status lemon-ipw
sudo journalctl -u lemon-ipw -f
```

验证：`curl http://127.0.0.1:8080/` 返回 `{"status":"ok"}` 即正常。中间件也可用同样的方式守护（`ExecStart` 指向中间件二进制，`WorkingDirectory` 指向 `middleware-go/setting.json` 所在目录）。
