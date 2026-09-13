# Lemon IPW

ipw.cn 替代品，提供 IP 查询、网站检测、SSL 检查、DNS 解析、TCPing 测速、Whois、DNSSEC、ASN 查询、网站截图等功能。

**[预览站点](https://ipw.wsmdn.top/)** · **[文档](https://github.com/nomdn/ipw-cn/tree/main/docs)**

## 快速开始

```bash
curl -fsSL https://raw.githubusercontent.com/nomdn/ipw-cn/main/install.sh -o install.sh && sudo bash install.sh
```

完整部署（后端节点 / 独立中间件 / 前端）与配置说明见 [文档](https://github.com/nomdn/ipw-cn/tree/main/docs)：

- [功能总览](docs/info/features.md)（探测能力、IP 数据源、可靠性设计）
- [快速开始](docs/guide/getting-started.md)
- [后端节点部署](docs/guide/deploy-node.md)
- [中间件部署](docs/guide/deploy-middleware.md)
- [前端部署](docs/guide/deploy-frontend.md)
- [配置文件](docs/guide/config.md)
- [节点 API 与协议](docs/guide/node-api.md)（业务接口、管理接口、WS 协议、上报协议）
- [公开 API（/api/v1）](docs/guide/public-api.md)（收集中心程序化接口：鉴权、端点与响应示例、公开状态接口）

## 项目结构

```
ipw-cn/
├── src/               # Go 后端
├── middleware-go/     # 独立组件 转发中间件（Fiber v3，多节点候选）
├── serverless/        # 独立组件 边缘/Serverless（edgeone、edgeone-getip、lemon-getip）
├── frontend-ssr/      # Nuxt 4 SSR 前端
├── docs/              # 文档站
└── install.sh         # 一键安装脚本
```
## 友情链接
### 来自社区的IPW版本
[ZAKOFLARE/ipw-cn-php](https://github.com/ZAKOFLARE/ipw-cn-php) 一个基于PHP的IPW节点   
[ZAKOFLARE/ipw-cn-rust](https://github.com/ZAKOFLARE/ipw-cn-rust) 一个生锈的IPW节点[开发成本过高,已存档]  
[KFCV50TK/ipw-cn](https://github.com/KFCV50TK/ipw-cn) 一个有许多扩展功能的IPW分支  

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
