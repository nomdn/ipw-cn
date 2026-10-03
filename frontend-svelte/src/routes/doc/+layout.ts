/**
 * /doc 站点整棵子树改为 SSG（构建期预渲染），对齐旧站 nuxt.config.ts 的
 * `routeRules: { '/doc/**': { prerender: true } }`。
 *
 * 为什么可以、也应该预渲染：
 *   - 文档正文是 `import.meta.glob('...?raw', { eager: true })` 在**构建期**打进
 *     server bundle 的（见 src/lib/server/docs.ts），运行时本来就没有任何动态数据；
 *   - meta 来自静态映射表 config/doc.ts；
 *   所以预渲染不引入任何额外"内容会过期"的风险，纯粹省掉源站的 SSR 开销。
 *
 * 路由枚举不写死：DocMenu 把 32 条文档链接全部渲染进 HTML（收起的分组只是
 * `hidden`，不摘 DOM），Kit 的预渲染爬虫顺着这些 <a href> 就能把整棵子树抓全。
 * ⚠️ 这也是为什么 DocMenu 里**不能**用 `{#if opened}` 来折叠分组 —— 那样爬虫
 * 只能看到一个没有链接的按钮，整棵文档目录会被漏掉。
 *
 * 副作用（与旧站一致）：预渲染页由 CDN/ASSETS 直接吐出，`hooks.server.ts` 的
 * 安全响应头不再经过；CSP 改由 Kit 以 `<meta http-equiv="content-security-policy">`
 * 的形式内联（vite.config.ts 里 csp.mode='auto' 对预渲染页走 hash）。
 */
export const prerender = true;
