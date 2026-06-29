<script setup lang="ts">
import { ref, onMounted, computed, nextTick, watch } from 'vue';
import { isIPv6 } from 'is-ip';
import {config} from '../../config/index';
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue';
import { useRoute } from 'vue-router'
import { codeToHtml } from 'shiki'
const route = useRoute();
const loading = ref(false);

useHead({
  title: 'IPv6地址查询工具 | IP归属地定位 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的IPv6地址查询工具,支持IPv4和IPv6地址归属地查询,提供BiliBili Live、GeoCN、IP2Region、Maxmind等多种数据源对比,精确定位IP地理位置、运营商信息,支持中国大陆及境外地址查询,助力IPv6普及与应用' },
    { name: 'keywords', content: 'ipv6地址查询,ipv4地址查询,ip归属地,ip地理位置,ip定位,运营商查询,ipv6归属地,ipv4归属地,maxmind,ip2region,geocn' },
    { property: 'og:title', content: 'IPv6/IPv4地址归属地查询工具 - 柠檬味ipw.cn' },
    { property: 'og:description', content: '多数据源IP地址归属地查询,支持IPv4和IPv6,提供地理位置、运营商等详细信息' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv6/IPv4地址归属地查询工具',
        description: '专业的IPv6/IPv4地址归属地查询工具，支持BiliBili Live、GeoCN、IP2Region、Maxmind、纯真社区库等多数据源对比，精确定位IP地理位置、运营商信息。',
        url: config.siteUrl + 'ipv6',
        applicationCategory: 'DeveloperApplication',
        operatingSystem: 'Web',
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'CNY'
        },
        provider: {
          '@type': 'Organization',
          name: '柠檬味ipw.cn'
        }
      })
    }
  ]
});
interface IPLocationType {
  bilibili?: any;
  ip?: string;
  geocn?: any;
  ip2region?: any;
  maxmind_asn?: any;
  maxmind_city?: any;
  qqwry?: any;
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
const remoteAPI = ref(config.IPLocationAPI);
const ipAddress = ref('');
const IPLocation = ref<IPLocationType>({});
const UserIP = ref('');
const locationUrl = computed(() => remoteAPI.value + "v1/location/" + ipAddress.value);

const { data: locationData, error: locationError, execute: executeLocation } = useFetch<IPLocationType>(locationUrl, {
  immediate: false,
  watch: false,
});

watch(locationData, (newData) => {
  if (newData) {
    IPLocation.value = newData;
    loading.value = false;
  }
});

watch(locationError, (newError) => {
  if (newError) {
    console.error('Error fetching IP location:', newError);
    loading.value = false;
  }
});

function locateIP(IP: string){
  ipAddress.value = IP;
  loading.value = true;
  nextTick(() => {
    executeLocation();
  });
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
  }else{
    $fetch<string>(config.DualStackAPI).then(
      function(data) {
        ipAddress.value = data;

        UserIP.value = data;
      }
    ).catch(
      function(error) {
        console.error('Error fetching IP address:', error);
      }
    );

  }
});
</script>
<template>
  <div class="title">
    <header>
      <h1>IPv6/IPv4地址归属地查询</h1>
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
      <div v-if="IPLocation" class="result-section">
          <table class="result-table">
            <tbody>
              <tr v-if="IPLocation.bilibili && (IPLocation.bilibili.administrative_area || IPLocation.bilibili.city)">
                <td class="table-label">bilibili Live接口</td>
                <td class="table-value">
                  <span>
                    {{ IPLocation.bilibili?.country }}&nbsp;{{ IPLocation.bilibili?.administrative_area }}&nbsp;{{ IPLocation.bilibili?.city }}
                  </span>
                </td>
                <td class="table-value">
                  <span>
                    {{ IPLocation.bilibili?.isp }}
                  </span>
                </td>
              </tr>
              <tr v-if="IPLocation.ip2region && IPLocation.ip2region.split('|').length >= 4">
                <td class="table-label">IP2Region</td>
                <td class="table-value">{{ IPLocation.ip2region?.split("|")[0] }}&nbsp;{{ IPLocation.ip2region?.split("|")[1] }}&nbsp;{{ IPLocation.ip2region?.split("|")[2] }}</td>
                <td class="table-value">{{ IPLocation.ip2region?.split("|")[3] }}</td>
              </tr>
              <tr v-if="IPLocation.geocn && (IPLocation.geocn.administrative_area || IPLocation.geocn.city || IPLocation.geocn.district)">
                <td class="table-label">GeoCN(仅中国大陆)</td>
                <td class="table-value">{{ IPLocation.geocn?.administrative_area }}&nbsp;{{ IPLocation.geocn?.city }}&nbsp;{{ IPLocation.geocn?.district }}</td>
                <td class="table-value">{{ IPLocation.geocn?.isp }}</td>
              </tr>
              <tr v-if="IPLocation.maxmind_city && IPLocation.maxmind_asn && (IPLocation.maxmind_city.country || IPLocation.maxmind_city.city)">
                <td class="table-label">Maxmind GEOLite2 City</td>
                <td class="table-value">{{ IPLocation.maxmind_city?.country }}&nbsp;{{ IPLocation.maxmind_city?.administrative_area }}&nbsp;{{ IPLocation.maxmind_city?.city }}</td>
                <td class="table-value">{{ IPLocation.maxmind_asn?.org }}</td>
              </tr>
              <tr v-if="IPLocation.qqwry && (IPLocation.qqwry.country || IPLocation.qqwry.administrative_area || IPLocation.qqwry.city)">
                <td class="table-label">纯真社区库</td>
                <td class="table-value">{{ IPLocation.qqwry?.country }}&nbsp;{{ IPLocation.qqwry?.administrative_area }}&nbsp;{{ IPLocation.qqwry?.city }}</td>
                <td class="table-value">{{ IPLocation.qqwry?.isp }}</td>
              </tr>
              <tr v-if="IPLocation.dbip_city && (IPLocation.dbip_city.administrative_area || IPLocation.dbip_city.city)">
                <td class="table-label">DB-IP City</td>
                <td class="table-value">{{ IPLocation.dbip_city?.country }}&nbsp;{{ IPLocation.dbip_city?.administrative_area }}&nbsp;{{ IPLocation.dbip_city?.city }}</td>
                <td class="table-value">--</td>
              </tr>
            </tbody>
          </table>
        </div>
    </div>
    <div style="font-size: 1.5em;">
      <h3 v-if="isIPv6(UserIP)"><el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>您的网络IPv6优先</h3>
      <h3 v-else-if="isIPv4(UserIP)"><el-icon><CircleCloseFilled style="color: red;"/></el-icon>您的网络IPv4优先</h3>
    </div>
    <blockquote>
      精度参照表：<br>
      中国大陆:BiliBili Live > GeoCN > IP2Region > 纯真社区库 > Maxmind GEOLite2 City ≈ DB-IP<br>
      中国大陆 IPv6 地址: BiliBili Live > GeoCN > IP2Region > Maxmind GEOLite2 City ≈ DB-IP > 纯真社区库<br>
      境外及港澳台地址: Maxmind GEOLite2 City ≈ DB-IP > BiliBili Live >  > IP2Region > 纯真社区库<br>
      
      手机默认开启 IPv6，宽带开启 IPv6 请自行搜索<br>
      访客IP: {{UserIP}}，<p v-if="isIPv4(UserIP)">您的网络IPv4优先</p><p v-else-if="isIPv6(UserIP)">您的网络IPv6优先</p>
    </blockquote>

    <div v-html="html"></div>

    </div>



