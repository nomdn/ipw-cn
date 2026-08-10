<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { useDark, useToggle } from '@vueuse/core';
import { Moon, Sunny, Expand } from '@element-plus/icons-vue';
import { useRoute } from 'vue-router';
import { config } from '../config/index';

const isNarrow = ref(false);
let mediaQueryList: MediaQueryList | null = null;
const drawer = ref(false);
const route = useRoute();

const isDark = useDark();
const toggleDark = useToggle(isDark);

watch(() => route.fullPath, () => {
  drawer.value = false;
});

function cleanChineseCharacters(str:string) {
return str.replace(/[\u4e00-\u9fa5]/g, '');
}
let umamiScript: HTMLScriptElement | null = null

useHead({
  meta: config.noindex
    ? [
        { name: 'robots', content: 'noindex, nofollow' },
        { name: 'googlebot', content: 'noindex, nofollow' },
        { name: 'bingbot', content: 'noindex, nofollow' },
      ]
    : [],
});

onMounted(() => {
  mediaQueryList = window.matchMedia('(max-width: 768px)');
  isNarrow.value = mediaQueryList.matches;

  const handler = (e: MediaQueryListEvent) => {
    isNarrow.value = e.matches;
  };

  mediaQueryList.addEventListener('change', handler);

  onBeforeUnmount(() => {
    mediaQueryList?.removeEventListener('change', handler);
  });

  if (!umamiScript && config.umamiScriptUrl) {
    umamiScript = document.createElement('script')
    umamiScript.src = config.umamiScriptUrl
    umamiScript.async = true
    umamiScript.setAttribute('data-website-id', config.umamiWebsiteId)
    document.head.appendChild(umamiScript)
  }
})
</script>

<template>
  
  <el-drawer id="mobile-navigation" v-if="isNarrow" v-model="drawer" direction="ltr" style="height: 100%;" size="50%">
      <router-link to="/ipv6webcheck" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 网站检测</p>
      </router-link>
      <router-link to="/location" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6/IPv4 地址查询</p>
      </router-link>
      <router-link to="/ipv6tcping" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 TCPing</p>
      </router-link>
      <router-link to="/dns"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 DNS解析</p></router-link>
      <router-link to="/ssl" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 SSL检查</p>
      </router-link>
      <a href="/ipv6speedtest"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 网站测速</p></a>
      <a href="/speedtest"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv4 网站测速</p></a>
      <a href="/tcping"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv4 TCPing</p></a>
      <a href="/screenshot"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">网站截图</p></a>
      <a href="/whois"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">Whois查询</p></a>
      <a href="/asn"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">ASN查询</p></a>
      <a href="/dnssec"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">DNSSEC验证</p></a>
  </el-drawer>
  <el-menu
      mode="horizontal"
      :ellipsis="!isNarrow"
      
    >
    <el-menu-item index="0">
      <button
        v-if="isNarrow"
        type="button"
        class="nav-icon-button"
        aria-label="打开导航菜单"
        :aria-expanded="drawer"
        aria-controls="mobile-navigation"
        @click.stop="drawer = !drawer"
      >
        <el-icon aria-hidden="true"><Expand /></el-icon>
      </button>
      <router-link to="/">
        <el-image src="/favicon.svg" style="margin-top: 20px;" /> 
        <h2 style="display: inline-block; margin-left: 10px" v-if="!isNarrow">柠檬味ipw.cn</h2>
      </router-link>
    </el-menu-item>
    
    <el-menu-item index="1" v-if="!isNarrow">
      <router-link to="/ipv6webcheck" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 网站检测</p>
      </router-link>
    </el-menu-item>
    <el-menu-item index="2" v-if="!isNarrow">
      <router-link to="/location">
        <p style="display: inline-block; margin-left: 10px">IPv6/IPv4 地址查询</p>
      </router-link>
    </el-menu-item>
    <el-menu-item index="3" v-if="!isNarrow">
      <router-link to="/ipv6tcping"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 TCPing测试</p></router-link>
    </el-menu-item>

    <el-divider style="margin-top: 20px;height: 1.2em;" direction="vertical" v-if="!isNarrow"/>

    <el-menu-item index="4" v-if="!isNarrow">
      <router-link to="/dns"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 DNS解析</p></router-link>
    </el-menu-item>
    <el-menu-item index="5" v-if="!isNarrow">
      <router-link to="/ssl">
        <p style="display: inline-block; margin-left: 10px">IPv6 SSL检查</p>
      </router-link>
    </el-menu-item>
    <el-menu-item index="6" v-if="!isNarrow">
      <router-link to="/ipv6speedtest"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 网站测速</p></router-link>
    </el-menu-item>
    <el-divider style="margin-top: 20px;height: 1.2em;" direction="vertical" v-if="!isNarrow"/>
    <el-sub-menu index="7" v-if="!isNarrow">
      <template #title>IPv4工具箱</template>
      <el-menu-item index="7-0">
        <router-link to="/speedtest"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv4 网站测速</p></router-link>
      </el-menu-item>
      <el-menu-item index="7-1">
        <router-link to="/tcping"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv4 TCPing测试</p></router-link>
      </el-menu-item>
    </el-sub-menu>
    <el-sub-menu index="8" v-if="!isNarrow">
      <template #title>其他工具</template>
      <el-menu-item index="8-0">
        <router-link to="/screenshot"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">网站截图</p></router-link>
      </el-menu-item>
      <el-menu-item index="8-1">
        <router-link to="/whois"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">Whois查询</p></router-link>
      </el-menu-item>
      <el-menu-item index="8-3">
        <router-link to="/asn"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">ASN查询</p></router-link>
      </el-menu-item>
      <el-menu-item index="8-4">
        <router-link to="/dnssec"  style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">DNSSEC验证</p></router-link>
      </el-menu-item>
    </el-sub-menu>
    <el-menu-item index="9" v-if="!isNarrow">
      <router-link to="/doc" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">文档</p></router-link>
    </el-menu-item>
    <el-menu-item index="10">
      <ClientOnly>
      <button
        type="button"
        class="nav-icon-button"
        :aria-label="isDark ? '切换到浅色模式' : '切换到深色模式'"
        :aria-pressed="isDark"
        @click="toggleDark()"
      >
        <el-icon v-if="isDark" aria-hidden="true"><Moon style="height: 20px; width: 20px;"/></el-icon>
        <el-icon v-else aria-hidden="true"><Sunny style="height: 20px; width: 20px;"/></el-icon>
      </button>
      </ClientOnly>
    </el-menu-item>


  </el-menu>
  
  <NuxtLoadingIndicator />
  <main id="main-content" role="main">
    <NuxtPage />
  </main>

  <footer>
    <div class="one-line">
      Copyright © nomdn & IP 查询 2026  | <img src="/ipv6-s1.svg" alt="IPv6 相关标识"/> | <img src="/ssl-s1.svg" alt="SSL 相关标识"/> | All right reserved
    </div>
    <div class="one-line">
      <a v-if="config.ICP" href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer" >{{ config.ICP }}</a>
      <span v-if="config.ICP">&nbsp;|&nbsp;</span>
      <el-image v-if="config.GongAn" style="height: 1em; width: 1em;" src="/备案图标.png" />
      <a :href="'https://beian.mps.gov.cn/#/query/webSearch?code=' + cleanChineseCharacters(config.GongAn)" target="_blank" rel="noreferrer" >{{ config.GongAn }}</a>
      <span v-if="config.GongAn">&nbsp;|&nbsp;</span>
      <a href="https://www.china-ipv6.cn/" target="_blank" rel="noreferrer" >国家IPv6发展监测平台</a>
      &nbsp;|&nbsp;请遵守中国法律法规&nbsp;|&nbsp;
      <a href="https://github.com/nomdn/ipw-cn" target="_blank" rel="noreferrer" >Github</a>&nbsp;|&nbsp;
      <a href="https://qm.qq.com/q/E1CGjkqgG6" target="_blank" rel="noreferrer" >QQ用户交流群</a>
   </div>
   <div class="one-line">
      致力于普及IPv6，推进IPv6规模部署和应用，以全面推进IPv6技术创新与融合应用为主线，以提升应用广度深度为主攻方向
  </div>
  </footer>

