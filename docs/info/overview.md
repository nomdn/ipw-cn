# 柠檬味IPW

柠檬味ipw是一个基于Golang的分布式站点测试平台，旨在构建一个去中心化的，共建贡献的测试平台。  
> [!TIP]
> 尝试一下？这里是官方站点：[链接](https://ipw.wsmdn.top)

## 简介

Lemon IPW（柠檬味 ipw.cn 替代品）是一个开源的分布式站点测试平台。与中心化测试服务不同，本项目鼓励社区成员部署自己的测试节点，前端多节点轮询、后端多地区部署，组成一张去中心化的测试网络，单一节点故障不影响整体可用性。

## 功能

- IP 地址查询（IPv4/IPv6，集成 ip2region、qqwry、MaxMind、GeoCN、DbIP 等多数据源）
- ASN 自治系统查询（MaxMind、DbIP，集成 WHOIS 解析）
- 网站检测（HTTP 状态码、响应时间）
- SSL 证书检查（有效期、颁发机构）
- DNS 解析（A/AAAA/CNAME/MX/TXT/NS/SRV/PTR/CAA 等，多节点并发）
- DNSSEC 验证（DNSKEY、RRSIG、DS 链式信任）
- Whois 查询
- TCPing（IPv4/IPv6 双栈）
- 网站测速
- 网站截图

## 架构

前后端分离，可独立部署、自由组合：

- 前端（Nuxt 4 SSR）负责数据可视化与接口调用
- 后端（Go）提供全部检测 API，可在任意地区部署测试节点，是分布式查询的核心
- 独立中间件（middleware-go）使用Fiber技术栈 负责请求转发和 Key 注入

后端支持二进制、Docker 自托管，也支持 AWS Lambda、EdgeOne Makers、Vercel、阿里云函数计算等 Serverless 平台部署。

## 技术栈

Go、Gin、Nuxt 4、Vue 3、Element Plus。

## 下一步

- 想自己部署一套？前往 [部署指南](/guide/)
- 想了解前端如何部署？阅读 [前端部署](/guide/deploy-frontend)
