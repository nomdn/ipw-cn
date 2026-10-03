import {config} from "./config/index";
import { docConfig } from "./config/doc";
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
    'build:manifest': (manifest) => {
      for (const chunk of Object.values(manifest) as Array<Record<string, unknown>>) {
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
        })),
        ...(middlewareOrigin
          ? [{ rel: 'dns-prefetch', href: middlewareOrigin }]
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
    // /doc 文档站改为 SSG：构建时预渲染为静态 HTML，由边缘静态资源直接响应
    '/doc/**': { prerender: true },
  },
  nitro: {
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
    prerender: {
      // 显式列出所有 doc 路由（来自 config/doc.ts 的 docConfig 键），
      // 让动态 [...slug] 页面在构建时全部预渲染为静态 HTML
      routes: ['/doc', ...Object.keys(docConfig).filter((p) => p !== '/doc')],
    },
  },
  security: {
    ssg: {
      // /doc 预渲染页的 CSP 会因 SRI modulepreload 哈希全量拼入而单行超 2000 字符，
      // 触发 Cloudflare _headers 限制；关闭静态页安全头写入 _headers。
      // SSR 页面仍由运行时中间件下发完整安全头，不受影响。
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
