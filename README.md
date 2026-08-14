# Lemon IPW

ipw.cn 替代品，提供 IP 查询、网站检测、SSL 检查、DNS 解析、TCPing 测速、Whois、DNSSEC、ASN 查询、网站截图等功能。

## 功能

- **IP 地址查询** — 支持 IPv4/IPv6，集成 ip2region、qqwry、MaxMind GeoIP、GeoCN、DbIP、Bilibili 等多数据源
- **ASN 自治系统查询** — 查询 IP 所属自治系统号，支持 MaxMind GeoLite2、DbIP 双数据源，集成 WHOIS 解析
- **网站检测** — HTTP 状态码、响应时间、Host 记录
- **SSL 证书检查** — 证书有效期、颁发机构、剩余天数
- **DNS 解析** — 多节点并发查询，支持 A/AAAA/CNAME/MX/TXT/NS/SRV/PTR/CAA 等记录类型
- **DNSSEC 验证** — 检测域名 DNSSEC 签名状态，验证 DNSKEY、RRSIG、DS 记录链式信任关系
- **Whois 查询** — 域名注册信息、注册商、到期时间等
- **TCPing** — TCP 连接延迟测试，支持 IPv4/IPv6 双栈
- **网站测速** — 下载速度、响应头、Host 记录
- **网站截图** — 在线网站页面截图
- **SSRF 防护** — 自动拦截对内网/私有 IP 的出站请求
- **暗色模式** — 支持明暗主题切换

## 项目结构

```
ipw-cn/
├── main.go                  # Go 后端入口（Gin 框架，自托管）
├── go.mod                   # Go 模块定义
├── Dockerfile               # 后端 Docker 镜像
├── setting.json             # 后端运行配置
│
├── ipdb/                    # IP 数据库查询模块
│   ├── searchip.go          # 多数据源 IP 查询（ip2region/qqwry/MaxMind/GeoCN/DbIP/Bilibili）
│   └── ipdb.go              # 数据库下载与更新
│
├── webtest/                 # 网络测试工具
│   ├── dns.go               # DNS 查询（基于 miekg/dns）
│   ├── dnssec.go            # DNSSEC 验证
│   ├── tcping.go            # TCP 连接测试
│   └── whois.go             # WHOIS 域名查询
│
├── ssrf/                    # SSRF 防护模块
│   └── ssrf.go              # 私有 IP 拦截、安全跳转校验
│
├── middleware-go/           # 独立转发中间件（Go Fiber v3，可单独部署）
│   ├── main.go              # 中间件入口（/v1/* 与 /middleware/* 路由）
│   ├── setting.json         # 中间件配置（节点列表、apiKeys、cors 等）
│   ├── setting.json.example # 中间件配置示例
│   ├── Dockerfile           # 中间件 Docker 镜像
│   └── middleware-go-linux-arm64  # Linux ARM64 预编译二进制
│
├── frontend-ssr/            # Nuxt 4 SSR 前端（主版本，部署至 Cloudflare Workers）
│   ├── app/                 # Nuxt 应用源码
│   │   └── pages/           # 页面组件（location/dns/tcping/ssl/speedtest/whois/dnssec/asn/screenshot）
│   ├── config/              # 前端配置（API 地址、节点列表）
│   ├── nuxt.config.ts       # Nuxt 配置
│   ├── package.json
│   ├── pnpm-workspace.yaml
│   └── wrangler.jsonc       # Cloudflare Workers 部署配置
│
├── edgeone/                 # 腾讯 EdgeOne 边缘函数版本后端
│   ├── cloud-functions/     # Go 边缘函数源码
│   │   ├── index.go         # 后端入口（Gin 框架）
│   │   ├── webtest/         # 网络测试工具
│   │   ├── ssrf/            # SSRF 防护
│   │   └── go.mod
│   └── .edgeone/            # EdgeOne 部署配置
│
├── lemon-getip/             # Cloudflare Workers IP 查询服务（Hono + TypeScript）
│   ├── src/index.ts
│   ├── wrangler.jsonc
│   ├── package.json
│   └── test/
│
├── edgeone-getip/           # EdgeOne Pages IP 查询服务（TypeScript 边缘函数）
│   ├── edge-functions/index.ts
│   └── .edgeone/
│
├── .github/workflows/       # CI/CD 流水线
│   ├── frontend-ssr.yml     # 部署至 Cloudflare Workers
│   ├── workers.yml          # 部署 lemon-getip 至 Cloudflare Workers
│   ├── edgeone-backend.yml  # 部署 edgeone 至 EdgeOne Pages
│   └── edgeone-getip.yml    # 部署 edgeone-getip 至 EdgeOne Pages
│
├── tmp/                     # IP 数据库下载临时目录
├── short.txt                # GeoCN 行政区划短表（省市级）
├── full.txt                 # GeoCN 行政区划全表（区县级）
├── ipw-backend.exe          # Windows 预编译后端二进制
├── .node-version            # Node.js 版本（v22）
└── LICENSE                  # GPL-3.0
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端（自托管） | Go 1.26 + Gin + Resty |
| 独立中间件 | Go 1.26 + Fiber v3（转发中间件，多节点候选） |
| 边缘后端（EdgeOne） | Go（边缘函数） |
| SSR 前端 | Nuxt 4 + Vue 3 + Element Plus + VueUse |
| Cloudflare Workers | Hono + TypeScript + Wrangler |
| EdgeOne Pages | TypeScript 边缘函数 |
| IP 数据库 | ip2region、qqwry.ipdb、MaxMind GeoLite2、GeoCN、DbIP、Bilibili API |
| DNS 查询 | miekg/dns（Go）/ 原生 DNS-over-HTTPS（EdgeOne） |
| 其他 | SSRF 防护、Zstandard 压缩、WordPress mshots 截图 |
| 部署 | Docker、Cloudflare Workers、EdgeOne Pages |

## 快速开始

### 后端（自托管）

```bash
# 安装依赖
go mod download

