#!/usr/bin/env node
// CI 部署完成后一次性提交 sitemap 全部 URL 到 IndexNow（Bing / Yandex / Seznam / Yep 共用端点）。
// 用法：node scripts/indexnow-submit.mjs --site https://nipw.cn --key <key>
//
// key 必须与 frontend-ssr/public/<key>.txt 完全一致 —— IndexNow 的所有权校验就是抓取
// 这个文件比对，所以 key 本身是公开凭证、写在 CI 里并不泄密。
// 逻辑与 server/utils/indexnow.ts 保持一致（那边是运行时端点，这边是构建机直跑）。

const args = process.argv.slice(2)
function argOf(name) {
  const i = args.indexOf(name)
  return i >= 0 ? args[i + 1] : undefined
}

const site = (argOf('--site') || 'https://nipw.cn').replace(/\/$/, '')
const key = argOf('--key')

if (!key) {
  console.error('usage: node scripts/indexnow-submit.mjs --site <url> --key <key>')
  process.exit(1)
}

const INDEXNOW_ENDPOINT = 'https://api.indexnow.org/IndexNow'
const sitemapUrl = `${site}/sitemap.xml`

const res = await fetch(sitemapUrl)
if (!res.ok) {
  console.error(`sitemap fetch failed: ${res.status} ${sitemapUrl}`)
  process.exit(1)
}
const xml = await res.text()
const urls = [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)].map(m => m[1].trim())
if (!urls.length) {
  console.error('sitemap has no <loc> entries, nothing to submit')
  process.exit(1)
}

// IndexNow 单请求最多 1 万条，分片兜底
const CHUNK = 10_000
let ok = 0
for (let i = 0; i < urls.length; i += CHUNK) {
  const urlList = urls.slice(i, i + CHUNK)
  const r = await fetch(INDEXNOW_ENDPOINT, {
    method: 'POST',
    headers: { 'content-type': 'application/json; charset=utf-8' },
    body: JSON.stringify({
      host: new URL(site).host,
      key,
      keyLocation: `${site}/${key}.txt`,
      urlList,
    }),
  })
  const body = await r.text().catch(() => '')
  console.log(`IndexNow chunk ${i / CHUNK + 1}: HTTP ${r.status} (${urlList.length} urls) ${body.slice(0, 120)}`)
  if (r.ok || r.status === 202) ok += urlList.length
}
console.log(`submitted ${ok}/${urls.length} urls from ${sitemapUrl}`)
// IndexNow 202 = 已受理；4xx 校验失败会让 CI 亮红，属预期失败信号
if (!ok) process.exit(1)
