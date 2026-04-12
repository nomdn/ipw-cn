<script setup lang="ts">
import { ref,onMounted } from 'vue';
import { isIPv6 } from 'is-ip';
import axios from 'axios';
import { codeToHtml } from 'shiki'
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue';

const code = `
# 请勿用于商业用途，仅供个人测试学习之用，请遵守中国法律法规
# 查询本机外网 IPv4 地址
curl 4.wsmdn.dpdns.org

# 查询本机外网 IPv6 地址
curl 6.wsmdn.dpdns.org

# 测试网络是 IPv4 还是 IPv6 访问优先 
# (访问 IPv4/IPv6 双栈站点，如果返回 IPv6 地址，则 IPv6 访问优先)
curl test.wsmdn.dpdns.org
`
const html = ref('');


const ipAddress = ref('');
const yourIPv4 = ref('');
const yourIPv6 = ref('');
axios.get('https://test.wsmdn.dpdns.org').then(
  function(response) {
    ipAddress.value = response.data;
  }
).catch(
  function(error) {
    console.error('Error fetching IP address:', error);
  }
);
axios.get('https://4.wsmdn.dpdns.org').then(
  function(response) {
    yourIPv4.value = response.data;
  }
).catch(
  function(error) {
    console.error('Error fetching IP address:', error);
  }
);
axios.get('https://6.wsmdn.dpdns.org').then(
  function(response) {
    yourIPv6.value = response.data;
  }
).catch(
  function(error) {
    console.error('Error fetching IP address:', error);
  }
);
onMounted(async () => {
  html.value = await codeToHtml(code, {
    lang: 'bash',
    theme: 'github-dark'
  })
});



function isIPv4(ip: string): boolean {
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
  return ipRegex.test(ip);
}
</script>

<template>
  <el-menu
      mode="horizontal"
      :ellipsis="false"
    >
    <el-menu-item index="0">
      <img
        style="width: 50px"
        src="/hyw.webp"
        alt="IPW logo"
      />
      <h2 style="display: inline-block; margin-left: 10px">柠檬味ipw.cn</h2>
    </el-menu-item>
  </el-menu>
  <div class="title">
    <header>
      <h1>IP查询</h1>
      <p>致力于IP查询去中心化,推进 IPv6 规模部署和应用</p>
    </header>

  </div>
  <div class="content">
    <div class="one-line">
      <b>IPv4</b>&nbsp<p>{{ yourIPv4 }}</p>
    </div>
    <div class="one-line">
      <b>IPv6</b>&nbsp<p v-if="yourIPv6">{{ yourIPv6 }}</p><a v-else href="https://www.bing.com/search?q=%E5%AE%B6%E5%AE%BD%E5%BC%80ipv6" target="_blank">没有IPv6地址,查看如何开启IPv6</a>
    </div>
    <div style="font-size: 1.5em;">
      <h3 v-if="isIPv6(ipAddress)"><el-icon><CircleCheckFilled style="color: aquamarine;"/></el-icon>您的网络IPv6优先</h3>
      <h3 v-else-if="isIPv4(ipAddress)"><el-icon><CircleCloseFilled style="color: red;"/></el-icon>您的网络IPv4优先</h3>
      <h3 v-else><el-icon><CircleCloseFilled /></el-icon>请输入正确的ip地址</h3>
    </div>
     <blockquote>
      手机默认开启 IPv6，宽带开启 IPv6 请自行搜索 (我们没有文档)
    </blockquote>

    <div v-html="html"></div>
  </div>
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
}
:deep(.shiki){
  padding: 20px;
  border-radius: 10px;
}


</style>
<style>
:root {
  --el-color-primary: #3EAF7C;
}
</style>