# 配置 setting.json（可选）
# {
#   "port": 8080,
#   "gh-proxy": "https://fastgit.cc/",
#   "single-stack": "",       # "ipv4" 或 "ipv6" 仅启用单栈
#   "dns-server": "",         # 自定义 DNS 服务器
#   "block-private-ips": true,# SSRF 防护开关
#   "ipdb": true,             # IP 数据库开关
#   "cors": "",               # 允许的请求来源，详见 https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Guides/CORS
#   "access_token": ""        # API 访问令牌（可选）
# }

# 运行
go run main.go
```

首次启动时会自动下载 IP 数据库文件（约 200MB），之后每 24 小时自动更新。

### 独立中间件（middleware-go）

转发中间件，供 SSR 前端在多个上游节点之间做候选重试，可独立部署。

```bash
cd middleware-go

# 构建（Linux ARM64 示例）
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o middleware-go-linux-arm64 .

# 运行（需配置文件，可用 SETTING_FILE 指定路径）
./middleware-go-linux-arm64

# 查看版本
./middleware-go-linux-arm64 --version
```

所有配置均可通过环境变量覆盖，无需修改配置文件（详见下方「中间件配置」）。

### SSR 前端（部署至 Cloudflare Workers）

```bash
cd frontend-ssr

# 安装依赖（需要 pnpm）
pnpm install

# 开发
pnpm dev