</template>
<style scoped>
@import "../style.css";
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
.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

:deep(.shiki span) {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace !important;
}

:deep(.shiki) {
  padding: 20px;
  border-radius: 10px;
}

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

.result-section {
  margin-top: 30px;
}

.result-table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
  border: 1px solid #dcdfe6;
}
html.dark .result-table {
  width: 100%;
  border-collapse: collapse;
  background: #2e2d2d;
  border: 1px solid #2e2e2e;
}
.result-table thead tr {
  background-color: #c0c4cc;
}
html.dark .result-table thead tr {
    background: #2e2d2d;
}

.result-table .table-header {
  padding: 12px 15px;
  font-weight: 600;
  color: #303133;
  text-align: left;
  border: 1px solid #dcdfe6;
}
html.dark .result-table .table-header {
  padding: 12px 15px;
  font-weight: 600;
  color: #cfcfcf;
  text-align: left;
  border: 1px solid #1a1919;
}

.result-table tbody tr {
  border-bottom: 1px solid #dcdfe6;
}
html.dark .result-table tbody tr {
  border-bottom: 1px solid #1a1919;
}


.result-table tbody tr:last-child {
  border-bottom: none;
}

.result-table tbody tr:hover {
  background-color: #f5f7fa;
}
html.dark .result-table tbody tr:hover {
  background-color: #393a3a;
}

.result-table .table-label {
  padding: 12px 15px;
  font-weight: 600;
  color: #606266;
  width: 150px;
  text-align: left;
  border: 1px solid #dcdfe6;
}
html.dark .result-table .table-label {
  color: #c0c4cc;
  border: 1px solid #1a1919;
}

.result-table .table-value {
  padding: 12px 15px;
  color: #303133;
  border: 1px solid #dcdfe6;
}
html.dark .result-table .table-value {
  color: #cfcfcf;
  border: 1px solid #1a1919;
}
.valid {
  color: #67C23A;
  font-weight: 600;
}

.expired {
  color: #F56C6C;
  font-weight: 600;
}

.status-code {
  font-weight: 600;
  padding: 4px 12px;
  border-radius: 4px;
}

.status-success {
  color: #67C23A;
  background: #f0f9eb;
}

.status-warning {
  color: #E6A23C;
  background: #fdf6ec;
}

.status-error {
  color: #F56C6C;
  background: #fef0f0;
}

.error-message {
  margin-top: 20px;
  padding: 15px;
  background: #fef0f0;
  color: #F56C6C;
  border-radius: 6px;
  text-align: center;
  font-size: 1.1em;
}

pre {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 6px;
  overflow-x: auto;
  white-space: pre;
  max-width: 100%;
}

pre code {
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 0.9em;
  color: #303133;
}
html.dark pre {
  background: #303133;
}

html.dark pre code {
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  color: #f8f9fa;
}



.badge-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
  margin-top: 20px;
}

.badge-item {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.badge-item h4 {
  margin: 0 0 15px 0;
  color: #3EAF7C;
  font-size: 1.2em;
}

.badge-item img {
  display: block;
  margin-bottom: 15px;
  max-width: 200px;
}
</style>