</template>
<style scoped>
@import "~/style.css";
.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}
.nav-icon-button {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  padding: 0;
  color: inherit;
  background: transparent;
  border: 0;
  border-radius: 8px;
  cursor: pointer;
}
.nav-icon-button:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
@media (hover: hover) and (pointer: fine) {
  .nav-icon-button:hover {
    background: var(--el-fill-color-light);
  }
}
:deep(.shiki span) {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace !important;
  word-wrap:break-word;
}
:deep(.shiki){
  padding: 20px;
  border-radius: 10px;
  overflow: auto;
  max-width: 100%;
}

:deep(.el-menu-item a) {
  font-size: 1em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:deep(.el-menu-item a p) {
  font-size: 1em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:deep(.el-menu-item a img) {
  width: 50px;
  margin-bottom: 20px;
  
}
</style>
<style>
:root {
  --el-color-primary: #3EAF7C;
}
html.dark {
  --el-color-primary: #3EAF7C;
}

/* Drawer 内部链接占满一行 */
.el-drawer__body {
  overflow-x: hidden;
}
.el-drawer__body a {
  display: block !important;
  width: 100% !important;
  box-sizing: border-box;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 1em;
}
.el-drawer__body a p {
  display: block !important;
  width: 100% !important;
  box-sizing: border-box;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 在 Vue 水合前也按视口隐藏桌面导航，避免窄屏横向溢出。 */
@media (max-width: 768px) {
  .el-menu--horizontal > .el-divider,
  .el-menu--horizontal > .el-sub-menu,
  .el-menu--horizontal > .el-menu-item:not(:first-child):not(:last-child) {
    display: none !important;
  }

  .el-menu--horizontal > .el-menu-item:first-child h2 {
    display: none !important;
  }
}
.el-menu--horizontal {
  --el-menu-hover-bg-color: transparent !important;
  --el-menu-active-color: var(--el-text-color-primary) !important;
  --el-menu-bg-color: transparent !important;
}

.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

/* 去除选中强调和下划线 */
.el-menu--horizontal > .el-menu-item.is-active {
  color: var(--el-text-color-primary) !important;
  background-color: transparent !important;
}

.el-menu--horizontal > .el-menu-item:hover {
  color: var(--el-text-color-primary) !important;
  background-color: transparent !important;
}

/* 去除所有可能的边框和下划线 */
.el-menu--horizontal::after {
  display: none !important;
}

.el-menu--horizontal > .el-menu-item {
  border-bottom: none !important;
  transition: none !important;
}

.el-menu--horizontal > .el-menu-item.is-active::after {
  display: none !important;
}
/* 覆盖选中、悬停和聚焦状态的高亮样式 */


</style>
