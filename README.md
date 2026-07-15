# Lemon IPW

ipw.cn 替代品，提供 IP 查询、网站检测、SSL 检查、DNS 解析、TCPing 测速等功能。

## 功能

- **IP 地址查询** — 支持 IPv4/IPv6，集成 ip2region、qqwry、MaxMind GeoIP、GeoCN、DbIP、Bilibili 等多数据源
- **网站检测** — HTTP 状态码、响应时间、Host 记录
- **SSL 证书检查** — 证书有效期、颁发机构、剩余天数
- **DNS 解析** — 多节点并发查询，支持 A/AAAA/CNAME/MX/TXT/NS/SRV/PTR/CAA 等记录类型
- **TCPing** — TCP 连接延迟测试，支持 IPv4/IPv6 双栈
- **网站测速** — 下载速度、响应头、Host 记录
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
│   └── tcping.go            # TCP 连接测试
│
├── frontend-ssr/            # Nuxt 4 SSR 前端（主版本，部署至 Cloudflare Workers）
│   ├── app/                 # Nuxt 应用源码
│   ├── config/              # 前端配置（API 地址、节点列表）
│   ├── nuxt.config.ts       # Nuxt 配置
│   ├── package.json
│   ├── pnpm-workspace.yaml
│   └── wrangler.jsonc       # Cloudflare Workers 部署配置
│
│
├── edgeone/                 # 腾讯 EdgeOne 边缘函数版本后端
│   ├── cloud-functions/     # Go 边缘函数源码
│   │   ├── index.go         # 后端入口（Gin 框架）
│   │   ├── webtest/         # 网络测试工具
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
│
├── ipw-backend.exe          # Windows 预编译后端二进制
├── .node-version            # Node.js 版本（v22）
└── LICENSE                  # GPL-3.0
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端（自托管） | Go 1.26 + Gin + Resty |
| 边缘后端（EdgeOne） | Go（边缘函数） |
| SSR 前端 | Nuxt 4 + Vue 3 + Element Plus + VueUse |
| Cloudflare Workers | Hono + TypeScript + Wrangler |
| EdgeOne Pages | TypeScript 边缘函数 |
| IP 数据库 | ip2region、qqwry.ipdb、MaxMind GeoLite2、GeoCN、DbIP、Bilibili API |
| DNS 查询 | miekg/dns（Go）/ 原生 DNS-over-HTTPS（EdgeOne） |
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
#   "dns-server": ""          # 自定义 DNS 服务器
# }

# 运行
go run main.go
```

首次启动时会自动下载 IP 数据库文件（约 200MB），之后每 24 小时自动更新。

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
> 原生SPA已停止支持
> 要部署SPA版本前端请使用Nuxt SPA版本

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
| GET | `/v1/detail/*url` | 检测网站可达性（HTTP 状态码、响应时间） |
| GET | `/v1/ssl/*url` | 检查 SSL 证书信息 |
| GET | `/v1/tcping/:ip?port=80&count=4` | TCP 连接测试 |
| GET | `/v1/dns/:type/*domain` | DNS 解析（type: A/AAAA/CNAME/MX/TXT/NS/SRV/PTR/CAA） |
| GET | `/v1/speed/:version/*url` | 网站测速（version: v4/v6/dual） |
| GET | `/` | 健康检查 |

## 部署架构

本项目支持多平台部署，前端和后端可独立部署到不同平台：

- **自托管后端**：使用 Docker 或直接运行 Go 二进制，配合 `setting.json` 配置
- **Cloudflare Workers 后端**：`lemon-getip/` 使用 Hono 框架部署到 Cloudflare Workers
- **EdgeOne 边缘后端**：`edgeone/cloud-functions/` 和 `edgeone-getip/` 部署到腾讯 EdgeOne Pages
- **SSR 前端**：`frontend-ssr/` 使用 Nuxt 4 + Wrangler 部署到 Cloudflare Workers
- **SPA 前端（旧）**：`frontend/` 使用 Vite 部署到 Cloudflare Pages

## 配置

### 后端环境变量

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `port` | `PORT` | `8080` | 监听端口 |
| `gh-proxy` | `GH_PROXY` | `""` | GitHub 文件代理（用于下载数据库） |
| `single-stack` | `SINGLE_STACK` | `""` | 单栈模式：`ipv4` 或 `ipv6` |
| `dns-server` | `DNS_SERVER` | `119.28.28.28:53` | DNS 服务器地址 |

### 前端配置

前端 API 地址和多节点配置位于各前端项目的 `config/` 目录中。

## 许可证

[GPL-3.0](LICENSE)
