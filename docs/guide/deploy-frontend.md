# 前端部署

前端（`frontend-ssr/`）是全项目的核心，负责数据可视化与接口调用。基于 **Nuxt 4 SSR** 构建。

## 架构与部署目标

| 项 | 说明 |
|----|------|
| 技术栈 | Nuxt 4 + Vue 3 + Element Plus + VueUse |
| 运行时 | Node.js 22（见 `.node-version`）、包管理器 pnpm 10 |
| 构建产物 | `.output/server/index.mjs` + `.output/public/`（静态资源） |
| 构建命令 | `pnpm build`（Nitro SSR 构建） |

## 准备工作

```bash
# 1. 进入前端目录
cd frontend-ssr

# 2. 安装依赖（需要 Node 22 + pnpm 10）
pnpm install

# 3. 本地开发
pnpm dev
```

## 配置说明

前端所有配置集中在 `frontend-ssr/config/index.ts`，完整配置项见 [配置文件 - 前端](/guide/config#前端-configindexts)：

| 配置项 | 说明 |
|--------|------|
| `siteUrl` | 站点对外地址 |
| `siteName` | 站点名称（页面标题 / 描述 / 页脚品牌） |
| `APIBaseURL` | 拨测上游节点池（whois / ssl / detail / dns / dnssec / tcping / speed），含 `DualStack` / `IPv4` / `IPv6` 三栈 |
| `IPLocationAPI` | IP 归属地 / ASN 上游节点池（纯数组，无栈区分） |
| `Middleware` | 外部独立中间件 base URL 列表，`/middleware/*` 请求依次尝试、失败重试下一个 |
| `EnableInternalMiddleware` | 是否启用前端内置中间件（本地转发，作为候选列表最后一位兜底），默认 `true` |
| `rateLimitPerMinute` | 内置中间件单 IP 限流次数（次/分钟），默认 `120`，`0` 表示不限流 |
| `v4OnlyAPI` / `v6OnlyAPI` / `DualStackAPI` | 出站 IP 检测接口（页面直连建议使用带 CORS 的 wsmdn.top 接口） |
| `umamiHost` 等 | Umami 统计配置 |
| `ICP` / `GongAn` | 网站备案号（页脚展示） |
| `noindex` | 是否禁止搜索引擎索引 |

> 生产环境建议通过运行时的环境变量或部署平台配置覆盖敏感项，不要把密钥写进 `config/index.ts` 提交到仓库。

## 部署方案

### 方案一：EdgeOne

腾讯云 EdgeOne 原生支持 Nuxt，点击下方按钮即可通过 **EdgeOne Makers** 一键部署（无需手动配置构建）：

[![使用 EdgeOne Makers 部署](https://cdnstatic.tencentcs.com/edgeone/pages/deploy.svg)](https://console.cloud.tencent.com/edgeone/makers/new?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=frontend-ssr&install-command=pnpm%20install&build-command=pnpm%20run%20build&output-directory=.output)



### 方案二：Vercel

点击下方按钮一键导入并部署到 Vercel（自动识别 Nuxt 项目，`root-directory` 已预置为 `frontend-ssr`）：

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2Fnomdn%2Fipw-cn&root-directory=frontend-ssr&install-command=pnpm%20install&build-command=pnpm%20run%20build&output-directory=.output)
### 方案三：Cloudflare Workers

使用 wrangler 手动部署：

```bash
cd frontend-ssr
pnpm build
pnpm deploy   # 等价于 pnpm build && wrangler deploy
```

部署前需要配置 wrangler 认证（二选一）：

```bash
# 方式一：环境变量（CI 中使用）
export CLOUDFLARE_API_TOKEN=your_api_token
export CLOUDFLARE_ACCOUNT_ID=your_account_id

# 方式二：wrangler login（本地交互登录）
npx wrangler login
```

> Cloudflare 部署按钮不支持子目录，而前端位于仓库 `frontend-ssr/` 子目录，因此不提供一键部署按钮；使用上方 wrangler 命令或仓库 CI/CD 工作流部署。

### 方案四：直接运行（Node 自托管）

前端是 Node.js 应用，构建产物为 JS 脚本 + 静态资源（非二进制），可直接用 Node 运行：

```bash
cd frontend-ssr
pnpm build
node .output/server/index.mjs   # 默认监听 3000 端口
```

生产环境建议使用进程守护（systemd / pm2 等）保持常驻。

## CI/CD 自动部署

仓库内置工作流 `.github/workflows/frontend-ssr.yml`：

- **触发条件**：push 到 `main` 且改动路径为 `frontend-ssr/**`；或手动 `workflow_dispatch`
- **步骤**：checkout → Node 22 + pnpm 10 → `pnpm install` → `pnpm run deploy --keep-vars`
- **所需 Secrets**：`CLOUDFLARE_API_TOKEN`、`CLOUDFLARE_ACCOUNT_ID`
- **注意**：工作流设置了 `NITRO_PRESET: cloudflare_module`

## 常见问题

- **部署后接口 403 / 跨域**：检查后端 `cors` 配置（独立中间件见 `middleware-go/setting.json` 的 `cors` 字段，逗号分隔允许域名）；服务端转发请求不带 `Origin`，浏览器直接调用才受 CORS 限制。
- **出站 IP 检测接口被浏览器拦截**：页面直连的 IP 检测接口需要 CORS，详见 [配置文件 - 出站 IP 检测接口](/guide/config#出站-ip-检测接口v4onlyapi--v6onlyapi--dualstackapi)。
