<script setup lang="ts">
import { ref ,onMounted} from 'vue';
import { isIPv6 } from 'is-ip';
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue';
import axios from 'axios';
import { useRoute } from 'vue-router'
import { codeToHtml } from 'shiki'
const route = useRoute();
const loading = ref(false);
interface IPDetailType {
  region?: any;
  [key: string]: any;
}
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
const remoteAPI = ref('https://api-ipw.wsmdn.dpdns.org/');
const ipAddress = ref('');
const IPDetail = ref<IPDetailType>({});
const IPLocation = ref<any[]>([]);
const UserIP = ref('');
axios.get('https://test.wsmdn.dpdns.org').then(
  function(response) {
    ipAddress.value = response.data;
    UserIP.value = response.data;
  }
).catch(
  function(error) {
    console.error('Error fetching IP address:', error);
  }
);

function locateIP(IP: string){
  loading.value = true;
  axios.get(remoteAPI.value+"v1/location/"+IP).then(
    function(response) {
      IPDetail.value = response.data;
      // 将IPLocation赋值移到这里，确保IPDetail已经更新
      if (IPDetail.value && IPDetail.value.hasOwnProperty('region')) {
        IPLocation.value = IPDetail.value.region.split("|");
        console.log(IPLocation.value);
      } else {
        IPLocation.value = [];
      }
      loading.value = false;
    }
  ).catch(
    function(error) {
      console.error('Error fetching IP address:', error);
    }
  );
}

function isIPv4(ip: string): boolean {
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
  return ipRegex.test(ip);
}
onMounted(async () => {
  html.value = await codeToHtml(code, {
    lang: 'bash',
    theme: 'github-dark'
  })
  const urlParam = route.query.ip;
  if (urlParam) {
    ipAddress.value = urlParam as string;
    locateIP(urlParam as string);
  }
});
</script>
<template>
  <div class="title">
    <header>
      <h1>IPv6地址查询</h1>
      <p>极简的IPv6地址查询工具，致力于普及 IPv6</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input 
        v-model="ipAddress" 
        placeholder="请输入IP地址" 
      />
      <el-button 
        @click="locateIP(ipAddress)" 
        type="primary" 
        :loading="loading"
      
      >
        查询
      </el-button>
    </div>
    <div class="location">
      <div class="one-line" style="height: 40px;">
        <b>IP</b>&nbsp<p>{{ ipAddress }}</p>
      </div>
      <div class="one-line" style="height: 40px;">
        <b>地区</b>&nbsp<p>{{ IPLocation[4] }}&nbsp{{ IPLocation[0] }}&nbsp{{ IPLocation[1] }}&nbsp{{ IPLocation[2] }}</p>
      </div>
      <div class="one-line" style="height: 40px;">
        <b>运营商</b>&nbsp<p>{{ IPLocation[3] }}</p>
      </div>
    </div>
    <div style="font-size: 1.5em;">
      <h3 v-if="isIPv6(UserIP)"><el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>您的网络IPv6优先</h3>
      <h3 v-else-if="isIPv4(UserIP)"><el-icon><CircleCloseFilled style="color: red;"/></el-icon>您的网络IPv4优先</h3>
    </div>
    <blockquote>
      该IPv6归属地的精度为市级，是Ip2Region的社区数据库，精度是棍母，随便用，不刷炸服务器就行。<br>
      手机默认开启 IPv6，宽带开启 IPv6 请自行搜索<br>
      访客IP: {{UserIP}}，<p v-if="isIPv4(UserIP)">您的网络IPv4优先</p><p v-else-if="isIPv6(UserIP)">您的网络IPv6优先</p>
    </blockquote>

    <div v-html="html"></div>
    </div>



</template>
<style scoped>
.el-input {
  width: 420px;
  height: 50px;
  font: 1.3em sans-serif;
  margin-right: 10px;
}

.el-button {
  width: 165px;
  height: 50px;
  font: 1.3em sans-serif;
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
@import "../style.css";
:root {
  --el-color-primary: #3EAF7C;
}
.el-icon{
  font-size: 1.3em;
}

</style>