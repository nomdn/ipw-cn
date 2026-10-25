// GET /api/indexnow?key=<key> —— 把 sitemap 里的全部 URL 提交到 IndexNow。
// 给运维手动触发（curl）或外部监控调用；key 校验失败一律 403。
export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const key = getQuery(event).key
  if (!config.indexnowKey || key !== config.indexnowKey) {
    throw createError({ statusCode: 403, statusMessage: 'invalid key' })
  }
  const siteUrl = config.public.siteUrl
  const urls = await fetchSitemapUrls(siteUrl)
  if (!urls.length) {
    return { submitted: 0, status: 0, body: 'sitemap is empty' }
  }
  return await submitIndexNow(urls, siteUrl)
})
