<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';
import { extractHost } from '~/utils/tools';

const route = useRoute()

useHead({
  title: 'TCPing测试工具 | IPv4服务器连通性检测 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的IPv4 TCPing测试工具,提供多节点TCP连通性检测服务,支持自定义端口测试,实时检测服务器丢包率、平均延迟、最大最小响应时间,帮助运维人员快速诊断服务器网络质量,确保服务稳定运行' },
    { name: 'keywords', content: 'tcping测试,tcp连通性检测,ipv4 tcping,服务器延迟测试,丢包率检测,端口连通性,服务器网络测试,网络质量检测' },
    { property: 'og:title', content: 'TCPing测试工具 - IPv4服务器连通性与延迟检测' },
    { property: 'og:description', content: '多节点TCPing测试,检测服务器连通性、丢包率与响应延迟' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv4 TCPing连通性测试工具',
        description: '专业的IPv4 TCPing测试工具，多节点检测服务器连通性和延迟，支持自定义端口测试，提供丢包率、平均延迟、最大最小响应时间等数据。',
        url: config.siteUrl + 'tcping',
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
const loading = ref(false)
const serverResults = ref<ServerResult[]>([])

const allServers = [
  ...config.TCPing.DualStack.map((s: ServerConfig) => ({ ...s, type: 'DualStack' })),
  ...config.TCPing.IPv4.map((s: ServerConfig) => ({ ...s, type: 'IPv4' }))
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
  
  config.TCPing.IPv4.forEach((server: ServerConfig) => {
    results.push({
      label: server.label,
      loading: false,
      error: ''
    })
  })
  
  serverResults.value = results
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

onMounted(() => {
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
      <h1>IPv4 TCPing 测试</h1>
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
            <tr v-if="server.ipv4">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value">{{ server.ipv4?.ip || '-' }}</td>
              <td class="table-value">{{ server.ipv4?.sent }}</td>
              <td class="table-value">{{ server.ipv4?.success }}</td>
              <td class="table-value">
                <span :class="server.ipv4.loss_rate > 0 ? 'loss-warning' : 'loss-ok'">
                  {{ server.ipv4.loss_rate.toFixed(1) }}
                </span>
              </td>
              <td class="table-value">{{ server.ipv4?.max_rtt.toFixed(2) }}</td>
              <td class="table-value">{{ server.ipv4?.min_rtt.toFixed(2) }}</td>
              <td class="table-value">{{ server.ipv4?.avg_rtt.toFixed(2) }}</td>
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

  </div>
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";

.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

.result-table .table-label {
  width: 200px;
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
