import {config} from "./config/index";
// https://nuxt.com/docs/api/configuration/nuxt-config
const extractDomains = (obj: any): string[] => {
  // 将对象转为 JSON 字符串，用正则匹配所有 https:// 开头的域名部分
  const urls = JSON.stringify(obj).match(/https?:\/\/[^"\/\\\s]+/g) || [];
  // 提取域名 (Origin) 并去重
  const domains = [...new Set(urls.map(url => new URL(url).origin))];
  return domains;
};

const allowedDomains = extractDomains(config);

// 首屏「IPv4 / IPv6 / 优先级」三行的数据源（见 app/pages/index.vue 的 onMounted）：
// 三个接口分属三个独立源站，且都要等水合完成后才发起 —— 那时才做 DNS+TCP+TLS 太晚。
// 把它们抽出来做 preconnect，让握手在 HTML 解析期就开始。
const ipProbeOrigins = [
  config.DualStackAPI,
  config.v4OnlyAPI,
  config.v6OnlyAPI,
].map((url) => new URL(url).origin);
// 中间件（工具页 /middleware/* 的上游）：只在部分页面用到，DNS 预解析即可，不占连接。
const middlewareOrigin = config.Middleware?.[0]
  ? new URL(config.Middleware[0]).origin
  : null;

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  experimental: {
    // 关闭 app manifest 版本检查：水合后 Nuxt 会在 requestIdleCallback 里请求
    // /_nuxt/builds/meta/<buildId>.json，buildId 对不上（本地「服务旧、磁盘新」、
    // 或线上发版切换瞬间）就抛 NUXT_E5002 污染 console，还拖累 Best Practices 分。
    // 本项目没依赖 appManifest 的客户端能力（无客户端 routeRules、无 prerender
    // 判定），关掉少一个请求，console 永远干净。
    appManifest: false,
    // 注：inlineStyles: true 实测是负优化——它只内联 page 组件样式（+12KB HTML），
    // entry/EP 等共享样式仍是 render-blocking 外链，关键请求一个没省。勿开。
    defaults: {
      nuxtLink: {
        // NuxtLink 默认「链接进入视口就预取目标页 JS」——顶栏 15 个菜单项会让
        // 首屏把其它工具页的 chunk（含 87KB 的校验库等重依赖）全拉下来，
        // 白吃带宽还挤占首屏关键请求。改为「悬停/聚焦才预取」：真正要去
        // 哪个页面时再拉那一页，首屏只加载当前页需要的代码。
        prefetchOn: { interaction: true, visibility: false },
      },
    },
  },
  modules: [
    "nitro-cloudflare-dev",
    '@element-plus/nuxt',
    '@nuxtjs/sitemap',
    '@nuxtjs/robots',
    '@vueuse/nuxt',
    "nuxt-security",
  ],
  vite: {
    optimizeDeps: {
      include: [
        'is-ip',
        'shiki',
        'dayjs',
        'lodash-unified'
      ]
    },
  },
  hooks: {
    // 掐掉「当前路由的动态 import 预取」。
    // Nuxt 的 vue-bundle-renderer 会把 manifest 里带 prefetch 标记的动态 chunk
    // 渲染成 `<link rel="prefetch" as="script">`，于是首页一边解析 HTML 一边就把
    // shiki 高亮器那一整块（实测 165 KB，占首页字节 25%）下下来了 —— 而高亮已经在
    // SSR 期算完写进 HTML，客户端根本不会执行它，纯属白下。
    // 把 manifest 上的标记抹掉即可：动态 chunk 回到「真正 import 时才加载」。
    // 静态闭包走的是 preload 分支（modulepreload），不受影响。
    // —— 2026-10-04 曾试把 preload 也置 false：FCP 确实 2333→1404（少了 22 个请求的
    // 下载竞争），但 LCP 2387→2933、TBT 86→280（权重 25%/30% 远大于 FCP 的 10%），
    // 总分 94→89 净亏。原因：这 22 个 modulepreload 是 entry 的静态依赖闭包，全掐掉后
    // hydration 推迟、长任务落进 LCP/TBT 窗口。故只掐 prefetch，preload 必须留。
    'build:manifest': (manifest) => {
      for (const chunk of Object.values(manifest) as unknown as Array<Record<string, unknown>>) {
        if (chunk.prefetch) chunk.prefetch = false
      }
    },
  },
  site: { 
  url: config.siteUrl, 
  name: 'Lemon IPW' 
  },
  // Element Plus 样式走按需引入：@element-plus/nuxt 默认 importStyle: 'css'，
  // 模板里用到的每个组件都会各自带上 `element-plus/es/components/<组件>/style/css`
  // （其内部再链到 theme-chalk/base.css，`:root` 的 --el-* 变量就在那里定义）。
  // 所以这里**不要**再引 element-plus/dist/index.css —— 那是整库样式（原始 360KB），
  // 而首页实测 98% 的规则用不到，却全压在关键路径上。
  css: [
    // 暗黑模式的 CSS 变量覆盖（html.dark 下的 --el-* 值），按需引入不含它，必须显式引
    'element-plus/theme-chalk/dark/css-vars.css',
  ],
  app:{
    head: {
      // 站点内容为简体中文。不写时 Nuxt 输出的就是裸 <html>，
      // 屏幕阅读器会按系统语言瞎猜发音，Lighthouse 的 html-has-lang 也会失败。
      htmlAttrs: {
        lang: 'zh-CN',
      },
      script: [
        {
          // 必须 innerHTML，不能 src（否则异步加载）
          innerHTML: `
            (function() {
              var stored = localStorage.getItem('vueuse-color-scheme');
              var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
              var dark = stored === 'dark' || (!stored && prefersDark);
              if (dark) document.documentElement.classList.add('dark');
            })();
          `,
          // 关键：不加 async/defer，确保同步阻塞执行
        }
      ],
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.svg' },
        // preconnect 三个 IP 探测源站。crossorigin 不能省：$fetch 走 fetch() 的跨域
        // （CORS）模式，不带它预热出来的是 no-cors 连接，真正的 fetch 用不上、还得重连。
        ...ipProbeOrigins.map((href) => ({
          rel: 'preconnect',
          href,
          crossorigin: 'anonymous',
        } as const)),
        ...(middlewareOrigin
          ? [{ rel: 'dns-prefetch', href: middlewareOrigin } as const]
          : []),
      ]
    }
  },
  runtimeConfig: {
    indexnowKey: '',
    apiKeys: '',
    public: {
      siteUrl: config.siteUrl,
      // 这里**不要**再放 docConfig：runtimeConfig.public 会被整份序列化进
      // 每个页面 HTML 的 `window.__NUXT__.config`，而这份文档树（32 条 title+description）
      // 实测占 4571 字节 —— 也就是每一页的 HTML 都白白多背 4.5KB（首页 HTML 的 10%）。
      // 它全站只有一个消费方，且是服务端路由（server/api/__sitemap__/urls.ts），
      // 那里改成直接 import 构建期常量即可；文档页本来就用 `getDocMeta` 直接 import。
    },
  },
  routeRules: {
    // 缓存头一律不在这里设，交由 CDN（EdgeOne）侧控制。
    // /doc/** 曾经是 SSG（prerender），已撤回改为 SSR，原因见下方 nitro.prerender 处的注释。
  },
  nitro: {
    // 构建期给 public/ 静态资源生成 .gz/.br 预压缩副本，按 Accept-Encoding 直接回；
    // SSR HTML 的压缩在 server/plugins/compress-html.ts。两者合起来让本地直连的
    // 传输体积与线上（EdgeOne 会压）一致，Lighthouse 的 Lantern 模拟才不会虚高。
    compressPublicAssets: true,
    publicAssets: [
      {
        dir: 'public',
        maxAge: 0
      }
    ],
    esbuild: {
      options: {
        target: 'es2022' // 明确告诉 Nitro 使用 es2022 进行打包
      }
    },
    // 这里曾用 prerender.routes 把整份 docConfig 的文档页在构建期预渲染成静态 HTML。
    // 已撤回，改回 SSR —— 因为预渲染与「按请求头决定渲染形态」根本冲突：
    // 构建期没有请求头，窄屏判定恒为宽屏，于是预渲染产物里永远躺着
    // 「15 条桌面顶栏菜单 + 被 CSS 隐藏的 40 项文档侧栏」；真手机打开时客户端判定为窄屏，
    // 首帧 vdom 与 HTML 对不上，Vue 水合直接抛
    // `Cannot read properties of null (reading 'nodeType')`，整页变成 500。
    // 顺带收益：/doc 现在也走运行时安全头（预渲染页拿不到 CSP，见下方 security.ssg）。
    // 代价只是源站多跑一层 SSR，前面有 EdgeOne 缓存兜着。
  },
  security: {
    ssg: {
      // 预渲染页的 CSP 会因 SRI modulepreload 哈希全量拼入而单行超 2000 字符，
      // 触发 Cloudflare _headers 限制，故关掉静态页的安全头写入。
      // 当前没有预渲染路由（/doc/** 已改回 SSR），这条是留给将来再开 prerender 时的开关：
      // 一旦重新启用 prerender，那些页就会退回「无 CSP」状态。
      nitroHeaders: false,
    },
    headers: {
      contentSecurityPolicy: {

        'script-src': [
          "'self'",
          "'strict-dynamic'",
          "'nonce-{{nonce}}'",
          "'wasm-unsafe-eval'",
          ...allowedDomains // 允许 Umami 发送数据
        ],
        
        'connect-src': [
          "'self'",
          ...allowedDomains,// 允许 Umami 发送数据
        ],
        
        'style-src': ["'self'", 'https:', "'unsafe-inline'"],
        'img-src': ["'self'", 'https://s0.wp.com', 'data:', 'https:'],
        'font-src': ["'self'", 'https:', 'data:'],
      }
    }
  },
  sitemap: {
    sources: [
      '/api/__sitemap__/urls',
    ]
  }


})
