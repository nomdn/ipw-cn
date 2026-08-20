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

> 完整字段说明、模板与远端配置用法见 [配置文件](/guide/config)。以下为三个组件的最小示例。

**后端**（`setting.json`，env 可覆盖 `PORTS` / `DNS_SERVER` / `CORS` / `ACCESS_TOKEN` 等）：

```json
{
    "port": 8080,
    "dns-server": "119.28.28.28:53",
    "cors": "",
    "access_token": ""
}
```

**前端**（`frontend-ssr/config/index.ts`，节点 `id` 与中间件 `apiKeys` 键对应）：

```ts
const config = {
    siteUrl: "https://your-domain.com/",
    siteName: "你的站点名称",
    EnableInternalMiddleware: true,
    rateLimitPerMinute: 120,
    Middleware: <string[]>[],
    umamiHost: "", umamiScriptUrl: "", umamiWebsiteId: "",
    ICP: "", GongAn: "",
    noindex: false,
    v4OnlyAPI: "https://4.wsmdn.top", v6OnlyAPI: "https://6.wsmdn.top", DualStackAPI: "https://test.wsmdn.top",
    apiBaseUrls: [
        { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
    ],
    IPLocationAPIs: [
        { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
    ],
    TCPing: { DualStack: [], IPv4: [ { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" } ], IPv6: [] },
    SpeedTest: { DualStack: [], IPv4: [ { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" } ], IPv6: [] },
    NSLookup: [
        { label: "节点显示名", id: "node-1", url: "https://node-1.example.com/" }
    ]
}
export { config }
```

**独立中间件**（`middleware-go/setting.json`）：

```json
{
    "port": "8091",
    "cors": "",
    "rate-limit": 120,
    "remote-config-url": "",
    "apiBaseUrls": [ { "label": "中国 江苏 移动", "id": "cn-jiangsu", "url": "https://cn-jiangsu.api-ipw.wsmdn.top/" } ],
    "apiKeys": { "cn-jiangsu": "" }
}
```

> 完整字段（`http-timeout-seconds` / `ws-port` / `remote-ingore-config` 等）见 [配置文件 - 独立中间件](/guide/config#独立中间件middleware-go)。**敏感凭据（`apiKeys` / `access_token`）不得上传远端配置，仅放本地或环境变量。**

前端内置中间件通过环境变量 **`NUXT_APIKEYS`**（JSON 字符串，如 `'{"cn-jiangsu":"your-key"}'`）注入 key。

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
