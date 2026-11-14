import { defineEventHandler, getRequestURL, setHeader, setResponseStatus } from 'h3'

// IndexNow 的站点所有权校验文件：/<key>.txt
//
// 协议要求：提交 URL 时声明 key 与 keyLocation，搜索引擎来抓 keyLocation，
// 比对「文件名」与「文件内容」是否都等于该 key。缺一即 403
// UserForbiddedToAccessSite。
//
// 这个文件不能以静态文件形式提交进仓库（key 由环境变量 NUXT_INDEXNOW_KEY
// 注入：本地 .env、线上 Vercel Environment Variables），所以改成运行时应答：
// 只有「请求的文件名恰好等于配置的 key」时才返回 200 + key 本身，
// 其余一律原样放行（不接管），不会影响 robots.txt 等其它 .txt。
//
// 用 middleware 而不是 server/routes/[key].txt.ts：Nitro 会把该文件名末尾的
// .txt 当成扩展名剥掉，实际注册成 /:key —— 那样会把所有单段路径（/doc、
// /ip-query…）全部吃掉并返回 404，实测确认过，务必别再改回路由写法。
export default defineEventHandler((event) => {
  const path = getRequestURL(event).pathname
  // 只关心 /xxx.txt 这一层形态，其它路径直接放行，零开销
  if (!/^\/[^/]+\.txt$/.test(path)) return

  const configured = useRuntimeConfig(event).indexnowKey
  const requested = path.slice(1, -'.txt'.length)
  // 未配置 key 或文件名对不上：不接管，交给正常流程（静态资源 / 404 页）
  if (!configured || requested !== configured) return

  setHeader(event, 'content-type', 'text/plain; charset=utf-8')
  setResponseStatus(event, 200)
  // IndexNow 只要求内容等于 key，不要求换行；纯 32 字节最干净
  return configured
})
