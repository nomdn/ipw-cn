<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref } from 'vue';
import { useDark, useToggle } from '@vueuse/core'
import { Moon, Sunny,Expand } from '@element-plus/icons-vue'
const isNarrow = ref(false);
let mediaQueryList:any = null;
const drawer = ref(false);

const isDark = useDark()
const toggleDark = useToggle(isDark)


onMounted(() => {
  mediaQueryList = window.matchMedia('(max-width: 768px)');
  isNarrow.value = mediaQueryList.matches;
  
  // 监听窗口大小变化（可选，增强体验）
  const handler = (e:any) => {
    isNarrow.value = e.matches;
  };
  mediaQueryList.addEventListener('change', handler);

  // 清理监听器
  onBeforeUnmount(() => {
    mediaQueryList.removeEventListener('change', handler);
  });

  console.log("当前设备是否是窄屏：" + isNarrow.value);
});
</script>

<template>
  <el-drawer v-model="drawer" direction="ltr" style="height: 100%;" size="50%">
      <router-link to="/ipv6webcheck" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 网站检测</p>
      </router-link>
      <router-link to="/ipv6" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 地址查询</p>
      </router-link>
      <a href="https://www.itdog.cn/ping_ipv6/" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 Ping测试</p></a>
      <a href="https://www.itdog.cn/dns/" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 DNS解析</p></a>
      <router-link to="/ssl" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 SSL检查</p>
      </router-link>
      <a href="https://www.itdog.cn/http_ipv6/" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 网站测速</p></a>
  </el-drawer>
  <el-menu
      mode="horizontal"
      :ellipsis="false"
      
    >
    <el-menu-item index="0">
      <el-icon v-if="isNarrow" @click="drawer = !drawer"><Expand /></el-icon>
      <router-link to="/">
        <img
          src="/hyw.webp"
          alt="IPW logo"
        />
        <h2 style="display: inline-block; margin-left: 10px">柠檬味ipw.cn</h2>
      </router-link>
    </el-menu-item>
    
    <el-menu-item index="1" v-if="!isNarrow">
      <router-link to="/ipv6webcheck" style="font-size: 1em;">
        <p style="display: inline-block; margin-left: 10px">IPv6 网站检测</p>
      </router-link>
    </el-menu-item>
    <el-menu-item index="2" v-if="!isNarrow">
      <router-link to="/ipv6">
        <p style="display: inline-block; margin-left: 10px">IPv6 地址查询</p>
      </router-link>
    </el-menu-item>
    <el-menu-item index="3" v-if="!isNarrow">
      <a href="https://www.itdog.cn/ping_ipv6/" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 Ping测试</p></a>
    </el-menu-item>

    <el-divider style="margin-top: 20px;height: 1.2em;" direction="vertical" v-if="!isNarrow"/>

    <el-menu-item index="4" v-if="!isNarrow">
      <a href="https://www.itdog.cn/dns/" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 DNS解析</p></a>
    </el-menu-item>
    <el-menu-item index="5" v-if="!isNarrow">
      <router-link to="/ssl">
        <p style="display: inline-block; margin-left: 10px">IPv6 SSL检查</p>
      </router-link>
    </el-menu-item>
    <el-menu-item index="6" v-if="!isNarrow">
      <a href="https://www.itdog.cn/http_ipv6/" style="font-size: 1em;"><p style="display: inline-block; margin-left: 10px">IPv6 网站测速</p></a>
    </el-menu-item>
    <el-divider style="margin-top: 20px;height: 1.2em;" direction="vertical" v-if="!isNarrow"/>
    <el-sub-menu index="7" v-if="!isNarrow">
      <template #title>IPv4工具箱</template>
      <el-menu-item index="7-0">没有</el-menu-item>
    </el-sub-menu>
    <el-menu-item index="8">
      <el-icon @click="toggleDark()" v-if="isDark" style="cursor: pointer;"><Moon style="height: 20px; width: 20px;"/></el-icon>
      <el-icon @click="toggleDark()" v-else style="cursor: pointer;"><Sunny style="height: 20px; width: 20px;"/></el-icon>
    </el-menu-item>


  </el-menu>
  
  <router-view />

  <footer>
    <div class="one-line">
      Copyright © nomdn & IP 查询 2026  | <img src="/ipv6-s1.svg"/> | <img src="/ssl-s1.svg" /> | All right reserved
    </div>
    <div class="one-line">
      <a href="https://www.china-ipv6.cn/">国家IPv6发展监测平台</a> | 请遵守中国法律法规
   </div>
   <div class="one-line">
      致力于普及IPv6，推进IPv6规模部署和应用，以全面推进IPv6技术创新与融合应用为主线，以提升应用广度深度为主攻方向
  </div>
  </footer>

</template>
<style scoped>
.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}
:deep(.shiki span) {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace !important;
  word-wrap:break-word;
}
:deep(.shiki){
  padding: 20px;
  border-radius: 10px;
}
:deep(.el-menu-item a) {
  font-size: 1em;
}
:deep(.el-menu-item a p) {
  font-size: 1em;
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

</style>