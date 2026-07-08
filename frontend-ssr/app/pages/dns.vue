<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';
import { isIPv6 } from 'is-ip';
const route = useRoute()

useHead({
  title: 'DNS查询工具 | 多节点域名解析检测 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的多节点DNS查询工具,支持A记录、AAAA记录、CNAME记录、MX记录、NS记录、TXT记录、SRV记录、CAA记录等多种DNS解析记录查询,提供全国多节点并发检测,快速返回DNS解析结果,助力域名解析问题排查与优化' },
    { name: 'keywords', content: 'dns查询,dns解析,域名解析,a记录查询,aaaa记录,cname记录,mx记录,ns记录,txt记录,srv记录,dns服务器,域名dns检测' },
    { property: 'og:title', content: 'DNS查询工具 - 多节点域名解析记录检测' },
    { property: 'og:description', content: '多节点DNS查询工具,支持多种DNS记录类型查询,快速检测域名解析状态' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'DNS查询与解析检测工具',
        description: '专业的多节点DNS查询工具，支持A、AAAA、CNAME、MX、NS、TXT、SRV、CAA等多种记录类型，全国多节点并发检测。',
        url: config.siteUrl + 'dns',
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

const tmpDomain = ref('www.zakoflare.com')
const domain = ref('')
const recordType = ref('a')
const loading = ref(false)
const results = ref<any>([])
const isloading = ref(false)
const nowRecordType = ref('')
const userIP = ref('')

async function getUserIP(){
  
  await $fetch<string>(config.DualStackAPI).then(
  function (data){
    userIP.value = data
  })
  return userIP.value
}

function formatTime(ms: number): string {
  if (ms == null) return '-'
  if (ms < 1000) {
    return `${ms} ms`
  }
  return `${(ms / 1000).toFixed(2)} s`
}

const recordTypes = [
  { value: 'a', label: 'A 记录' },
  { value: 'aaaa', label: 'AAAA 记录' },
  { value: 'cname', label: 'CNAME 记录' },
  { value: 'mx', label: 'MX 记录' },
  { value: 'ns', label: 'NS 记录' },
  { value: 'txt', label: 'TXT 记录' },
  { value: 'srv', label: 'SRV 记录' },
  { value: 'caa', label: 'CAA 记录' },
  { value: 'ptr', label: 'PTR 记录' }
]

const dnsServerFetches = config.NSLookup.map((server) => {
  const url = computed(() => server.url + 'v1/dns/' + recordType.value + "/" + domain.value);
  const { data, error, execute } = useFetch(url, {
    immediate: false,
    watch: false,
  });
  return { label: server.label, data, error, execute };
});

async function queryDNS() {
  isloading.value = true
  domain.value = tmpDomain.value
  await nextTick()
  
  // 初始化结果数组，保持响应式结构
  results.value = dnsServerFetches.map(fetch => ({
    server: fetch.label,
    loading: true,
    data: null,
    error: null
  }));

  const promises = dnsServerFetches.map(async (fetch, index) => {
    try {
      await fetch.execute();
      
      //  直接更新对应索引的结果，保持响应式
      results.value[index] = {
        server: fetch.label,
        loading: false,
        data: fetch.data.value,
        error: null
      };
      
      return {
        server: fetch.label,
        data: fetch.data.value
      };
    } catch (err) {
      //  错误时也更新对应索引
      results.value[index] = {
        server: fetch.label,
        loading: false,
        data: null,
        error: err
      };
      
      return {
        server: fetch.label,
        error: err
      };
    }
  });

  const promiseResults = await Promise.all(promises)
  console.log(promiseResults)
  isloading.value = false
  nowRecordType.value = recordType.value
  return promiseResults
}


onMounted(() => {
  
  const domainParam = route.query.domain as string
  const typeParam = route.query.type as string
  if (domainParam) {
    tmpDomain.value = domainParam
  }
  if (typeParam && recordTypes.some(t => t.value === typeParam)) {
    recordType.value = typeParam
  }
  if (domainParam) {
    queryDNS()
  }
  getUserIP()
})
const { data: page } = await useAsyncData('dns', () =>
  queryCollection('content').path('/dns').first()
);
</script>

<template>
  <div class="title">
    <header>
      <h1>DNS查询</h1>
      <p>多节点 DNS 查询，检测域名解析记录</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input 
        v-model="tmpDomain" 
        placeholder="请输入域名（如：example.com）" 
      />
      <el-select v-model="recordType" style="width: 150px;" class="custom-height-select">
        <el-option 
          v-for="item in recordTypes" 
          :key="item.value" 
          :label="item.label" 
          :value="item.value" 
          
        />
      </el-select>
      <el-button 
        @click="queryDNS()" 
        type="primary" 
        :loading="loading"
      >
        查询
      </el-button>
    </div>
        <div class="result-section">
        <table class="result-table" v-if="results.length > 0">
        <thead>
          <tr>
            
            <th class="table-header">服务器</th>
            <th class="table-header">类型</th>
            <th class="table-header">记录</th>
            <th class="table-header">记录数</th>
            <th class="table-header">耗时</th>
            <th class="table-header">TTL (S)</th>
            
          </tr>
        </thead>
        <tbody>
          <tr v-for="(result) in results" :key="result.server">
            <td class="table-label">{{result.server}}</td>
            <td class="table-value">{{nowRecordType || '--'}}</td>
            
            <td class="table-value" style="text-align: center;">
              <template v-if="result && result.data?.record">
                <div v-for="(ip, index) in result.data.record.slice(0, 5)" :key="index" class="ip-address">
                  <span>{{ ip }}</span>
                </div>
              </template>
              
              <span v-else-if="!isloading && " class="status-code" style="color: #F56C6C; background: #fef0f0;">
                失败 {{ result.error }}
              </span>
              <span v-else-if="isloading" class="status-code" style="color: #909399; background: #f4f4f5;">
                加载中
              </span>
            </td>
            
            <td class="table-value">{{result.data?.record?.length || 0}}</td>
            <td class="table-value">{{formatTime(result.data?.duration)}}</td>
            <td class="table-value">{{result.data?.ttl}}</td>
          </tr>
        </tbody>
        
        </table>
    </div>
    <blockquote>
        <span class="quate">A</span> 将域名指向一个 IPv4 地址，如 <span class="quate">106.55.75.123</span>。 A记录归属地是柠檬超绝大杂烩免费接口，随便用。<br/>

        <span class="quate">AAAA</span> 将域名指向一个 IPv6 地址，如 <span class="quate">2402:4e00:1013:e500:0:9671:f018:4947</span>。同一个主机名可以同时解析到 IPv4(A记录)地址 和 IPv6(AAAA 记录)地址上，当只有IPv4 地址的用户会解析到 IPv4 地址，一般情况下有 IPv6 地址的用户会优先解析到 IPv6 地址。<br/>

       <span class="quate">CNAME</span> 将域名指向另一个域名地址，与其保持相同解析，如 <span class="quate">ipw.wsmdn.top</span> 别名到 <span class="quate">ipw.wsmdn.top.eo.dnse1.com</span>.<br/>

        <span class="quate">MX</span> 用于邮件服务器，一般由邮件注册商提供，如 <span class="quate">mxbiz1.qq.com</span>。如果邮箱格式为 test@<span class="quate">wsmdn.top</span> 则输入 <span class="quate">wsmdn.top</span> 查询。如果邮箱格式为 test@<span class="quate">mail.wsmdn.top</span>则输入<span class="quate">mail.wsmdn.top</span>查询。推荐2个免费的企业邮箱：腾讯企业邮、网易免费企业邮。<br/>

        <span class="quate">TXT</span> 附加文本信息，常用于域名所有权验证，如在申请 HTTPS 证书时需要增加记录、<br/>

        <span class="quate">PTR</span> IP 的反向解析记录，例如 <span class="quate">159.75.190.197</span> 反解析到 <span class="quate">wsmdn.top</span>，一般用于提升自建域名邮件服务器的可信度，可提单找云服务商添加。<br/>

        <span class="quate">NS</span> 域名的 DNS 服务器地址，例如 <span class="quate">ns3.dnsv2.com</span>，推荐 华为云DNS.<br/>

        网站开启IPv6检测 网站开启IPv6检测 | SSL证书在线检查<br/>

        访客IP: {{userIP }}，您的网络 {{ isIPv6(userIP) ? 'IPv6' : 'IPv4'}} 访问优先<br/>
    </blockquote>
    <div class="markdown">
    <ContentRenderer v-if="page" :value="page" />
    </div>
    </div>

</template>

<style scoped>
@import "../style.css";

.markdown :deep(a) {
    color: #3EAF7C !important;
    font-size: 1.3em;
    text-decoration: none
}
.el-input {
  width: 420px;
  height: 50px;
  font: 1.3em sans-serif;
  margin-right: 10px;
}
.custom-height-select {
  --el-component-size: 50px; 
}

.el-select {
  margin-right: 10px;
}
/* 2. 针对新版 Element Plus (>= 2.4.0) 的 el-select 结构 */
:deep(.el-select__wrapper) {
  height: 50px !important;
  min-height: 50px !important;
  padding-top: 0 !important;
}



/* 让选中的文字垂直居中 */
:deep(.el-select__selected-item) {
  line-height: 48px !important;
  height: 48px !important;
  font-size: 16px;
}

/* 右侧下拉箭头垂直居中 */
:deep(.el-select__caret) {
  font-size: 18px;
  line-height: 50px !important;
}

/* 3. 兼容老版本 Element Plus (< 2.4.0)，以防万一 */
:deep(.el-input__wrapper) {
  height: 50px !important;
  box-shadow: 0 0 0 1px var(--el-border-color) inset !important;
}
:deep(.el-input__inner) {
  height: 48px !important;
  line-height: 48px !important;
  font-size: 16px;
}
.el-button {
  width: 165px;
  height: 50px;
  font: 1.3em sans-serif;
}

.result-section {
  margin-top: 30px;
}

.server-block {
  margin-bottom: 30px;
}

.server-block h3 {
  margin-bottom: 15px;
  color: #303133;
  font-size: 1.2em;
}

html.dark .server-block h3 {
  color: #cfcfcf;
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
  text-align: center;
}

html.dark .result-table .table-value {
  color: #cfcfcf;
  border: 1px solid #1a1919;
}
html.dark .result-table thead tr {
  background: #2e2d2d;
}
.ip-value {
    width: 100%;
    height: 100%;
    margin: 0;
}

.ip-address {
  display: block;
  text-align: center;
  padding: 2px 0;
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
  width: 100px;
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
  word-break: break-all;
}

html.dark .result-table .table-value {
  color: #cfcfcf;
  border: 1px solid #1a1919;
}

.status-message {
  padding: 15px;
  color: #909399;
  text-align: center;
}

.error-message {
  padding: 15px;
  background: #fef0f0;
  color: #F56C6C;
  border-radius: 6px;
  text-align: center;
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
