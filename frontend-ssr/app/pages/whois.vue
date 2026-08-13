<script setup lang="ts">
import { useMiddlewareFetch } from '../../utils/middleware';
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index'
import { isIPv6 } from 'is-ip'

const route = useRoute()

useHead({
  title: 'Whois查询 | ' + (route.query.site || '柠檬味ipw.cn'),
  titleTemplate: '%s',
  link: [
    { rel: 'canonical', href: computed(() => new URL(route.path, config.siteUrl).toString()).value }
  ],
  meta: [
    { name: 'description', content: '专业的Whois域名查询工具,支持.com、.net等国内外域名WHOIS信息查询,提供域名注册商、注册时间、到期时间、DNS服务器等详细信息,助力域名管理和交易决策' },
    { name: 'keywords', content: 'whois查询,域名whois,whois信息查询,域名注册信息,域名到期时间,域名注册商,dns服务器查询,域名whois工具' },
    { property: 'og:title', content: 'Whois域名查询工具 - 域名注册信息在线查询' },
    { property: 'og:description', content: '专业的Whois域名查询工具,支持国内外域名WHOIS信息查询,提供域名注册商、注册时间、到期时间等详细信息' },
    { property: 'og:image', content: `${config.siteUrl}favicon.svg` },
    { property: 'og:type', content: 'website' },
    { property: 'og:url', content: computed(() => new URL(route.path, config.siteUrl).toString()).value },
    { name: 'twitter:card', content: 'summary_large_image' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'Whois域名查询工具',
        description: '专业的Whois域名查询工具,支持国内外域名WHOIS信息查询,提供域名注册商、注册时间、到期时间、DNS服务器等详细信息',
        url: config.siteUrl + 'whois',
        applicationCategory: 'DeveloperApplication',
        operatingSystem: 'Any',
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'CNY'
        },
        featureList: 'Whois查询,域名注册信息,域名到期时间,域名注册商查询,DNS服务器查询',
        about: [
          {
            '@type': 'Thing',
            name: 'Whois查询',
            description: '通过Whois协议查询域名的注册信息,包括注册商、注册时间、到期时间、DNS服务器等技术服务。'
          },
          {
            '@type': 'Thing',
            name: '域名管理',
            description: '通过Whois查询结果,域名持有者可以了解域名到期时间,及时续费避免域名被释放或高价赎回。'
          }
        ]
      })
    }
  ]
});

const domain = ref('')
const tmpdomain = ref('zakoflare.com')
const loading = ref(false)
const result = ref<any>(null)
const error = ref('')
const userIP = ref('')

const apiList = config.apiBaseUrls
const currentApiIndex = ref(0)
const backendID = computed(() => apiList[currentApiIndex.value]?.id || '')
const whoisUrl = computed(() => '/middleware/' + backendID.value + '/whois/' + domain.value)

const { data: whoisData, error: whoisError, execute: executeWhois } = useMiddlewareFetch<any>(whoisUrl, {
  immediate: false,
  watch: false,
})

watch(whoisData, (newData) => {
  if (newData) {
    result.value = newData
    loading.value = false
    error.value = ''
  }
})

watch(whoisError, async (newError) => {
  if (newError) {
    console.log(newError)
    if (currentApiIndex.value < apiList.length - 1) {
      const nextIndex = currentApiIndex.value + 1
      error.value = `${(newError as any).message || '请求失败'}，正在重试 ${apiList[nextIndex]?.label || ''}...`
      currentApiIndex.value = nextIndex
      await nextTick()
      executeWhois()
    } else {
      error.value = '请求失败，请检查域名或网络'
      loading.value = false
    }
  }
})

async function queryWhois() {
  currentApiIndex.value = 0
  loading.value = true
  error.value = ''
  result.value = null
  domain.value = tmpdomain.value.trim()
  if (!domain.value) {
    error.value = '请输入域名'
    loading.value = false
    return
  }
  nextTick(() => {
    executeWhois()
  })
}

onMounted(() => {
    getUserIP()
    const whoisParam = route.query.site as string
    if (whoisParam) {
        tmpdomain.value = whoisParam
        queryWhois()
    }
})

async function getUserIP(){
  await $fetch<string>(config.DualStackAPI).then(
    function (data){
      userIP.value = data
    }
  )
  return userIP.value
}

const { data: page } = await useAsyncData('whois-doc', () =>
  $fetch('/api/markdown/whois')
)
const doc = page.value

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const d = new Date(dateStr)
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function getStatusClass(status: string): string {
  if (status.includes('prohibited') || status.includes('lock')) return 'status-warning'
  return 'status-success'
}
</script>

