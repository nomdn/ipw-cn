import { defineSitemapEventHandler } from '#imports'
// 直接 import 构建期常量，而不是走 runtimeConfig.public.docConfig：
// 后者会被 Nuxt 序列化进每个页面的 HTML（4571 字节 / 页），而本路由是全站唯一消费方。
// 相对路径与 server/routes/middleware/[...slug].get.ts 保持一致（`~` 在 Nuxt 4 里指向 app/，指不到根目录的 config/）。
import { docConfig } from '../../../config/doc'


export default defineSitemapEventHandler(async () => {
    const urls = Object.keys(docConfig).map(path => ({ path }));

    return urls.map(url => ({
        loc: url.path,
        _encoded: true,
    }))
})
