<script setup lang="ts">



</script>

<template>
  <el-menu
      mode="horizontal"
      :ellipsis="false"
      router
    >
    <el-menu-item index="/">
      <img
        style="width: 50px"
        src="/hyw.webp"
        alt="IPW logo"
      />
      <h2 style="display: inline-block; margin-left: 10px">柠檬味ipw.cn</h2>
    </el-menu-item>
    <el-menu-item index="/ssl">
      <p style="display: inline-block; margin-left: 10px">IPv6 SSL检查</p>
    </el-menu-item>
        <el-menu-item index="/ipv6">
      <p style="display: inline-block; margin-left: 10px">IPv6 地址查询</p>
    </el-menu-item>

    <el-menu-item index="/ipv6webcheck">
      <p style="display: inline-block; margin-left: 10px">IPv6 网站检测</p>
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