# 构建并部署
pnpm deploy
```

### SPA 前端（旧版本，部署至 Cloudflare Pages）

> [!WARNING]
> 原生 SPA 已停止支持。要部署 SPA 版本前端请使用 Nuxt SPA 版本。

### Docker 部署

```bash
docker build -t lemon-ipw .
docker run -p 8080:8080 -v $(pwd)/setting.json:/app/setting.json lemon-ipw
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/location/:ip` | 查询指定 IP 地理位置 |
| GET | `/v1/location` | 查询请求者 IP 地理位置 |
| GET | `/v1/asn/:ip` | 查询 IP 所属自治系统号（ASN） |
| GET | `/v1/detail/*url` | 检测网站可达性（HTTP 状态码、响应时间） |
| GET | `/v1/ssl/*url` | 检查 SSL 证书信息 |
| GET | `/v1/speed/:version/*url` | 网站测速（version: v4/v6/dual） |
| GET | `/v1/tcping/:ip?port=80&count=4` | TCP 连接测试 |
| GET | `/v1/dns/:type/*domain` | DNS 解析（type: A/AAAA/CNAME/MX/TXT/NS/SRV/PTR/CAA） |
| GET | `/v1/dnssec/:domain` | DNSSEC 验证 |
| GET | `/v1/whois/:domain` | Whois 域名查询 |
| GET | `/` | 健康检查 |

## 部署架构

本项目支持多平台部署，前端和后端可独立部署到不同平台：

- **自托管后端**：使用 Docker 或直接运行 Go 二进制，配合 `setting.json` 配置
- **独立转发中间件**：`middleware-go/` 使用 Go Fiber v3，提供 `/v1/*` 与 `/middleware/*` 转发路由，供前端做多节点候选重试
- **Cloudflare Workers 后端**：`lemon-getip/` 使用 Hono 框架部署到 Cloudflare Workers
- **EdgeOne 边缘后端**：`edgeone/` 和 `edgeone-getip/` 部署到腾讯 EdgeOne Pages
- **SSR 前端**：`frontend-ssr/` 使用 Nuxt 4 + Wrangler 部署到 Cloudflare Workers

## 配置

### 后端环境变量

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `port` | `PORT` | `8080` | 监听端口 |
| `gh-proxy` | `GH_PROXY` | `""` | GitHub 文件代理（用于下载数据库） |
| `single-stack` | `SINGLE_STACK` | `""` | 单栈模式：`ipv4` 或 `ipv6` |
| `dns-server` | `DNS_SERVER` | `119.28.28.28:53` | DNS 服务器地址 |
| `block-private-ips` | `BLOCK_PRIVATE_IPS` | `true` | SSRF 防护开关 |
| `ipdb` | `IPDB` | `true` | IP 数据库开关 |
| `access_token` | `ACCESS_TOKEN` | `""` | API 访问令牌 |
| `cors` | `CORS` | `""` | 允许的请求来源 |

### 前端配置

前端 API 地址和多节点配置位于各前端项目的 `config/` 目录中。

### 中间件配置（middleware-go）

配置来源优先级：**环境变量 > `setting.json` > 默认值**。`setting.json` 采用平铺结构（键名与前端 `config/index.ts` 一致），也可用环境变量覆盖任意配置项；数组/对象类配置项从环境变量读取 **JSON 字符串**解析。

```json
{
    "port": "8091",
    "httpTimeoutSeconds": 30,
    "cors": "https://ipw.wsmdn.top",
    "apiBaseUrls": [{ "label": "中国 江苏 移动", "id": "cn-jiangsu", "url": "https://cn-jiangsu.api-ipw.wsmdn.top/" }],
    "IPLocationAPIs": [],
    "TCPing": { "DualStack": [], "IPv4": [], "IPv6": [] },
    "SpeedTest": { "DualStack": [], "IPv4": [], "IPv6": [] },
    "NSLookup": [],
    "apiKeys": { "cn-jiangsu": "token" }
}
```

| 配置项 | 环境变量 | 类型 | 说明 |
|--------|----------|------|------|
| `port` | `PORT` | 数字/字符串 | 监听端口 |
| `httpTimeoutSeconds` | `HTTP_TIMEOUT` | 数字 | 上游请求超时（秒） |
| `cors` | `CORS` | 字符串 | 逗号分隔的允许域名；为空则允许所有（`*`） |
| `apiBaseUrls` | `API_BASE_URLS` | JSON 数组 | whois/ssl/detail 上游节点 |
| `IPLocationAPIs` | `IP_LOCATION_APIS` | JSON 数组 | location/asn 上游节点 |
| `TCPing` | `TCPING` | JSON 对象 | tcping 节点（DualStack/IPv4/IPv6） |
| `SpeedTest` | `SPEED_TEST` | JSON 对象 | speed 节点（DualStack/IPv4/IPv6） |
| `NSLookup` | `NS_LOOKUP` | JSON 数组 | dns/dnssec 上游节点 |
| `apiKeys` | `APIKEYS` | JSON 对象 | `{"backendID": "token"}`，转发时加 `Authorization: Bearer <token>` |
| 配置文件路径 | `SETTING_FILE` | 字符串 | 指定 setting.json 路径（默认 `./setting.json` 或 `../setting.json`） |

**路由格式**：`/{prefix}/{backendID}/{apiType}/{raw...}`，其中 `prefix` 为 `/v1` 或 `/middleware`，`apiType` 支持 `whois/dns/location/ssl/asn/dnssec/detail/tcping/speed`。响应透传上游状态码与 body；上游网络不可达返回 502。

**版本信息**：通过 ldflags 注入 `main.VERSION / main.COMMIT / main.BUILD_TIME`，运行 `--version` 查看。

## 许可证

[GPL-3.0](LICENSE)
