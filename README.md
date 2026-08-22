# Lemon IPW

ipw.cn 替代品，提供 IP 查询、网站检测、SSL 检查、DNS 解析、TCPing 测速、Whois、DNSSEC、ASN 查询、网站截图等功能。

**[预览站点](https://ipw.wsmdn.top/)** · **[文档](https://github.com/nomdn/ipw-cn/tree/main/docs)**

## 快速开始

```bash
curl -fsSL https://raw.githubusercontent.com/nomdn/ipw-cn/main/install.sh -o install.sh && sudo bash install.sh
```

完整部署（后端节点 / 独立中间件 / 前端）与配置说明见 [文档](https://github.com/nomdn/ipw-cn/tree/main/docs)：

- [快速开始](docs/guide/getting-started.md)
- [后端节点部署](docs/guide/deploy-node.md)
- [中间件部署](docs/guide/deploy-middleware.md)
- [前端部署](docs/guide/deploy-frontend.md)
- [配置文件](docs/guide/config.md)

## 项目结构

```
ipw-cn/
├── main.go            # Go 后端入口（Gin，自托管）
├── middleware-go/     # 独立转发中间件（Fiber v3，多节点候选）
├── frontend-ssr/      # Nuxt 4 SSR 前端
├── edgeone/           # 腾讯 EdgeOne 边缘函数后端
├── webtest/           # 网络测试工具库
├── ipdb/              # IP 数据库模块
├── ssrf/              # SSRF 防护
├── docs/              # 文档站
└── install.sh         # 一键安装脚本
```

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端（自托管） | Go + Gin + Resty |
| 独立中间件 | Go + Fiber v3（WS 通道：coder/websocket） |
| 边缘后端（EdgeOne） | Go 边缘函数 |
| SSR 前端 | Nuxt 4 + Vue 3 + Element Plus |
| DNS 查询 | miekg/dns（DoH / UDP 双通道 + 主从 failover） |
| 部署 | Docker、Cloudflare Workers、EdgeOne Pages、systemd |

## 许可证

[GPL-3.0](LICENSE)
