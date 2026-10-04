import { brotliCompressSync, gzipSync, constants } from 'node:zlib'
import type { NitroApp } from 'nitro/types'

/**
 * SSR HTML 压缩。
 *
 * ## 为什么要压
 *
 * nitro 不压 SSR 响应体（静态资源另有 compressPublicAssets 预压缩），
 * 本地直连 / Lighthouse 测到的 HTML 就是裸传的。生产链路前面有 EdgeOne 兜底，
 * 但源站自己压是标准做法，也让本地测量贴近线上。
 *
 * 只压 SSR 渲染出的 text/html：body 是字符串、体积 > 1KB、对方 Accept-Encoding
 * 支持才压，按客户端优先级选 br（brotli）否则 gzip。
 *
 * ## 为什么必须显式指定 brotli quality = 4（别删这行）
 *
 * `brotliCompressSync` 不带 params 时默认 **quality 11**，那是给「离线压一次、
 * 反复复用」准备的（压一次存盘，读一万次）。SSR 是**每请求现压**：
 *
 *   quality    33KB HTML      68KB HTML
 *   11         123ms / 7.2KB  169ms / 11.0KB
 *   6            4ms / 7.8KB    5ms / 12.2KB
 *   4            2ms / 8.6KB    3ms / 13.5KB
 *
 * Node 是单线程，这段同步压缩**全程阻塞事件循环，直接进 TTFB**。
 * 首页 TTFB 实测：q11 时 225~260ms，改 q4 后 55~105ms —— 差的就是这 120~170ms。
 * q4 只多 2.5KB，在 1.6Mbps 上是 12ms，换回 166ms，净赚。
 *
 * 教训：凡「每请求现压」的同步压缩都要显式指定质量，别吃库默认值。
 */
export default defineNitroPlugin((nitroApp: NitroApp) => {
  nitroApp.hooks.hook('render:response', (response, { event }) => {
    const headers = (response.headers ??= {})
    const type = String(
      headers['content-type'] ?? event.node.res.getHeader('content-type') ?? '',
    )
    if (!type.includes('text/html')) return
    if (typeof response.body !== 'string' || response.body.length < 1024) return
    if (headers['content-encoding']) return // 已有编码（异常路径），不叠加

    // —— modulepreload 降优先级（2026-10-04 实测，mobile LH 三档对比后选定 low）——
    // entry 的 26 个静态依赖 chunk 默认 High 优先级，和 4 个 render-blocking CSS 一起
    // 排队（本地 H1 只有 6 个连接），CSS 被挤到后面 ⇒ Lantern 模拟 FCP 2345ms。
    // 加 fetchpriority="low" 后 CSS 先行：FCP 2345→1109（-1236ms，首次过 1800 满分线），
    // desktop 三轮全 100。代价是 JS 下载推迟 → hydration 落到 FCP 之后，TBT 86→213。
    // 三档实测（mobile perf）：无 fp 94/95，medium 92/93（medium 仍挤 CSS，FCP 2253，
    // 收益尽失），low 94/95 —— low 总分最优。
    // TBT 的 213 是这项改动的全部代价：FCP 提前后，原本在 FCP 前不计账的 hydration
    // 长任务进了 TBT 窗口（FCP 每提前 ~500ms，TBT 约 +55，两者在当前架构下互斥）。
    if (response.body.includes('rel="modulepreload"')) {
      response.body = response.body.replaceAll(
        'rel="modulepreload"',
        'rel="modulepreload" fetchpriority="low"',
      )
    }

    const accept = String(event.node.req.headers['accept-encoding'] ?? '')
    let encoding: 'br' | 'gzip'
    if (/\bbr\b/.test(accept)) {
      response.body = brotliCompressSync(Buffer.from(response.body), {
        params: { [constants.BROTLI_PARAM_QUALITY]: 4 },
      })
      encoding = 'br'
    } else if (/\bgzip\b/.test(accept)) {
      response.body = gzipSync(Buffer.from(response.body))
      encoding = 'gzip'
    } else {
      return
    }

    headers['content-encoding'] = encoding
    // 体积变了，content-length 必须跟着改，否则连接会被截断/挂起
    headers['content-length'] = String((response.body as Buffer).length)
  })
})
