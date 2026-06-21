# Lemon IPW

ipw.cn 替代品，提供 IP 查询、网站检测、SSL 检查、DNS 解析、TCPing 测速等功能。

## 功能

- **IP 地址查询** — 支持 IPv4/IPv6，集成 ip2region、qqwry、MaxMind GeoIP、GeoCN、Bilibili 等多数据源
- **网站检测** — HTTP 状态码、响应时间、Host 记录
- **SSL 证书检查** — 证书有效期、颁发机构、剩余天数
- **DNS 解析** — 多节点并发查询，支持 A/AAAA/CNAME/MX/TXT/NS/SRV 等记录类型
- **TCPing** — TCP 连接延迟测试，支持 IPv4/IPv6 双栈
- **网站测速** — 下载速度、响应头、Host 记录
- **暗色模式** — 支持明暗主题切换

## 项目结构

```
ipw-cn/
├── main.go                  # Go 后端入口（Gin 框架）
├── ipdb/                    # IP 数据库查询模块
│   ├── searchip.go          # 多数据源 IP 查询（ip2region/qqwry/MaxMind/GeoCN/Bilibili）
│   └── ipdb.go              # 数据库下载与更新
├── webtest/                 # 网络测试工具
│   ├── dns.go               # DNS 查询（基于 miekg/dns）
│   └── tcping.go            # TCP 连接测试
├── frontend-ssr/            # Nuxt 3 SSR 前端（当前主版本）
│   ├── app/
│   │   ├── app.vue          # 根布局（导航栏、暗色模式、Umami 统计）
│   │   └── pages/           # 页面组件
│   ├── config/index.ts      # 前端配置（API 地址、节点列表）
│   └── nuxt.config.ts       # Nuxt 配置
├── frontend/                # Vue 3 SPA 前端（旧版本，Cloudflare Pages 部署）
│   ├── src/
│   │   ├── pages/           # 页面组件
│   │   └── config/index.ts  # 前端配置
│   └── vite.config.ts       # Vite 配置
├── edgeone/                 # 腾讯 EdgeOne 边缘函数版本后端
│   ├── cloud-functions/     # 源码
│   └── .edgeone/            # EdgeOne 部署配置
├── setting.json             # 后端运行配置
├── dbip-city-lite.mmdb      # IP 数据库文件
├── GeoLite2-City.mmdb
├── GeoLite2-ASN.mmdb
├── GeoCN.mmdb
├── ip2region_v4.xdb
├── ip2region_v6.xdb
├── qqwry.ipdb
├── full.txt                 # 行政区划全称映射
└── short.txt                # 行政区划简称映射
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.26 + Gin + Resty |
| SSR 前端 | Nuxt 4 + Vue 3 + Element Plus + VueUse |
| SPA 前端 | Vue 3 + Vite + Element Plus + Vue Router |
| EdgeOne 版本 | Go（边缘函数） |
| IP 数据库 | ip2region、qqwry.ipdb、MaxMind GeoLite2、GeoCN、DbIP、Bilibili API |
| DNS 查询 | miekg/dns（Go）/ 原生 DNS-over-HTTPS（EdgeOne） |

## 快速开始

### 后端

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

### SSR 前端

```bash
cd frontend-ssr

# 安装依赖
pnpm install

# 开发
pnpm dev

# 构建
pnpm build
```

### SPA 前端（旧版本）

```bash
cd frontend

# 安装依赖
npm install

# 开发
npm run dev

# 部署到 Cloudflare Pages
npm run deploy
```

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/location/:ip` | 查询指定 IP 地理位置 |
| GET | `/v1/location` | 查询请求者 IP 地理位置 |
| GET | `/v1/detail/*url` | 检测网站可达性（HTTP 状态码、响应时间） |
| GET | `/v1/ssl/*url` | 检查 SSL 证书信息 |
| GET | `/v1/tcping/:ip?port=80&count=4` | TCP 连接测试 |
| GET | `/v1/dns/:type/*domain` | DNS 解析（type: A/AAAA/CNAME/MX/TXT/NS/SRV） |
| GET | `/v1/speed/:version/*url` | 网站测速（version: v4/v6/dual） |
| GET | `/` | 健康检查 |

## 配置

后端通过 `setting.json` 或环境变量配置：

| 配置项 | 环境变量 | 默认值 | 说明 |
|--------|----------|--------|------|
| `port` | `PORT` | `8080` | 监听端口 |
| `gh-proxy` | `GH_PROXY` | `""` | GitHub 文件代理（用于下载数据库） |
| `single-stack` | `SINGLE_STACK` | `""` | 单栈模式：`ipv4` 或 `ipv6` |
| `dns-server` | `DNS_SERVER` | `119.28.28.28:53` | DNS 服务器地址 |

## 许可证

[GPL-3.0](LICENSE)
