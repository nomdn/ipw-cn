import { config } from '../config/index'

/**
 * 全站 SEO 的 URL 与分享图收口。
 *
 * 以前各页面直接写 `config.siteUrl + 'dns'`、`${config.siteUrl}favicon.svg`，
 * 甚至自己造一份 siteUrlWithSlash —— 全靠配置末尾恰好有一个斜杠，
 * 一旦有人去掉斜杠就会拼出 `https://nipw.cndns` 这种坏 URL，
 * 而它藏在 canonical / og:image / JSON-LD 里，肉眼很难发现。
 * 这里统一成一处，页面只传路径。
 */

/** 站点基准 origin，保证有且仅有一个尾斜杠 */
export const siteOrigin = config.siteUrl.replace(/\/+$/, '') + '/'

/** 拼站点绝对 URL。path 可带可不带前导斜杠，空串返回首页 */
export function siteUrl(path = ''): string {
  return new URL(String(path).replace(/^\/+/, ''), siteOrigin).href
}

/** 社交分享图：必须是位图且 ≥1200×630，SVG favicon 在微信/QQ/X/FB 一律不显示 */
export const ogImage = siteUrl('og-image.png')

/**
 * 展开到 useHead 的 meta 数组里：`meta: [ ...ogImageMeta ]`。
 * twitter:card 由 app.vue 全局下发为 summary_large_image，
 * 所以 twitter:image 也在这里显式给出，避免回退链上出现 SVG。
 */
export const ogImageMeta = [
  { property: 'og:image', content: ogImage },
  { property: 'og:image:width', content: '1200' },
  { property: 'og:image:height', content: '630' },
  { property: 'og:image:alt', content: `${config.siteName} - 专业 IP 查询与网络测试工具平台` },
  { name: 'twitter:image', content: ogImage },
]
