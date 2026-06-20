import {config} from "./config/index";
// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  modules: ['@element-plus/nuxt', '@nuxtjs/sitemap', '@nuxtjs/robots', '@vueuse/nuxt'],
  vite: {
    optimizeDeps: {
      include: [
        'dayjs/plugin/*.js',
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
      link: [
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.svg' }
      ]
    }
  }
})
