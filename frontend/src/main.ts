import { createApp } from 'vue'
import App from './App.vue'

import router from "./index.ts"
import 'element-plus/theme-chalk/dark/css-vars.css'
import './style.css'
createApp(App).use(router).mount('#app')
