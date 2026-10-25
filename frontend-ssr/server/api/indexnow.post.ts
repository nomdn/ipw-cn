// POST /api/indexnow?key=<key>（或请求头 x-indexnow-key），body: { urls: string[] }
// —— 精确提交新增/变更的 URL（发布新文档、工具页上线等场景）。
// URL 限本站域名（相对路径按 siteUrl 补全），跨域 URL 直接过滤。
export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const key = getQuery(event).key || getHeader(event, 'x-indexnow-key')
  if (!config.indexnowKey || key !== config.indexnowKey) {
    throw createError({ statusCode: 403, statusMessage: 'invalid key' })
  }
  const body = await readBody<{ urls?: unknown[] }>(event).catch(() => null)
  const urls = normalizeIndexNowUrls(body?.urls ?? [], config.public.siteUrl)
  if (!urls.length) {
    throw createError({ statusCode: 400, statusMessage: 'no valid urls (same host only)' })
  }
  return await submitIndexNow(urls, config.public.siteUrl)
})
