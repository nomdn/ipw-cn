import { brotliCompressSync, gzipSync } from 'node:zlib'
import type { NitroApp } from 'nitro/types'

/**
 * SSR HTML 压缩。
 *
 * 为什么要有它：nitro 不压 SSR 响应体（静态资源另有 compressPublicAssets 预压缩），
 * 本地直连 / Lighthouse 测到的 HTML 就是裸传的 —— Lantern 会按无压缩体积去模拟
 * 低速网络，把 FCP/LCP/SpeedIndex 放大好几倍（实测 836ms 模拟成 3.8s）。
 * 生产链路前面有 EdgeOne 兜底，但源站自己会压是标准做法，也让本地测量贴近线上。
 *
 * 只压 SSR 渲染出的 text/html：body 是字符串、体积 > 1KB、对方 Accept-Encoding 支持才压，
 * 按客户端优先级选 br（brotli，文本比 gzip 再小 ~15%）否则 gzip。
 */
export default defineNitroPlugin((nitroApp: NitroApp) => {
  nitroApp.hooks.hook('render:response', (response, { event }) => {
    const headers = (response.headers ??= {})
    const type = String(
      headers['content-type'] ?? event.node.res.getHeader('content-type') ?? '',
    )
    if (!type.includes('text/html')) return
    const body = response.body
    if (typeof body !== 'string' || body.length < 1024) return
    if (headers['content-encoding']) return // 已有编码（异常路径），不叠加

    const accept = String(event.node.req.headers['accept-encoding'] ?? '')
    let encoding: 'br' | 'gzip'
    if (/\bbr\b/.test(accept)) {
      response.body = brotliCompressSync(Buffer.from(body))
      encoding = 'br'
    } else if (/\bgzip\b/.test(accept)) {
      response.body = gzipSync(Buffer.from(body))
      encoding = 'gzip'
    } else {
      return
    }

    headers['content-encoding'] = encoding
    // 体积变了，content-length 必须跟着改，否则连接会被截断/挂起
    headers['content-length'] = String((response.body as Buffer).length)
  })
})
