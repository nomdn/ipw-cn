<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';


const route = useRoute()

useHead({
  title: 'IPv6网站测速工具 | 全国多节点并发测速 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的IPv6网站测速工具,提供全国多节点并发测速服务,1秒内快速返回网站响应时间、下载速度、页面大小、DNS解析时间、HTTP连接时间等详细信息,支持IPv6网站性能检测与优化,助力IPv6网站性能监控与用户体验提升' },
    { name: 'keywords', content: 'ipv6网站测速,ipv6测速,网站速度测试,ipv6性能检测,网站响应时间,ipv6下载速度,ipv6性能优化,ipv6网站监控' },
    { property: 'og:title', content: 'IPv6网站测速 - 全国多节点并发性能检测工具' },
    { property: 'og:description', content: '全国多节点IPv6网站测速,快速获取响应时间、下载速度等性能数据' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv6网站性能测速工具',
        description: '专业的IPv6网站测速工具，全国多节点并发测速，提供响应时间、下载速度、页面大小、DNS解析时间等详细性能指标。',
        url: config.siteUrl + 'ipv6speedtest',
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

const tmpDomain = ref('https://www.zakoflare.com')
const loading = ref(false)
const result = ref<any[]>([]) 
const error = ref('')
const domain = ref('')

function getStatusCodeClass(code: number): string {
  if (code >= 200 && code < 300) return 'status-success'
  if (code >= 300 && code < 400) return 'status-warning'
  return 'status-error'
}
function formatTime(ms: number): string {
  if (ms == null) return '-'
  if (ms < 1000) {
    return `${ms} ms`
  }
  return `${(ms / 1000).toFixed(2)} s`
}

function formatSpeed(speed: number): string {
  if (speed == null) return '-'
  return `${speed.toFixed(2)} KB/s`
}

function formatSize(bytes: number): string {
  if (bytes == null) return '-'
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`
  }
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

function extractHost(url: string): string {
  const regex = /^(?:[a-zA-Z][a-zA-Z\d+.-]*:\/\/)?(?:[^\s@/]+@)?(?<host>(?:\[(?:[0-9a-fA-F:]+)\]|(?:\d{1,3}(?:\.\d{1,3}){3})|(?:[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?(?:\.[\p{L}\p{N}][\p{L}\p{N}\p{M}\u200c\u200d._-]*?)*)))(?::\d{1,5})?(?:[/?#][^\s]*)?$/u;
  
  const match = url.trim().match(regex);
  return match?.groups?.host ?? url;
}

const dualStackFetches = config.SpeedTest.DualStack.map((server) => {
  const url = computed(() => server.url + 'v1/speed/v6/' + domain.value);
  const { data, error: fetchError, execute } = useFetch(url, {
    immediate: false,
    watch: false,
  });
  return { label: server.label, data, error: fetchError, execute };
});

const ipv6Fetches = config.SpeedTest.IPv6.map((server) => {
  const url = computed(() => server.url + 'v1/speed/v6/' + domain.value);
  const { data, error: fetchError, execute } = useFetch(url, {
    immediate: false,
    watch: false,
  });
  return { label: server.label, data, error: fetchError, execute };
});

async function SpeedTest() {
  domain.value = extractHost(tmpDomain.value)
  loading.value = true
  result.value = []
  error.value = ''
  await nextTick()

  const allFetches = [
    ...dualStackFetches.map(fetch => ({ ...fetch, type: 'DualStack' })),
    ...ipv6Fetches.map(fetch => ({ ...fetch, type: 'IPv6' }))
  ];

  const promises = allFetches.map(async (fetch) => {
    try {
      await fetch.execute();
      return {
        server: fetch.label,
        data: fetch.data.value
      };
    } catch (err: any) {
      return {
        server: fetch.label,
        data: {},
        error: err.data?.error || err.message || '请求失败'
      };
    }
  });

  const promiseResults = await Promise.all(promises)
  console.log(promiseResults)
  result.value = promiseResults
  loading.value = false
  return promiseResults
}

onMounted(() => {
  const urlParam = route.query.site as string
  if (urlParam) {
    tmpDomain.value = urlParam
    SpeedTest()
  }
})
</script>

<template>
  <div class="title">
    <header>
      <h1>IPv6 网站测速</h1>
      <p>全国并发测速，1s 内快速返回测速结果</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input 
        v-model="tmpDomain" 
        placeholder="请输入域名（如：https://zakoflare.com）" 
      />
      <el-button 
        @click="SpeedTest()" 
        type="primary" 
        :loading="loading"
      >
        开始测试
      </el-button>
    </div>

    <div v-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-if="result" class="result-section">
      <table class="result-table">
        <thead>
          <tr>
            <th class="table-header">测速服务器</th>
            <th class="table-header">解析 IP</th>
            <th class="table-header">HTTP状态码</th>
            <th class="table-header">HTTPS状态码</th>
            <th class="table-header">总时间</th>
            <th class="table-header">解析时间</th>
            <th class="table-header">HTTP连接</th>
            <th class="table-header">网页大小</th>
            <th class="table-header">下载速度</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="data in result" :key="data.server">
          <tr v-if="data.error">
            <td class="table-value">{{ data.server }}</td>
            <td class="table-value" colspan="8">
              <span class="status-code status-error">{{ data.error }}</span>
            </td>
          </tr>
          <tr v-else>
            <td class="table-value">{{ data.server }}</td>
            <td class="table-value">{{ data.data?.host_record }}</td>
            <td class="table-value">
                <span class="status-code" :class="getStatusCodeClass(data.data?.http_status_code)">
                    {{ data.data?.http_status_code }}
                </span>
            </td>
            <td class="table-value">
                <span class="status-code" :class="getStatusCodeClass(data.data?.https_status_code)">
                    {{ data.data?.https_status_code }}
                </span>
            </td>
            <td class="table-value">{{ formatTime(data.data?.total_time) }}</td>
            <td class="table-value">{{ formatTime(data.data?.dns_lookup_time) }}</td>
            <td class="table-value">{{ formatTime(data.data?.first_byte_time) }}</td>
            <td class="table-value">{{ formatSize(data.data?.page_size) }}</td>
            <td class="table-value">{{ formatSpeed(data.data?.download_speed) }}</td>
          </tr>
          </template>
        </tbody>
        </table>
    </div>

  </div>
</template>

<style scoped>
@import "../style.css";
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
  font-size: 1.05em;
  color: #303133;
  text-align: left;
  border: 1px solid #dcdfe6;
}
html.dark .result-table .table-header {
  padding: 12px 15px;
  font-weight: 600;
  font-size: 1.05em;
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

<style>
:root {
  --el-color-primary: #3EAF7C;
}
html.dark {
  --el-color-primary: #3EAF7C;
}
.el-icon{
  font-size: 1.3em;
}
</style>
