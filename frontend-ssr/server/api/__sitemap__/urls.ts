import { defineSitemapEventHandler } from '#imports'
// 直接 import 构建期常量，而不是走 runtimeConfig.public.docConfig：
// 后者会被 Nuxt 序列化进每个页面的 HTML（4571 字节 / 页），而本路由是全站唯一消费方。
// 相对路径与 server/routes/middleware/[...slug].get.ts 保持一致（`~` 在 Nuxt 4 里指向 app/，指不到根目录的 config/）。
import { docConfig } from '../../../config/doc'

// 显式枚举全部工具页并标注权重：与 nuxt:pages 自动扫描并存（模块会按 loc 去重合并），
// 好处是收录不依赖扫描行为、且带 priority/changefreq。
// 注意 /ipv6 不在列表里：它已改为 server/routes/ipv6.get.ts 的服务端 301，不再是页面。
const TOOL_PAGES: Array<{ loc: string; priority: number; changefreq: string }> = [
    { loc: '/', priority: 1.0, changefreq: 'daily' },
    { loc: '/location', priority: 0.9, changefreq: 'weekly' },
    { loc: '/asn', priority: 0.9, changefreq: 'weekly' },
    { loc: '/dns', priority: 0.9, changefreq: 'weekly' },
    { loc: '/dnssec', priority: 0.9, changefreq: 'weekly' },
    { loc: '/whois', priority: 0.9, changefreq: 'weekly' },
    { loc: '/ssl', priority: 0.9, changefreq: 'weekly' },
    { loc: '/tcping', priority: 0.9, changefreq: 'weekly' },
    { loc: '/speedtest', priority: 0.9, changefreq: 'weekly' },
    { loc: '/ipv6speedtest', priority: 0.9, changefreq: 'weekly' },
    { loc: '/ipv6tcping', priority: 0.9, changefreq: 'weekly' },
    { loc: '/ipv6webcheck', priority: 0.9, changefreq: 'weekly' },
    { loc: '/screenshot', priority: 0.8, changefreq: 'weekly' },
]

export default defineSitemapEventHandler(async () => {
    const toolPages = TOOL_PAGES.map(({ loc, priority, changefreq }) => ({ loc, priority, changefreq }))
    // 文档页权重：索引页 /doc 高一档，正文页低一档
    const docPages = Object.keys(docConfig).map(path => ({
        loc: path,
        priority: path === '/doc' ? 0.8 : 0.7,
        changefreq: 'monthly',
    }))

    return [...toolPages, ...docPages]
})
