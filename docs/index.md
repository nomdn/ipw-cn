---
# 首页（VitePress 标准布局，已在 config 中设置 vpHome: true / teekHome: false）
layout: home

hero:
  name: "LEMON IPW"
  text: "LEMON IPW官方文档"
  tagline: IP 查询 · 网站检测 · SSL · DNS · DNSSEC · Whois · ASN · 截图
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/nomdn/ipw-cn
      target: _blank

features:
  - title: 多源 IP 查询
    details: 支持 IPv4/IPv6，集成 ip2region、qqwry、MaxMind GeoIP、GeoCN、DbIP、Bilibili 等多数据源，覆盖国内外归属地。
  - title: ASN 与 Whois
    details: 查询 IP 所属自治系统号（多数据源 + WHOIS 解析），以及域名注册信息、注册商、到期时间等。
  - title: 网站检测全家桶
    details: HTTP 状态码与响应时间、SSL 证书有效期与颁发机构、TCPing 延迟、下载测速、在线页面截图一站式完成。
  - title: DNS 解析
    details: 多节点并发查询，支持 A / AAAA / CNAME / MX / TXT / NS / SRV / PTR / CAA 等记录类型，并提供 DNSSEC 链式信任验证。
  - title: SSRF 防护
    details: 自动拦截对内网 / 私有 IP 的出站请求，校验安全跳转，避免服务端请求伪造风险。
  - title: 多平台部署
    details: 支持二进制，Docker，乃至AWS Lambda，EdgeOne Makers，Vercel和阿里云函数计算
---
