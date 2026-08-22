<script setup lang="ts">
import { useMiddlewareFetch } from '../../utils/middleware';
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index'
import { isIPv6 } from 'is-ip'


const route = useRoute()

useHead({
  title: `DNSSEC查询 | 域名DNSSEC验证检测 | ${config.siteName}`,
  meta: [
    { name: 'description', content: '专业的DNSSEC查询工具,支持域名DNSSEC验证检测,检查域名是否启用DNSSEC签名,验证DNSKEY、RRSIG、DS记录链式信任关系,检测域名DNS数据完整性和真实性,防止DNS欺骗和中间人攻击,保障域名解析安全' },
    { name: 'keywords', content: 'dnssec查询,dnssec验证,dnskey,rrsig,ds记录,域名安全,链式信任,dns欺骗防护,域名解析安全,dns签名验证' },
    { property: 'og:title', content: 'DNSSEC查询工具 - 域名DNSSEC验证与安全检测' },
    { property: 'og:description', content: 'DNSSEC验证检测工具,检查域名DNSSEC签名状态,验证DNSKEY、RRSIG、DS记录链式信任关系' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'DNSSEC验证查询工具',
        description: '专业的DNSSEC验证查询工具，检测域名DNSSEC签名状态，验证DNSKEY、RRSIG、DS记录链式信任关系，保障域名解析安全。',
        url: config.siteUrl + 'dnssec',
        applicationCategory: 'DeveloperApplication',
        operatingSystem: 'Web',
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'CNY'
        },
        provider: {
          '@type': 'Organization',
          name: config.siteName
        }
      })
    }
  ]
})

interface DNSSECResult {
  domain: string
  enabled: boolean
  valid: boolean
  has_rrsig: boolean
  has_dnskey: boolean
  has_ds: boolean
  algorithm: number
  key_tag: number
  signer_name: string
  validation: string
  duration: number
}

interface ServerResult {
  label: string
  loading: boolean
  error?: string
  data?: DNSSECResult
}

const tmpDomain = ref('www.zakoflare.com')
const loading = ref(false)
const serverResults = ref<ServerResult[]>([])
const userIP = ref('')

const apiList = [
  ...config.APIBaseURL.DualStack,
  ...config.APIBaseURL.IPv4,
  ...config.APIBaseURL.IPv6
]

const dnssecFetches = apiList.map((server) => {
  const url = computed(() => "/middleware/" + server.id + "/dnssec/" + tmpDomain.value)
  const { data, error, execute } = useMiddlewareFetch<DNSSECResult>(url, {
    immediate: false,
    watch: false,
  })
  return { label: server.label, data, error, execute }
})

function initServerResults() {
  serverResults.value = apiList.map((server) => ({
    label: server.label,
    loading: false,
  }))
}

async function checkDNSSEC() {
  const domain = tmpDomain.value.trim()
  if (!domain) return

  loading.value = true

  serverResults.value.forEach((result) => {
    result.loading = true
    result.data = undefined
  })

  const promises = dnssecFetches.map(async (fetch, index) => {
    try {
      await fetch.execute()
      const result = serverResults.value[index]
      if (result) {
        result.data = fetch.data.value
      }
    } catch (err) {
      console.error(err)
      const result = serverResults.value[index]
      if (result) {
        result.error = (err as any)?.message || '请求失败'
      }
    } finally {
      const result = serverResults.value[index]
      if (result) {
        result.loading = false
      }
    }
  })

  Promise.allSettled(promises).finally(function () {
    loading.value = false
  })
}

function getValidationClass(result: DNSSECResult): string {
  if (result.valid) return 'status-success'
  if (result.has_dnskey || result.has_rrsig) return 'status-warning'
  return 'status-error'
}

function formatDuration(ms: number): string {
  return ms.toFixed(2) + ' ms'
}

onMounted(() => {
  initServerResults()
  getUserIP()
  const domainParam = route.query.domain as string
  if (domainParam) {
    tmpDomain.value = domainParam
    checkDNSSEC()
  }
})

async function getUserIP() {
  await $fetch<string>(config.DualStackAPI).then(
    function (data) {
      userIP.value = data
    }
  )
  return userIP.value
}
</script>

<template>
  <div class="title">
    <header>
      <h1>DNSSEC 验证查询</h1>
      <p>检测域名 DNSSEC 签名状态和链式信任验证</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input
        v-model="tmpDomain"
        placeholder="请输入域名（如：example.com）"
      />
      <el-button
        @click="checkDNSSEC()"
        type="primary"
        :loading="loading"
      >
        验证
      </el-button>
    </div>



    <div class="result-section" v-if="serverResults.some(r => r.data)">
      <table class="result-table">
        <thead>
          <tr>
            <th class="table-header">DNS 服务器</th>
            <th class="table-header">DNSSEC</th>
            <th class="table-header">DNSKEY</th>
            <th class="table-header">RRSIG</th>
            <th class="table-header">DS</th>
            <th class="table-header">验证结果</th>
            <th class="table-header">耗时</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="(server, index) in serverResults" :key="index">
            <tr v-if="server.data">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value">
                <span :class="server.data.enabled ? 'status-success' : 'status-error'" class="status-code">
                  {{ server.data.enabled ? '已启用' : '未启用' }}
                </span>
              </td>
              <td class="table-value">
                <span :class="server.data.has_dnskey ? 'status-success' : 'status-error'" class="status-code">
                  {{ server.data.has_dnskey ? '有' : '无' }}
                </span>
              </td>
              <td class="table-value">
                <span :class="server.data.has_rrsig ? 'status-success' : 'status-error'" class="status-code">
                  {{ server.data.has_rrsig ? '有' : '无' }}
                </span>
              </td>
              <td class="table-value">
                <span :class="server.data.has_ds ? 'status-success' : 'status-error'" class="status-code">
                  {{ server.data.has_ds ? '有' : '无' }}
                </span>
              </td>
              <td class="table-value">
                <span :class="getValidationClass(server.data)" class="status-code">
                  {{ server.data.valid ? '验证通过' : '验证失败' }}
                </span>
              </td>
              <td class="table-value">{{ formatDuration(server.data.duration) }}</td>
            </tr>
            <tr v-else-if="server.loading">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value" colspan="6">加载中...</td>
            </tr>
            <tr v-else-if="server.error">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value error-text" colspan="6">{{ server.error }}</td>
            </tr>
            <tr v-else-if="!server.data">
              <td class="table-label">{{ server.label }}</td>
              <td class="table-value" colspan="6">-</td>
            </tr>
          </template>
        </tbody>
      </table>
    </div>

    <blockquote>
      <a href="/doc/user/dnssec" target="_blank">DNSSEC 原理介绍</a><br/>
      DNSSEC 通过 DNSKEY、RRSIG、DS 记录为 DNS 数据提供完整性验证，防止 DNS 欺骗攻击。<br/>
      如果显示 "验证失败"，可能原因：域名未配置 DNSSEC、DNSKEY 密钥不匹配、DS 记录缺失等。<br/>
      <a href="/dns" target="_blank">DNS 解析查询</a> | <a href="/whois" target="_blank">Whois 查询</a><br/>
      访客IP: {{ userIP || '获取中...' }}，您的网络{{ isIPv6(userIP) ? 'IPv6' : 'IPv4' }}优先<br/>
    </blockquote>
  </div>
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";

.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

.result-table .table-label {
  width: 180px;
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
.el-icon{
  font-size: 1.3em;
}
</style>
