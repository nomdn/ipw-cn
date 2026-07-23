<script setup lang="ts">
import { ref, onMounted, computed, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index';
import { CircleCheckFilled, CircleCloseFilled,InfoFilled,Position } from '@element-plus/icons-vue';
import { extractHost, getStatusCodeClass, formatTime, formatSpeed, formatSize } from '~/utils/tools';

const route = useRoute()

useHead({
  title: 'IPv6网站检测工具 | IPv6访问支持检查 | 柠檬味ipw.cn',
  meta: [
    { name: 'description', content: '专业的IPv6网站检测工具,全面检查网站是否支持IPv6访问,提供IPv4和IPv6双栈HTTP/HTTPS状态码、DNS解析时间、TCP连接时间、下载速度等详细对比数据,帮助网站管理员确认IPv6部署状态,提供IPv6徽标认证,推进IPv6规模部署和应用' },
    { name: 'keywords', content: 'ipv6网站检测,ipv6访问检测,ipv6支持检查,ipv6双栈检测,ipv6 http检测,ipv6 https检测,ipv6网站认证,ipv6徽标,ipv6部署' },
    { property: 'og:title', content: 'IPv6网站检测 - IPv6访问支持状态检查与认证' },
    { property: 'og:description', content: '全面检测网站IPv6访问支持情况,提供详细性能对比与IPv6徽标认证' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv6网站访问支持检测',
        description: '专业的IPv6网站检测工具，检查网站是否支持IPv6访问，提供IPv4和IPv6双栈HTTP/HTTPS状态码、DNS解析时间、TCP连接时间等详细对比数据，提供IPv6徽标认证。',
        url: config.siteUrl + 'ipv6webcheck',
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

interface PerformanceCheckResponse {
  ipv4?: PerformanceCheckItem
  ipv6?: PerformanceCheckItem
}

interface PerformanceCheckItem {
  host_record: string
  http_status_code: number
  https_status_code: number
  dns_lookup_time: number
  tcp_connect_time: number
  http_connect_time: number
  first_byte_time: number
  total_time: number
  page_size: number
  download_speed: number
  is_reachable: boolean
}
const remoteAPI = ref(config.apiBaseUrl)
const tmpDomain = ref('https://www.zakoflare.com')
const testDomain = ref('')
const loading = ref(false)
const error = ref('')
const result = ref<PerformanceCheckResponse | null>(null)
const checkUrl = computed(() => remoteAPI.value + 'v1/detail/' + testDomain.value);

const { data: checkData, error: checkError, execute: executeCheck } = useFetch<PerformanceCheckResponse>(checkUrl, {
  immediate: false,
  watch: false,
});

watch(checkData, (newData) => {
  if (newData) {
    result.value = newData;
    loading.value = false;
  }
});

watch(checkError, (newError) => {
  if (newError) {
    console.log(newError);
    error.value = '请求失败，请检查域名或网络';
    loading.value = false;
  }
});

function checkSSL() {
  testDomain.value = tmpDomain.value
  loading.value = true
  error.value = ''
  result.value = null
  nextTick(() => {
    executeCheck();
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
      <h1>IPv6网站检测</h1>
      <p>检查网站是否开启 IPv6 访问，致力于普及IPv6</p>
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
        开始测试
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
            <td class="table-label">主机记录</td>
            <td class="table-value"><div class="one-line"><RouterLink :to="`/ipv6?ip=${result.ipv4!.host_record}`">{{ result.ipv4!.host_record }}</RouterLink><el-icon><Position /></el-icon></div></td>
            <td class="table-value"><div class="one-line"><RouterLink :to="`/ipv6?ip=${result.ipv6?.host_record}`">{{ result.ipv6?.host_record || '-' }}</RouterLink><el-icon><Position /></el-icon></div></td>
          </tr>
          <tr>
            <td class="table-label">HTTP 状态码</td>
            <td class="table-value">
              <span :class="getStatusCodeClass(result.ipv4!.http_status_code)" class="status-code">
                {{ result.ipv4!.http_status_code }}
              </span>
            </td>
            <td class="table-value">
              <span v-if="result.ipv6" :class="getStatusCodeClass(result.ipv6.http_status_code)" class="status-code">
                {{ result.ipv6.http_status_code }}
              </span>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">HTTPS 状态码</td>
            <td class="table-value">
              <span :class="getStatusCodeClass(result.ipv4!.https_status_code)" class="status-code">
                {{ result.ipv4!.https_status_code }}
              </span>
            </td>
            <td class="table-value">
              <span v-if="result.ipv6" :class="getStatusCodeClass(result.ipv6.https_status_code)" class="status-code">
                {{ result.ipv6.https_status_code }}
              </span>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">DNS 查询耗时</td>
            <td class="table-value">{{ formatTime(result.ipv4!.dns_lookup_time) }}</td>
            <td class="table-value">{{ result.ipv6?.dns_lookup_time ? formatTime(result.ipv6.dns_lookup_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">TCP 连接耗时</td>
            <td class="table-value">{{ formatTime(result.ipv4!.tcp_connect_time) }}</td>
            <td class="table-value">{{ result.ipv6?.tcp_connect_time ? formatTime(result.ipv6.tcp_connect_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">HTTP 连接耗时</td>
            <td class="table-value">{{ formatTime(result.ipv4!.http_connect_time) }}</td>
            <td class="table-value">{{ result.ipv6?.http_connect_time ? formatTime(result.ipv6.http_connect_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">首字节耗时</td>
            <td class="table-value">{{ formatTime(result.ipv4!.first_byte_time) }}</td>
            <td class="table-value">{{ result.ipv6?.first_byte_time ? formatTime(result.ipv6.first_byte_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">总耗时</td>
            <td class="table-value">{{ formatTime(result.ipv4!.total_time) }}</td>
            <td class="table-value">{{ result.ipv6?.total_time ? formatTime(result.ipv6.total_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">页面大小</td>
            <td class="table-value">{{ formatSize(result.ipv4!.page_size) }}</td>
            <td class="table-value">{{ result.ipv6?.page_size ? formatSize(result.ipv6.page_size) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">下载速度</td>
            <td class="table-value">{{ formatSpeed(result.ipv4!.download_speed) }}</td>
            <td class="table-value">{{ result.ipv6?.download_speed ? formatSpeed(result.ipv6.download_speed) : '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="result && result.ipv4  && result.ipv6&& result.ipv4.is_reachable && result.ipv6.is_reachable">
      <h3>结论：<el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>网站{{ extractHost(testDomain) }} 支持IPv6访问 </h3>
      <p><el-icon><InfoFilled style="color: lightgreen;"/></el-icon>请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。</p>
        <img src="/ipv6-s1.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s2.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s2.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s3.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s3.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s4.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s4.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s5.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s5.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s6.svg"/>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s6.svg"&gt;&lt;/a&gt;</code></pre>
        <div class="one-line">
            <img src="/ipv6-certified-lite-s1.svg"/>
            <img src="/ipv6-certified-lite-s2.svg"/>
            <img src="/ipv6-certified-lite-s3.svg"/>
            <img src="/ipv6-certified-lite-s4.svg"/>
            <img src="/ipv6-certified-lite-s5.svg"/>
            <img src="/ipv6-certified-lite-s6.svg"/>
        </div>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-certified-lite-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <div class="one-line">
            <img src="/ipv6-certified-s1.svg"/>
            <img src="/ipv6-certified-s2.svg"/>
            <img src="/ipv6-certified-s3.svg"/>
            <img src="/ipv6-certified-s4.svg"/>
            <img src="/ipv6-certified-s5.svg"/>
            <img src="/ipv6-certified-s6.svg"/>
        </div>
        <pre><code>&lt;a href="{{ config.siteUrl }}ipv6webcheck/?site={{ extractHost(testDomain) }}" title="本站支持 IPv6访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="https://ipw.wsmdn.dpdns.org/ipv6-certified-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <p>提示：修改IPv6徽标文件名，可修改对应样式</p>
    </div>
    <div v-else-if="result && result.ipv4 && result.ipv6 && result.ipv4.is_reachable && !result.ipv6.is_reachable">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ extractHost(testDomain) }} 不支持IPv6访问 </h3>
      <h2>国家正在支持IPv6发展，我建议你赶紧想办法给IPv6适配</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div>
    <div v-else-if="result && (!result.ipv6?.is_reachable && !result.ipv4?.is_reachable)">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ extractHost(testDomain) }} 不可达 </h3>
      <h2>...</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div>
    <blockquote>
      网站不支持 IPv6可能原因：<br/>
      <br/>
      1. 网站所在服务器未开启 IPv6，请参考 网站开启 IPv6 的三种方式<br/>
      2. 网站所在服务器已开启 IPv6，但防火墙未对源地址是 IPv6 地址(::/0)的 443（HTTPS）端口开放访问<br/>
      3. 网站所在服务器已开启 IPv6，但未开启SSL证书，请参考 Nginx 开启 IPv6 SSL<br/>
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
html.dark {
  --el-color-primary: #3EAF7C;
}
.el-icon{
  font-size: 1.3em;
}
</style>
