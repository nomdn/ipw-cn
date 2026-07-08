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
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: ["nitro-cloudflare-dev",'@element-plus/nuxt', '@nuxtjs/sitemap', '@nuxtjs/robots', '@vueuse/nuxt', '@nuxt/content',"nuxt-security"],
  vite: {
    optimizeDeps: {
      include: [
        'dayjs/plugin/*.js',
        'dayjs',
        'lodash-unified',
        'shiki',
      ]
    }
  },
  site: { 
  url: config.siteUrl, 
  name: 'Lemon IPW' 
  },
  css: [
    // 1. 引入 Element Plus 基础样式 (如果你还没有引入的话)
    'element-plus/dist/index.css',
    
    // 2. 🌟 关键：引入 Element Plus 官方的暗黑模式 CSS 变量文件
    'element-plus/theme-chalk/dark/css-vars.css'
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
    public: {
      siteUrl: config.siteUrl,
    },
  },
nitro: {
    preset: "cloudflare_module",
    cloudflare: {
      deployConfig: true,


      nodeCompat: true
    },
  },
  security: {
    headers: {
      contentSecurityPolicy: {

        'script-src': [
          "'self'",
          "'strict-dynamic'",
          "'nonce-{{nonce}}'",
          ...allowedDomains // 允许 Umami 发送数据
        ],
        
        'connect-src': [
          "'self'",
          ...allowedDomains,// 允许 Umami 发送数据
        ],
        
        'style-src': ["'self'", 'https:', "'unsafe-inline'"],
        'font-src': ["'self'", 'https:', 'data:'],
      }
    }
  }
  
})