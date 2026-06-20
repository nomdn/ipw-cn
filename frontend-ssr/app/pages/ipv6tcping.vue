<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';

const route = useRoute()

useHead({
  title: 'TCPing测试 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: 'TCPing测试 | 多节点 TCPing 测试，检测服务器连通性和延迟' },
    { name: 'keywords', content: 'tcping,tcp,tcp测试,延迟测试,连通性测试,服务器测试' },
    { property: 'og:title', content: 'TCPing测试 | 柠檬味ipw.cn' },
    { property: 'og:description', content: 'TCPing测试 | 多节点 TCPing 测试，检测服务器连通性和延迟' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
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

const tmpDomain = ref('')
const port = ref('80')
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
      <h1>TCPing 测试</h1>
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
              <td class="table-value">{{ server.ipv6?.ip || '-' }}</td>
              <td class="table-value">{{ server.ipv6?.sent }}</td>
              <td class="table-value">{{ server.ipv6?.success }}</td>
              <td class="table-value">
                <span :class="server.ipv4.loss_rate > 0 ? 'loss-warning' : 'loss-ok'">
                  {{ server.ipv4.loss_rate.toFixed(1) }}
                </span>
              </td>
              <td class="table-value">{{ server.ipv6?.max_rtt.toFixed(2) }}</td>
              <td class="table-value">{{ server.ipv6?.min_rtt.toFixed(2) }}</td>
              <td class="table-value">{{ server.ipv6?.avg_rtt.toFixed(2) }}</td>
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
