<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';
import { isIPv6 } from 'is-ip';
const route = useRoute()

useHead({
  title: 'IPv6 TCPing测试工具 | IPv6服务器连通性检测 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的IPv6 TCPing测试工具,提供多节点IPv6 TCP连通性检测服务,支持自定义端口测试,实时检测IPv6服务器丢包率、平均延迟、最大最小响应时间,助力IPv6网络质量诊断与优化,推进IPv6规模部署和应用' },
    { name: 'keywords', content: 'ipv6 tcping测试,ipv6连通性检测,ipv6服务器延迟,ipv6丢包率,ipv6端口测试,ipv6网络质量,ipv6服务器测试,ipv6网络诊断' },
    { property: 'og:title', content: 'IPv6 TCPing测试 - IPv6服务器连通性与延迟检测' },
    { property: 'og:description', content: '多节点IPv6 TCPing测试,检测服务器连通性、丢包率与响应延迟' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv6 TCPing连通性测试工具',
        description: '专业的IPv6 TCPing测试工具，多节点检测IPv6服务器连通性和延迟，支持自定义端口测试，提供丢包率、平均延迟、最大最小响应时间等数据。',
        url: config.siteUrl + 'ipv6tcping',
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

interface TCPingStats {
  ip: string
  port: string
  sent: number
  success: number
  loss_rate: number
  max_rtt: number
  min_rtt: number
  avg_rtt: number
}

interface TCPingResponse {
  ipv4?: TCPingStats
  ipv6?: TCPingStats
}

interface ServerConfig {
  label: string
  url: string
}

interface ServerResult {
  label: string
  loading: boolean
  error: string
  ipv4?: TCPingStats
  ipv6?: TCPingStats
}

const tmpDomain = ref('www.zakoflare.com')
const port = ref('80')
const userIP = ref('')
const loading = ref(false)
const serverResults = ref<ServerResult[]>([])

const allServers = [
  ...config.TCPing.DualStack.map((s: ServerConfig) => ({ ...s, type: 'DualStack' })),
  ...config.TCPing.IPv6.map((s: ServerConfig) => ({ ...s, type: 'IPv6' }))
];

const tcpingFetches = allServers.map((server) => {
  const url = computed(() => server.url + 'v1/tcping/' + extractHost(tmpDomain.value) + '?port=' + port.value);
  const { data, error: fetchError, execute } = useFetch<TCPingResponse>(url, {
    immediate: false,
    watch: false,
  });
  return { label: server.label, data, error: fetchError, execute };
});

function initServerResults() {
  const results: ServerResult[] = []
  
  config.TCPing.DualStack.forEach((server: ServerConfig) => {
    results.push({
      label: server.label,
      loading: false,
      error: ''
    })
  })
  
  config.TCPing.IPv6.forEach((server: ServerConfig) => {
    results.push({
      label: server.label,
      loading: false,
      error: ''
    })
  })
  
  serverResults.value = results
}

function extractHost(url: string): string {
  const regex = /^(?:[a-zA-Z][a-zA-Z\d+.-]*:\/\/)?(?:[^\s@/]+@)?(?<host>(?:\[(?:[0-9a-fA-F:]+)\]|(?:\d{1,3}(?:\.\d{1,3}){3})|(?:[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?(?:\.[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?)*)))(?::\d{1,5})?(?:[/?#][^\s]*)?$/u;
  
  const match = url.trim().match(regex);
  return match?.groups?.host ?? url;
}

function TCPingAll() {
  const host = extractHost(tmpDomain.value)
  if (!host) return
  
  loading.value = true
  
  serverResults.value.forEach((result) => {
    result.loading = true
    result.error = ''
    result.ipv4 = undefined
    result.ipv6 = undefined
  })
  
  const promises = tcpingFetches.map(async (fetch, index) => {
    try {
      await fetch.execute();
      const result = serverResults.value[index];
      if (result) {
        result.ipv4 = fetch.data.value?.ipv4;
        result.ipv6 = fetch.data.value?.ipv6;
      }
    } catch (err) {
      console.error(err);
      const result = serverResults.value[index];
      if (result) {
        result.error = '请求失败';
      }
    } finally {
      const result = serverResults.value[index];
      if (result) {
        result.loading = false;
      }
    }
  })
  
  Promise.all(promises).finally(function () {
    loading.value = false
  })
}
async function getUserIP(){
  
  await $fetch<string>(config.DualStackAPI).then(
  function (data){
    userIP.value = data
  })
  return userIP.value
}
onMounted(() => {
  getUserIP()
  initServerResults()
  const urlParam = route.query.site as string
  if (urlParam) {
    tmpDomain.value = urlParam
    TCPingAll()
  }
})
</script>

<template>
  <div class="title">
    <header>
      <h1>IPv6 TCPing连通性测试工具</h1>
      <p>多节点 TCPing 测试，检测服务器连通性和延迟</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input 
        v-model="tmpDomain" 
        placeholder="请输入域名（如：example.com）" 
      />
      <el-input 
        v-model="port" 
        placeholder="端口号（默认 80）" 
        style="width: 200px;"
      />
      <el-button 
        @click="TCPingAll()" 
        type="primary" 
        :loading="loading"
      >
        开始测试
      </el-button>
    </div>

    <div class="result-section">
      <table class="result-table">
        <thead>
          <tr>
            <th class="table-header">服务器</th>
            <th class="table-header">解析 IP</th>
            <th class="table-header">发送包</th>
            <th class="table-header">接收包</th>
            <th class="table-header">丢包率(%)</th>
            <th class="table-header">最长时间(ms)</th>
            <th class="table-header">最短时间(ms)</th>
            <th class="table-header">平均时间(ms)</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="(server, index) in serverResults" :key="index">
            <tr v-if="server.ipv6">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value">{{ server.ipv6.ip || '-' }}</td>
              <td class="table-value">{{ server.ipv6.sent }}</td>
              <td class="table-value">{{ server.ipv6.success }}</td>
              <td class="table-value">
                <span :class="server.ipv6.loss_rate > 0 ? 'loss-warning' : 'loss-ok'">
                  {{ server.ipv6.loss_rate.toFixed(1) }}
                </span>
              </td>
              <td class="table-value">{{ server.ipv6.max_rtt.toFixed(2) }}</td>
              <td class="table-value">{{ server.ipv6.min_rtt.toFixed(2) }}</td>
              <td class="table-value">{{ server.ipv6.avg_rtt.toFixed(2) }}</td>
            </tr>
            <tr v-else-if="server.loading">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value" colspan="7">加载中...</td>
            </tr>
            <tr v-else-if="server.error">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value error-text" colspan="7">{{ server.error }}</td>
            </tr>
            <tr v-else>
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value" colspan="7">-</td>
            </tr>
          </template>
        </tbody>
      </table>

    </div>
      <blockquote>
        <a href="https://ipw-docs.wsmdn.top/user/ipv6_ping.html" target="_blank">IPv6 Ping 原理介绍</a><br/>
        <strong>注意本页是TCPing，不是ICMPv6 Ping，下列文本仅供参考</strong><br/>
        #1. 本地 IPv6 方式<br/>
        Windows: ping -6 ipw.wsmdn.top<br/>

        macOS 或 Linux: ping6 ipw.wsmdn.top<br/>

        #2. 服务器 IPv6 Ping 失败可能原因：<br/>
        服务器已开启 IPv6，但防火墙（又名安全组）未对源地址是 IPv6 地址(::/0)的 ICMPv6协议 开放访问，<br/>
        服务器未开启 IPv6，请参考 服务器开启 IPv6<br/>
        <a href="/tcping" target="_blank">IPv4 TCPing 测试</a> | <a href="/ipv6speedtest" target="_blank">IPv6 网站测速</a> | <a href="/ipv6webcheck">网站开启IPv6检测</a> | <a href="/dns">DNS解析查询</a> <br/>

        访客IP: {{ userIP }}，您的网络{{ isIPv6(userIP) ? 'IPv6' : 'IPv4'}}访问优先
      </blockquote>

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
  color: #cfcfcf;
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
  width: 200px;
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

.loss-ok {
  color: #67C23A;
  font-weight: 600;
}

.loss-warning {
  color: #F56C6C;
  font-weight: 600;
}

.error-text {
  color: #F56C6C;
}
</style>

<style>
:root {
  --el-color-primary: #3EAF7C;
}

html.dark {
  --el-color-primary: #3EAF7C;
}

.el-icon {
  font-size: 1.3em;
}
</style>