<template>
  <div class="title">
    <header>
      <h1>Whois 域名查询</h1>
      <p>查询域名注册信息、注册商、到期时间等 Whois 数据</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input
        v-model="tmpdomain"
        placeholder="请输入域名（如：example.com）"
      />
      <el-button
        @click="queryWhois()"
        type="primary"
        :loading="loading"
      >
        查询
      </el-button>
    </div>

    <div v-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-if="result" class="result-section">
      <table class="result-table">
        <thead>
          <tr>
            <th class="table-header">项目</th>
            <th class="table-header">信息</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td class="table-label">域名</td>
            <td class="table-value">{{ result.domain }}</td>
          </tr>
          <tr v-if="result.status?.length">
            <td class="table-label">域名状态</td>
            <td class="table-value">
              <span v-for="(s, i) in result.status" :key="i" :class="getStatusClass(s)" class="status-code" style="margin-right: 4px;">
                {{ s }}
              </span>
            </td>
          </tr>
          <tr v-if="result.registrar?.name">
            <td class="table-label">注册商</td>
            <td class="table-value">
              {{ result.registrar.name }}
              <span v-if="result.registrar.ianaId" style="color: #999; font-size: 0.9em;">(IANA ID: {{ result.registrar.ianaId }})</span>
            </td>
          </tr>
          <tr v-if="result.registrant && Object.keys(result.registrant).length">
            <td class="table-label">注册人</td>
            <td class="table-value">
              <div v-if="result.registrant.name">{{ result.registrant.name }}</div>
              <div v-if="result.registrant.org">{{ result.registrant.org }}</div>
              <div v-if="result.registrant.phone">{{ result.registrant.phone }}</div>
              <div v-if="result.registrant.email">{{ result.registrant.email }}</div>
              <div v-if="result.registrant.province">{{ result.registrant.province }}</div>
              <a v-if="result.registrant.contactUri" :href="result.registrant.contactUri" target="_blank" rel="noreferrer">查看注册信息</a>
            </td>
          </tr>
          <tr v-if="result.technical && Object.keys(result.technical).length">
            <td class="table-label">技术联系人</td>
            <td class="table-value">
              <div v-if="result.technical.name">{{ result.technical.name }}</div>
              <div v-if="result.technical.org">{{ result.technical.org }}</div>
              <div v-if="result.technical.phone">{{ result.technical.phone }}</div>
              <div v-if="result.technical.email">{{ result.technical.email }}</div>
            </td>
          </tr>
          <tr v-if="result.abuseContact && Object.keys(result.abuseContact).length">
            <td class="table-label">abuse 联系人</td>
            <td class="table-value">
              <div v-if="result.abuseContact.name">{{ result.abuseContact.name }}</div>
              <div v-if="result.abuseContact.phone">{{ result.abuseContact.phone }}</div>
              <div v-if="result.abuseContact.email">{{ result.abuseContact.email }}</div>
            </td>
          </tr>
          <tr v-if="result.dates?.registration">
            <td class="table-label">注册时间</td>
            <td class="table-value">{{ formatDate(result.dates.registration) }}</td>
          </tr>
          <tr v-if="result.dates?.expiration">
            <td class="table-label">到期时间</td>
            <td class="table-value">{{ formatDate(result.dates.expiration) }}</td>
          </tr>
          <tr v-if="result.dates?.lastChanged">
            <td class="table-label">最后修改</td>
            <td class="table-value">{{ formatDate(result.dates.lastChanged) }}</td>
          </tr>
          <tr v-if="result.nameservers?.length">
            <td class="table-label">DNS 服务器</td>
            <td class="table-value">
              <div v-for="ns in result.nameservers" :key="ns">{{ ns }}</div>
            </td>
          </tr>
          <tr v-if="result.whoisServer">
            <td class="table-label">Whois 服务器</td>
            <td class="table-value">{{ result.whoisServer }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <blockquote>
      数据来源：后端 WHOIS 服务，通过 IANA 注册局 bootstrap 文件自动匹配对应 WHOIS 服务器。<br/>
      部分注册局对注册人信息做了隐私保护，将显示为 "Redacted for Privacy"。<br/>
      如需查看完整注册信息，可点击 "查看注册信息" 链接跳转到注册局官网查询。<br/>
      访客IP: {{ userIP || '获取中...' }} 您的网络{{ isIPv6(userIP) ? 'IPv6' : 'IPv4' }}优先<br/>
    </blockquote>

    <div v-if="doc" class="markdown">
      <div v-html="doc"></div>
    </div>
  </div>
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";
.markdown :deep(a) {
  color: #3EAF7C !important;
  font-size: 1.3em;
  text-decoration: none
}
@media (max-width: 768px) {
  .markdown :deep(a) {
    font-size: 1.1em;
  }
  .markdown :deep(img) {
    height: auto;
    max-width: 100%;
  }
}
.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

.status-code {
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  display: inline-block;
  margin-bottom: 2px;
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
