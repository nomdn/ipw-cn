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
      ]
    },
  },
  site: { 
  url: config.siteUrl, 
  name: 'Lemon IPW' 
  },
  css: [
    // 1. 引入 Element Plus 基础样式 (如果你还没有引入的话)
    'element-plus/dist/index.css',
    
    // 2. 🌟 关键：引入 Element Plus 官方的暗黑模式 CSS 变量文件
    'element-plus/theme-chalk/dark/css-vars.css',
  ],
  app:{
    head: {
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
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.svg' }
      ]
    }
  },
  runtimeConfig: {
    indexnowKey: '',
    apiKeys: '',
    // 数据上报（协议见收集中心中间件 report.go）。敏感项 token 只从环境变量注入，不进仓库。
    // NUXT_BOCE_REPORT_URL：收集中心基址（如 https://collector.example.com）；空 = 不上报
    boceReportUrl: '',
    // NUXT_BOCE_REPORT_TOKEN：收集中心 /report 鉴权 token
    boceReportToken: '',
    // NUXT_BOCE_REPORT_INSTANCE：上报方标识（存收集库 origin）；空 = 默认 frontend-ssr
    boceReportInstance: '',
    public: {
      siteUrl: config.siteUrl,
      docConfig: docConfig,

    },
  },
  routeRules: {
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
