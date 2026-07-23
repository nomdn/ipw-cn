<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';
import { extractHost, formatTime, formatSpeed, formatSize, getStatusCodeClass } from '~/utils/tools';


const route = useRoute()

useHead({
  title: 'IPv4网站测速工具 | 全国多节点并发测速 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的IPv4网站测速工具,提供全国多节点并发测速服务,快速返回网站响应时间、下载速度、页面大小、DNS解析时间、HTTP连接时间等详细性能指标,支持IPv4网站性能检测与优化,助力网站性能监控与用户体验改善' },
    { name: 'keywords', content: 'ipv4网站测速,ipv4测速,网站速度测试,ipv4性能检测,网站响应时间,ipv4下载速度,ipv4性能优化,网站性能监控' },
    { property: 'og:title', content: 'IPv4网站测速 - 全国多节点并发性能检测工具' },
    { property: 'og:description', content: '全国多节点IPv4网站测速,快速获取响应时间、下载速度等性能数据' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv4网站性能测速工具',
        description: '专业的IPv4网站测速工具，全国多节点并发测速，提供响应时间、下载速度、页面大小、DNS解析时间等详细性能指标。',
        url: config.siteUrl + 'speedtest',
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

/** 服务器结果接口 */
interface ServerResult {
  label: string
  loading: boolean
  error: string
  data: any
}

/** 预构建的服务器列表，页面挂载即展示，用户可提前看到测速节点 */
const serverResults = ref<ServerResult[]>([])

// 提前构建服务器列表（学 tcping.vue 策略）
// 在组件初始化时就创建好 useFetch 实例，
// 设置 immediate: false + watch: false，
// 只在用户点击"开始测试"时才调用 execute() 发起请求，
// 避免页面加载时自动发请求浪费资源。
// =============================================

/** 双栈服务器：同时支持 IPv4/IPv6 的节点 */
const dualStackFetches = config.SpeedTest.DualStack.map((server) => {
  const url = computed(() => server.url + 'v1/speed/v4/' + domain.value);
  const { data, error: fetchError, execute } = useFetch(url, {
    immediate: false,
    watch: false,
  });
  return { label: server.label, data, error: fetchError, execute };
});

/** IPv4 专用服务器：仅支持 IPv4 的节点 */
const ipv4Fetches = config.SpeedTest.IPv4.map((server) => {
  const url = computed(() => server.url + 'v1/speed/v4/' + domain.value);
  const { data, error: fetchError, execute } = useFetch(url, {
    immediate: false,
    watch: false,
  });
  return { label: server.label, data, error: fetchError, execute };
});

/** 合并所有服务器 fetch 列表 */
const allFetches = [
  ...dualStackFetches.map(fetch => ({ ...fetch, type: 'DualStack' })),
  ...ipv4Fetches.map(fetch => ({ ...fetch, type: 'IPv4' }))
];

/**
 * 初始化服务器结果列表
 * 页面挂载时调用，让用户提前看到所有测速节点
 */
function initServerResults() {
  serverResults.value = allFetches.map((fetch) => ({
    label: fetch.label,
    loading: false,
    error: '',
    data: null
  }))
}

/**
 * 执行全站测速
 * 并发调用所有服务器的 execute() 方法，逐个更新对应行的结果
 */
async function SpeedTest() {
  domain.value = extractHost(tmpDomain.value)
  loading.value = true
  error.value = ''
  await nextTick()

  // 重置所有行状态为加载中
  serverResults.value.forEach((row) => {
    row.loading = true
    row.error = ''
    row.data = null
  })

  // 并发请求所有服务器，每个请求完成后立即更新对应行
  const promises = allFetches.map(async (fetch, index) => {
    const row = serverResults.value[index]
    if (!row) return

    try {
      await fetch.execute()
      row.data = fetch.data.value
    } catch (err: any) {
      row.error = err?.data?.error || err?.message || '请求失败'
    } finally {
      row.loading = false
    }
  })

  Promise.all(promises).finally(() => {
    loading.value = false
  })
}

// 页面挂载后，初始化服务器列表并检查 URL 参数
onMounted(() => {
  initServerResults()
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
      <h1>IPv4 网站测速</h1>
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

    <div class="result-section">
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
          <template v-for="(row, index) in serverResults" :key="index">
            <!-- 请求失败 -->
            <tr v-if="row.error">
              <td class="table-label">{{ row.label }}</td>
              <td class="table-value" colspan="8">
                <span class="status-code status-error">{{ row.error }}</span>
              </td>
            </tr>
            <!-- 加载中 -->
            <tr v-else-if="row.loading">
              <td class="table-label">{{ row.label }}</td>
              <td class="table-value" colspan="8">测速中...</td>
            </tr>
            <!-- 有结果 -->
            <tr v-else-if="row.data">
              <td class="table-label">{{ row.label }}</td>
              <td class="table-value">{{ row.data?.host_record }}</td>
              <td class="table-value">
                  <span class="status-code" :class="getStatusCodeClass(row.data?.http_status_code)">
                      {{ row.data?.http_status_code }}
                  </span>
              </td>
              <td class="table-value">
                  <span class="status-code" :class="getStatusCodeClass(row.data?.https_status_code)">
                      {{ row.data?.https_status_code }}
                  </span>
              </td>
              <td class="table-value">{{ formatTime(row.data?.total_time) }}</td>
              <td class="table-value">{{ formatTime(row.data?.dns_lookup_time) }}</td>
              <td class="table-value">{{ formatTime(row.data?.first_byte_time) }}</td>
              <td class="table-value">{{ formatSize(row.data?.page_size) }}</td>
              <td class="table-value">{{ formatSpeed(row.data?.download_speed) }}</td>
            </tr>
            <!-- 等待测速 -->
            <tr v-else>
              <td class="table-label">{{ row.label }}</td>
              <td class="table-value" colspan="8">-</td>
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

.result-table .table-header {
  font-size: 1.05em;
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
