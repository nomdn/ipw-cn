import { defineEventHandler, getRequestURL, sendRedirect } from 'h3'

// /doc/** 早期是 SSG，产物形如 /doc/xxx/index.html，对外 URL 带尾斜杠。
// 改成 SSR 后正式路径不再带尾斜杠，带斜杠的请求会落到 404 页：404 页有 title
// 但没有 meta description，Bing 站长工具因此把这 8 个旧文档 URL 判成
// 「页面的标头部分缺少说明」，历史外链的权重也全耗在 404 上。
//
// 统一 301 到无尾斜杠版本（保留 query）。canonical 和 sitemap 本来就是
// 无尾斜杠形式，这里只收敛入口，不改任何页面内容与 head。
export default defineEventHandler((event) => {
  const url = getRequestURL(event)
  const path = url.pathname
  if (path === '/' || !path.endsWith('/')) return
  // 目录式静态资源（如 /doc/img.png/）不参与规范化，避免误伤
  if (/\.[a-z0-9]{2,5}\/$/i.test(path)) return
  return sendRedirect(event, path.replace(/\/+$/, '') + url.search, 301)
})
