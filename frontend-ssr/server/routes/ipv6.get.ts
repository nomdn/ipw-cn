// /ipv6 是历史对外地址（ipw.cn 同款 ?ip= 参数曾大量被外部引用，不能 404）。
// 原实现是一个客户端 onMounted 跳转的空壳页：无 title、无 description、无 h1，
// 被 SEO 审计判为「缺少 H1」页，还以低质量页面的身份混进 sitemap。
// 这里改为服务端 301：爬虫直接拿到重定向，HTML 空壳页不复存在，
// query 原样带到 /location（?ip=xxx → /location?ip=xxx）。
export default defineEventHandler((event) => {
  const query = getQuery(event)
  const qs = new URLSearchParams(
    Object.entries(query).flatMap(([k, v]) =>
      v === undefined ? [] : Array.isArray(v) ? v.map((item) => [k, String(item)] as [string, string]) : [[k, String(v)] as [string, string]]
    )
  ).toString()
  return sendRedirect(event, '/location' + (qs ? `?${qs}` : ''), 301)
})
