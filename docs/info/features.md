# 功能总览

后端节点（`src/`，Go + Gin）承担全部检测能力，可在任意地区部署组成去中心化测试网络。接口清单见 [节点 API 与协议参考](/guide/node-api)。

## 探测能力矩阵

| 能力 | 路由 | 依赖 | 说明 |
|------|------|------|------|
| 网站探测明细 | `/v1/detail/*url` | 出站 HTTP（v4/v6 双客户端） | DNS 解析、TCP 连接、TLS 握手、首字节、总耗时、状态码、页面体积、下载速度、可达性 |
| SSL 证书检查 | `/v1/ssl/*url` | TLS 握手 | 证书有效期/剩余天数、颁发机构、主体、TLS 版本、是否过期 |
| 网站测速 | `/v1/speed/:version/*url` | 出站下载 | 指定栈的下载速度测试 |
| TCPing | `/v1/tcping/:ip` | TCP 拨号 | 端口连通性 + RTT 统计（次数可配） |
| DNS 解析 | `/v1/dns/:type/*domain` | `dns-server` | 9 种记录类型；DNS 服务器支持主从 failover 与 DoH |
| DNSSEC 验证 | `/v1/dnssec/:domain` | `dnssec-server` | DNSKEY / RRSIG / DS 链式信任校验 |
| Whois | `/v1/whois/:domain` | 上游 whois 服务器 | 域名注册商、注册/到期时间等 |
| IP 归属地 | `/v1/location`、`/v1/location/:ip` | IP 数据库（10 源） | 多源聚合，含国家/省/市/ISP/经纬度/时区 |
| ASN 查询 | `/v1/asn/:ip` | 3 套 ASN 库 + Whois | 自治系统号与所属组织 |

**双栈**：除 IP 归属地/ASN 外的能力均按 IPv4 / IPv6 分别执行并同时返回（`single-stack` 可声明单栈节点以跳过另一栈）。
**统计口径**：所有 `/v1/*` 请求经统计中间件计数，由节点按周期上报收集中心（见 [节点 API · 数据上报](/guide/node-api)）。

## IP 数据源

`ipdb` 未关闭时启动加载，共 10 个源；任一本地文件缺失即从远端拉取全部数据库（首次约 450MB，之后每 24h + 随机抖动更新一次）。

| 源 | 用途 | 形式 |
|----|------|------|
| ip2region | 国家 / 省 / 市 / ISP | 本地 xdb（v4 + v6） |
| IP2Location LITE DB11 | 国家 / 省 / 市 / ISP / 经纬度 / 时区 | 本地 BIN |
| IP2Location LITE ASN | ASN | 本地 BIN |
| qqwry | 国家 / 省 / 市 / ISP / 国家码 | 本地 ipdb |
| MaxMind GeoLite2 City | 国家 / 省 / 市 / 经纬度 | 本地 MMDB |
| MaxMind GeoLite2 ASN | ASN / 组织 | 本地 MMDB |
| DbIP City Lite | 国家 / 省 / 市 / 经纬度 | 本地 MMDB |
| DbIP ASN Lite | ASN / 组织 | 本地 MMDB |
| GeoCN | 中国行政区划码 + ISP + 类型 | 本地 MMDB + 行政区划表 |
| Bilibili | 国家 / 省 / 市 / ISP / 经纬度 | **在线 API**，24h 缓存 |

- `/v1/location` 聚合全部 10 源（结果按源分组返回，便于交叉比对）；`/v1/asn` 只查三套 ASN 库并对 MaxMind ASN 做 Whois 解析。
- `gh-proxy` 可给 GitHub 下载加代理前缀加速；GeoCN 走 jsDelivr CDN。
- `ipdb=false` 时整体关闭：不加载数据库，`/v1/location`、`/v1/asn` 路由不注册，适合只做网络拨测的轻量节点。

## 可靠性设计

- **结果缓存**：7 类缓存（6 个业务缓存 5 分钟 + Bilibili 24 小时），失败结果 30s 后即失效；每 10 分钟定时清扫过期条目。详见 [节点 API · 缓存与并发](/guide/node-api)。
- **singleflight**：`detail` / `ssl` / `tcping` / `speed` 的并发同 key 请求合并为一次真实探测。
- **SSRF 防护**（`block-private-ips`）：拒绝出站连接私有/内网地址，探测目标命中私有网段时直接返回伪造结果，禁止跨跳重定向；`trusted-proxies` 控制 `X-Forwarded-For` 信任范围，防止访客伪造来源 IP。
- **DNS 兜底**：`dns-server` / `dnssec-server` 留空时自动探测系统 DNS，避免容器环境下解析不可用。

## 与收集中心 / 中间件的关系

```
                    ┌───────────────────────────────┐
   浏览器 / 前端 ───▶│  中间件（ipw-cn/middleware-go │
                    │  或 ipw-boce 收集中心）        │
                    └───────┬───────────────┬───────┘
                     HTTP   │               │  WS（/ws）
                    转发    ▼               ▼
                    ┌───────────────────────────────┐
                    │        后端节点（本仓库）      │
                    │  执行探测 → 结果原路回传        │
                    │  统计/明细 → 周期上报收集中心   │
                    └───────────────────────────────┘
```

- **转发链路**：前端不直连节点，而是经中间件按池轮询转发；节点也可以直接对外提供 `/v1/*`（自建纯节点模式）。
- **WS 通道**：节点作为客户端接入中间件，拨测请求经 WS 下发、结果经 WS 回传；同时承载配置管理（`config`）与升级（`ota`）指令。
- **数据上报**：统计与拨测明细只由节点上报，中间件不在转发路径上采集，避免双算。
- **版本与能力上报**：节点上报版本号与能力清单，收集中心据此展示版本、判断能否下发管理指令。

## 部署形态

| 形态 | 说明 |
|------|------|
| 二进制 / systemd | `install.sh` 一键安装，配置以环境变量注入 |
| Docker | `src/Dockerfile` 多阶段构建，多架构（amd64 / arm64 / armv7）；镜像不含 IP 库，首次运行现场拉取 |
| Serverless | EdgeOne Makers、Vercel、AWS Lambda、阿里云函数计算等（均为 IPv4-only，需声明 `SINGLE_STACK=ipv4`） |

各形态的详细步骤见 [后端节点部署](/guide/deploy-node)。
