<script setup lang="ts">
import { useMiddlewareFetch } from '../../utils/middleware';
import { ref, onMounted, computed, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index'
import { isIPv6 } from 'is-ip'

const route = useRoute()

useHead({
  title: `ASN查询 | IP自治系统号查询 | ${config.siteName}`,
  meta: [
    { name: 'description', content: '专业的ASN自治系统查询工具,支持IP地址ASN号查询,提供Maxmind GEOLite2、DB-IP双数据源对比,同时集成WHOIS解析获取ASN详细信息,包括组织名称、国家、注册日期等,助力网络运维和路由分析' },
    { name: 'keywords', content: 'asn查询,自治系统号,asn lookup,ip asn,asn whois,网络自治系统,运营商asn,ip归属asn' },
    { property: 'og:title', content: 'ASN自治系统查询工具 - IP ASN号与组织信息查询' },
    { property: 'og:description', content: '多数据源ASN查询,支持Maxmind GEOLite2和DB-IP,集成WHOIS解析获取ASN详细信息' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'ASN自治系统查询工具',
        description: '专业的ASN自治系统查询工具，支持IP地址ASN号查询，提供Maxmind GEOLite2、DB-IP双数据源对比，集成WHOIS解析。',
        url: config.siteUrl + 'asn',
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

interface ASNResult {
  ip: string
  geolite2_asn?: { asn: string; org: string }
  dbip_asn?: { asn: string; org: string }
  whois?: any
}

const tmpIP = ref('1.1.1.1')
const loading = ref(false)
const result = ref<ASNResult | null>(null)
const error = ref('')
const userIP = ref('')

const apiList = config.IPLocationAPI
const currentApiIndex = ref(0)
const backendID = computed(() => apiList[currentApiIndex.value]?.id || '')


const { data: asnData, error: asnError, execute: executeASN } = useMiddlewareFetch<ASNResult>(() => '/middleware/' + backendID.value + '/asn/' + tmpIP.value, {
  key: 'asn-fetch',
  immediate: false,
  watch: false,
  deep: false,
  server: false,
  lazy: true,
})

watch(asnData, (newData) => {
  if (newData) {
    result.value = newData
    loading.value = false
    error.value = ''
  }
})

watch(asnError, async (newError) => {
  if (newError) {
    console.log(newError)
    if (currentApiIndex.value < apiList.length - 1) {
      const nextIndex = currentApiIndex.value + 1
      error.value = `${(newError as any).message || '请求失败'}，正在重试 ${apiList[nextIndex]?.label || ''}...`
      currentApiIndex.value = nextIndex
      await nextTick()
      executeASN()
    } else {
      error.value = '请求失败，请检查IP或网络'
      loading.value = false
    }
  }
})

async function queryASN() {
  currentApiIndex.value = 0
  loading.value = true
  error.value = ''
  result.value = null
  tmpIP.value = tmpIP.value.trim()
  if (!tmpIP.value) {
    error.value = '请输入IP地址'
    loading.value = false
    return
  }
  await nextTick()
  executeASN()
}

onMounted(() => {
  getUserIP()
  const ipParam = route.query.ip as string
  if (ipParam) {
    tmpIP.value = ipParam
    queryASN()
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

function getStatusClass(status: string): string {
  if (status.includes('prohibited') || status.includes('lock')) return 'status-warning'
  return 'status-success'
}
</script>

<template>
  <div class="title">
    <header>
      <h1>ASN 自治系统查询</h1>
      <p>查询 IP 地址所属的自治系统号（ASN）和组织信息</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input
        v-model="tmpIP"
        placeholder="请输入IP地址（如：1.1.1.1）"
      />
      <el-button
        @click="queryASN()"
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
            <td class="table-label">IP 地址</td>
            <td class="table-value">{{ result.ip }}</td>
          </tr>
          <tr v-if="result.geolite2_asn?.asn">
            <td class="table-label">Maxmind ASN</td>
            <td class="table-value">
              AS{{ result.geolite2_asn.asn }}
              <span v-if="result.geolite2_asn.org" style="color: #999; font-size: 0.9em;">{{ result.geolite2_asn.org }}</span>
            </td>
          </tr>
          <tr v-if="result.dbip_asn?.asn">
            <td class="table-label">DB-IP ASN</td>
            <td class="table-value">
              AS{{ result.dbip_asn.asn }}
              <span v-if="result.dbip_asn.org" style="color: #999; font-size: 0.9em;">{{ result.dbip_asn.org }}</span>
            </td>
          </tr>
          <tr v-if="result.whois?.asNumber">
            <td class="table-label">ASN</td>
            <td class="table-value">AS{{ result.whois.asNumber }}</td>
          </tr>
          <tr v-if="result.whois?.asName">
            <td class="table-label">AS 名称</td>
            <td class="table-value">{{ result.whois.asName }}</td>
          </tr>
          <tr v-if="result.whois?.orgName">
            <td class="table-label">组织名称</td>
            <td class="table-value">{{ result.whois.orgName }}</td>
          </tr>
          <tr v-if="result.whois?.orgId">
            <td class="table-label">组织 ID</td>
            <td class="table-value">{{ result.whois.orgId }}</td>
          </tr>
          <tr v-if="result.whois?.country">
            <td class="table-label">国家</td>
            <td class="table-value">{{ result.whois.country }}</td>
          </tr>
          <tr v-if="result.whois?.regDate">
            <td class="table-label">注册日期</td>
            <td class="table-value">{{ result.whois.regDate }}</td>
          </tr>
          <tr v-if="result.whois?.updated">
            <td class="table-label">更新日期</td>
            <td class="table-value">{{ result.whois.updated }}</td>
          </tr>
          <tr v-if="result.whois?.abuseEmail">
            <td class="table-label">Abuse 邮箱</td>
            <td class="table-value">{{ result.whois.abuseEmail }}</td>
          </tr>
          <tr v-if="result.whois?.abusePhone">
            <td class="table-label">Abuse 电话</td>
            <td class="table-value">{{ result.whois.abusePhone }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <blockquote>
      <a href="/doc/user/asn" target="_blank">ASN 查询原理介绍</a><br/>
      ASN（Autonomous System Number）是互联网中每个自治系统的唯一标识符。<br/>
      数据来源：Maxmind GEOLite2、DB-IP，部分节点集成 WHOIS 进一步解析 ASN 详情。<br/>
      <a href="/location" target="_blank">IP 归属地查询</a> | <a href="/whois" target="_blank">Whois 查询</a><br/>
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
.el-icon{
  font-size: 1.3em;
}
</style>
