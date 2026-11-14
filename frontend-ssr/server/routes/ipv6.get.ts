// /ipv6 是历史对外地址（ipw.cn 同款 ?ip= 参数曾大量被外部引用，不能 404）。
// 原实现是一个客户端 onMounted 跳转的空壳页：无 title、无 description、无 h1，
// 被 SEO 审计判为「缺少 H1」页，还以低质量页面的身份混进 sitemap。
// 服务端 301 保证爬虫直接拿到重定向，HTML 空壳页不复存在。
//
// 带 ?ip= 的老外链必须维持原语义（?ip=xxx → /location?ip=xxx）；
// 不带参数的裸 /ipv6 过去也 301 到 /location，等于把这段历史权重
// 全倒给「IP 归属地」页，跟 IPv6 主题无关。现在改指向 /ipv6check 聚合页，
// 让老链接参与 IPv6 检测词族的主题簇。
export default defineEventHandler((event) => {
  const query = getQuery(event)
  if (!('ip' in query)) {
    return sendRedirect(event, '/ipv6check', 301)
  }
  const qs = new URLSearchParams(
    Object.entries(query).flatMap(([k, v]) =>
      v === undefined ? [] : Array.isArray(v) ? v.map((item) => [k, String(item)] as [string, string]) : [[k, String(v)] as [string, string]]
    )
  ).toString()
  return sendRedirect(event, '/location' + (qs ? `?${qs}` : ''), 301)
})
