# 部署指南

### 本章节将一步步指导您自部署柠檬味IPW

Lemon IPW 采用前后端分离架构，**后端节点、独立中间件、前端**三个组件可各自独立部署，并按需组合：

| 组件 | 职责 | 部署方案 |
|------|------|----------|
| [后端节点](/guide/deploy-node) | 提供全部检测 API | EdgeOne / Vercel / Docker / 二进制 |
| [独立中间件](/guide/deploy-middleware) | 请求转发 + Key 注入 | Docker / 二进制 |
| [前端](/guide/deploy-frontend) | 数据可视化与接口调用 | EdgeOne / Vercel / Cloudflare Workers / 直接运行 |

组件之间通过 `/v1/*`（直连后端）与 `/middleware/*`（经中间件转发）两类接口通信，前端按节点列表轮询、失败自动重试下一个节点。

## 快速开始

想用最低成本部署一套功能完整的测试站，直接看 [快速入门](/guide/getting-started)：后端构建 → 前端构建 → 配置文件 → systemd 守护。

## CI/CD

仓库内置 GitHub Actions 流水线，push 到 `main` 时按改动路径自动部署：

| 工作流 | 部署目标 |
|--------|----------|
| `frontend-ssr.yml` | SSR 前端 → Cloudflare Workers |
| `edgeone-backend.yml` | `edgeone/` → EdgeOne Pages |
| `build_and_release.yml` | 后端多平台构建与发布（Release 附带配置文件） |
