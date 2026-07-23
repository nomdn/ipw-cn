<script setup lang="ts">
import { ref, onMounted, computed, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { CircleCheckFilled, CircleCloseFilled,InfoFilled } from '@element-plus/icons-vue';
import { config } from '../../config/index';
import { extractHost, formatTime, formatSpeed, getStatusCodeClass } from '~/utils/tools';
const route = useRoute()

useHead({
  title: 'SSL证书检测工具 | IPv4/IPv6证书检查 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的SSL证书检测工具,全面检查网站的IPv4和IPv6 SSL证书状态、有效期、签发机构、HTTP版本等信息,支持HTTPS状态码检测、下载速度测试,帮助网站管理员及时发现证书问题,确保网站安全访问' },
    { name: 'keywords', content: 'ssl证书检测,ssl检查,https证书,ipv6 ssl,ipv4 ssl,证书有效期,ssl状态,https检测,网站安全,证书签发机构' },
    { property: 'og:title', content: 'SSL证书检测 - IPv4/IPv6双栈证书状态检查工具' },
    { property: 'og:description', content: '全面检测网站SSL证书状态,支持IPv4和IPv6双栈检测,提供证书有效期、签发机构等详细信息' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'SSL证书检测工具',
        description: '专业的SSL证书检测工具，支持IPv4和IPv6 SSL证书状态、有效期、签发机构、HTTP版本等检测，提供HTTPS状态码检测、下载速度测试。',
        url: config.siteUrl + 'ssl',
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

interface SSLCheckResponse {
  ipv4: SSLCheckItem
  ipv6: SSLCheckItem
}

interface SSLCheckItem {
  cert_validity_days: number
  cert_start_time: string
  cert_end_time: string
  http_version: string
  host_record: string
  https_status_code: number
  total_time: number
  download_speed: number
  domain: string
  issuer_organization: string[] | null
  issuer_common_name: string
  subject_common_name: string
  is_expired: boolean
  is_reachable: boolean
}
const remoteAPI = ref(config.apiBaseUrl)
const tmpDomain = ref('https://www.zakoflare.com')
const testDomain = ref('')
const loading = ref(false)
const error = ref('')
const result = ref<SSLCheckResponse | null>(null)
const sslUrl = computed(() => remoteAPI.value + 'v1/ssl/' + testDomain.value);

const { data: sslData, error: sslError, execute: executeSSL } = useFetch<SSLCheckResponse>(sslUrl, {
  immediate: false,
  watch: false,
});

watch(sslData, (newData) => {
  if (newData) {
    result.value = newData;
    loading.value = false;
  }
});

watch(sslError, (newError) => {
  if (newError) {
    console.log(newError);
    error.value = '请求失败，请检查域名或网络';
    loading.value = false;
  }
});

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function checkSSL() {
  testDomain.value = extractHost(tmpDomain.value)
  loading.value = true
  error.value = ''
  result.value = null
  nextTick(() => {
    executeSSL();
  });
}
onMounted(() => {
  const urlParam = route.query.site as string
  if (urlParam) {
    tmpDomain.value = urlParam
    checkSSL()
  }
})
</script>

<template>
  <div class="title">
    <header>
      <h1>SSL证书检查</h1>
      <p>检查网站是否开启 IPv4 和 IPv6 SSL 证书</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input 
        v-model="tmpDomain" 
        placeholder="请输入域名（如：https://zakoflare.com）" 
      />
      <el-button 
        @click="checkSSL()" 
        type="primary" 
        :loading="loading"
      >
        SSL 证书检查
      </el-button>
    </div>

    <div v-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-if="result && result.ipv4" class="result-section">
      <table class="result-table">
        <thead>
          <tr>
            <th class="table-header">检测项目</th>
            <th class="table-header">IPv4</th>
            <th class="table-header">IPv6</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td class="table-label">证书状态</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">
                <span :class="result.ipv4!.is_expired ? 'expired' : 'valid'">
                  {{ result.ipv4!.is_expired ? '已过期' : '有效' }}
                </span>
              </template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">
                <span :class="result.ipv6.is_expired ? 'expired' : 'valid'">
                  {{ result.ipv6.is_expired ? '已过期' : '有效' }}
                </span>
              </template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">常用名称</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">{{ result.ipv4!.subject_common_name || '-'}}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">{{ result.ipv6.subject_common_name || '-' }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">签发者</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">{{ result.ipv4!.issuer_organization?.join(', ')|| '-' }}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">{{ result.ipv6.issuer_organization?.join(', ') || '-' }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr v-if="result.ipv6 && result.ipv6.is_reachable && result.ipv4!.is_reachable && result.ipv6.cert_validity_days > 0 && result.ipv4.cert_validity_days > 0">
            <td class="table-label">证书有效期 (天)</td>
            <td class="table-value">{{ result.ipv4!.cert_validity_days || '-'}} 天</td>
            <td class="table-value">{{ result.ipv6?.cert_validity_days || '-' }} 天</td>
          </tr>
          <tr v-else-if="result.ipv6 && result.ipv6.is_reachable && result.ipv4!.is_reachable && result.ipv6.cert_validity_days <= 0 && result.ipv4.cert_validity_days <= 0">
            <td class="table-label">证书已过期（天）</td>
            <td class="table-value">{{ Math.abs(result.ipv4.cert_validity_days) || '-'}}</td>
            <td class="table-value">{{ Math.abs(result.ipv6.cert_validity_days) || '-' }}</td>
          </tr>
          <tr v-else-if="result.ipv4!.is_reachable && result.ipv4.cert_validity_days > 0 && (!result.ipv6 || !result.ipv6.is_reachable)">
            <td class="table-label">证书有效期 (天)</td>
            <td class="table-value">{{ result.ipv4!.cert_validity_days }} 天</td>
            <td class="table-value">-</td>
          </tr>
          <tr v-else-if="result.ipv4!.is_reachable && result.ipv4.cert_validity_days <= 0 && (!result.ipv6 || !result.ipv6.is_reachable)">
            <td class="table-label">证书已过期（天）</td>
            <td class="table-value">{{ Math.abs(result.ipv4.cert_validity_days) || '-'}}</td>
            <td class="table-value">-</td>
          </tr>

          <tr>
            <td class="table-label">证书开始时间</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable && result.ipv4!.cert_start_time">{{ formatDate(result.ipv4!.cert_start_time) }}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable && result.ipv6.cert_start_time">{{ formatDate(result.ipv6.cert_start_time) }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">证书结束时间</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable && result.ipv4!.cert_end_time">{{ formatDate(result.ipv4!.cert_end_time) }}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable && result.ipv6.cert_end_time">{{ formatDate(result.ipv6.cert_end_time) }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">HTTP 版本</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">{{ result.ipv4!.http_version || '-'}}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">{{ result.ipv6.http_version || '-' }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">主机记录</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">{{ result.ipv4!.host_record || '-'}}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">{{ result.ipv6.host_record || '-' }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">HTTPS 访问返回码</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">
                <span :class="getStatusCodeClass(result.ipv4!.https_status_code)" class="status-code">
                  {{ result.ipv4!.https_status_code }}
                </span>
              </template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">
                <span :class="getStatusCodeClass(result.ipv6.https_status_code)" class="status-code">
                  {{ result.ipv6.https_status_code }}
                </span>
              </template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">总耗时</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">{{ formatTime(result.ipv4.total_time) }}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">{{ formatTime(result.ipv6.total_time) }}</template>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">下载速度</td>
            <td class="table-value">
              <template v-if="result.ipv4!.is_reachable">{{ formatSpeed(result.ipv4!.download_speed) }}</template>
              <span v-else>-</span>
            </td>
            <td class="table-value">
              <template v-if="result.ipv6 && result.ipv6.is_reachable">{{ formatSpeed(result.ipv6.download_speed) }}</template>
              <span v-else>-</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="result && result.ipv4 && result.ipv4.is_reachable && !result.ipv4.is_expired && (!result.ipv6 || (!result.ipv6.is_expired && result.ipv6.is_reachable))">
      <h3>结论：<el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>网站{{ extractHost(testDomain) }} 证书有效 </h3>
      <p><el-icon><InfoFilled style="color: lightgreen;"/></el-icon>请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。</p>
        <img src="/ssl-s1.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s2.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s2.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s3.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s3.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s4.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s4.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s5.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s5.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s6.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s6.svg"&gt;&lt;/a&gt;</code></pre>

    </div>
    <div v-if="result && result.ipv4 && result.ipv4.is_reachable && !result.ipv4.is_expired && result.ipv6 && result.ipv6.is_reachable && result.ipv6.is_expired">
      <h3>结论：<el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>网站{{ extractHost(testDomain) }} 证书有效,但不支持IPv6访问 </h3>
      <p><el-icon><InfoFilled style="color: lightgreen;"/></el-icon>请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。</p>
        <img src="/ssl-s1.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s2.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s2.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s3.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s3.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s4.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s4.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s5.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s5.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s6.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ssl/?site={{ extractHost(testDomain) }}" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s6.svg"&gt;&lt;/a&gt;</code></pre>

    </div>
    <div v-else-if="result && result.ipv4 && result.ipv4.is_reachable && result.ipv4.is_expired">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ testDomain }} 证书无效 </h3>
      <h2>都没有证书了这网站还活啥</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div>
    <div v-else-if="result && result.ipv4 && !result.ipv4.is_reachable && result.ipv6 && !result.ipv6.is_reachable">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ testDomain }} 不可达 </h3>
      <h2>...</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div> 
    <blockquote>
      网站不支持 IPv6 SSL 可能原因：<br/>
      <br/>
      1. 网站所在服务器未开启 IPv6，请参考 <a href="/doc/server/website_enable_ipv6" target="_blank">网站开启 IPv6 的三种方式</a><br/>
       2. 网站所在服务器已开启 IPv6，但防火墙未对源地址是 IPv6 地址(::/0)的 443（HTTPS）<a href="/doc/server/website_enable_ipv6" target="_blank">端口开放访问</a><br/>
       3. 网站所在服务器已开启 IPv6，但未开启SSL证书，请参考 <a href="/doc/server/nginx_ipv6" target="_blank">Nginx 开启 IPv6 SSL</a><br/>
    </blockquote>

  </div>

  
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";

.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}
</style>

<style>
:root {
  --el-color-primary: #3EAF7C;
}
.el-icon{
  font-size: 1.3em;
}
</style>
