// IndexNow 提交的共享实现：server/api/indexnow.get.ts / indexnow.post.ts 与
// scripts/indexnow-submit.mjs（CI 部署后一次性提交）三方语义保持一致。
//
// key 是「站点所有权凭证」而非机密：IndexNow 的校验方式就是抓取
// https://nipw.cn/<key>.txt 并比对，所以 key 必须能公开访问（见 public/ 下的 key 文件）。
// 放进 runtimeConfig 只是为了支持环境变量覆盖（NUXT_INDEXNOW_KEY），默认值与 key 文件一致。
import { useRuntimeConfig } from '#imports'

const INDEXNOW_ENDPOINT = 'https://api.indexnow.org/IndexNow'

export interface IndexNowResult {
  submitted: number
  status: number
  body: string
}

// 归一化 + 同域校验：只允许提交本站 URL（相对路径按 siteUrl 补全），
// 防止提交端点被当作向 IndexNow 提交任意域名的跳板。
export function normalizeIndexNowUrls(input: unknown[], siteUrl: string): string[] {
  const origin = new URL(siteUrl).origin
  const out = new Set<string>()
  for (const raw of input) {
    if (typeof raw !== 'string' || !raw) continue
    const abs = new URL(raw, siteUrl)
    if (abs.origin !== origin) continue
    out.add(abs.href)
  }
  return [...out]
}

// 拉取 sitemap.xml 并抽出全部 <loc>（本站 sitemap 的 XML 很简单，正则足够且免依赖）
export async function fetchSitemapUrls(siteUrl: string): Promise<string[]> {
  const res = await fetch(new URL('sitemap.xml', siteUrl).href)
  if (!res.ok) throw new Error(`sitemap fetch failed: ${res.status}`)
  const xml = await res.text()
  return [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map(m => m[1].trim())
}

export async function submitIndexNow(urls: string[], siteUrl: string): Promise<IndexNowResult> {
  const key = useRuntimeConfig().indexnowKey
  if (!key) throw new Error('indexnowKey is not configured')
  const res = await fetch(INDEXNOW_ENDPOINT, {
    method: 'POST',
    headers: { 'content-type': 'application/json; charset=utf-8' },
    body: JSON.stringify({
      host: new URL(siteUrl).host,
      key,
      keyLocation: new URL(`/${key}.txt`, siteUrl).href,
      // IndexNow 单请求最多 1 万条 URL，超出截断（本站量级远够）
      urlList: urls.slice(0, 10_000),
    }),
  })
  const body = await res.text().catch(() => '')
  return { submitted: urls.length, status: res.status, body: body.slice(0, 200) }
}
