/**
 * 顶栏 / 抽屉「宽屏还是窄屏」的唯一判定入口（服务端算一次，客户端沿用）。
 *
 * 为什么客户端不自己嗅探 navigator.userAgent：
 * SSR 出的 HTML 是跟着【服务端】的判定走的。客户端若各算各的，只要两边结论不一致，
 * Vue 水合就会整棵树失配 —— 实测在预渲染的 /doc/** 上是直接抛
 * `Cannot read properties of null (reading 'nodeType')`，然后被 Nuxt 渲染成 500 页
 * （真手机打开文档站必然复现：预渲染时没有请求头 ⇒ 服务端恒判宽屏）。
 * 把服务端的判定序列化进 payload 当客户端的初值，首帧 vdom 与 HTML 必然一致；
 * 顺带还免疫「CDN 把桌面版缓存回给手机」——那种情况下客户端也会乖乖按 HTML 走宽屏。
 *
 * 权威值仍由 app.vue 的 onMounted 用 matchMedia 覆盖，管「桌面窗口被拖窄」这类
 * 请求头判不出的情形。窄屏判定是全局共享状态（useState），文档页据此决定要不要
 * 渲染侧栏，所以两边读的必须是同一个 ref。
 */
const CRAWLER_RE =
  /bot|crawler|spider|slurp|bingbot|baiduspider|yandex|duckduckbot|facebookexternalhit|preview/i

export function uaIsNarrow(ua: string, chMobile?: string) {
  if (!ua && !chMobile) return false // 两个都拿不到（含预渲染）：按宽屏，与改造前一致
  if (CRAWLER_RE.test(ua)) return false // 抓取端一律宽屏，否则顶栏导航链接会从 HTML 里消失
  if (chMobile) return chMobile === '?1' // Chromium 的 Client Hint，最准
  return /Android|iPhone|iPod|Windows Phone|webOS|BlackBerry|Opera Mini|IEMobile|Mobile/i.test(ua)
}

export function useShellNarrow() {
  // 只有服务端拿得到请求头；客户端分支只为类型与「init 被意外调用」兜底。
  const headers: Record<string, string | undefined> = import.meta.server
    ? useRequestHeaders(['user-agent', 'sec-ch-ua-mobile'])
    : {}
  return useState<boolean>('ipw-shell-narrow', () =>
    uaIsNarrow(headers['user-agent'] || '', headers['sec-ch-ua-mobile']),
  )
}
