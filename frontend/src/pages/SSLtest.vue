<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { CircleCheckFilled, CircleCloseFilled,InfoFilled } from '@element-plus/icons-vue';

const route = useRoute()

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
// 等蛋饺给我双栈运行容器
const remoteAPI = ref('https://api-ipw.wsmdn.dpdns.org/')
const tmpDomain = ref('https://zakoflare.com')
const testDomain = ref('')
const loading = ref(false)
const error = ref('')
const result = ref<SSLCheckResponse | null>(null)

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

function formatTime(ms: number): string {
  if (ms < 1000) {
    return `${ms} ms`
  }
  return `${(ms / 1000).toFixed(2)} s`
}

function formatSpeed(speed: number): string {
  return `${speed.toFixed(2)} KB/s`
}

function getStatusCodeClass(code: number): string {
  if (code >= 200 && code < 300) return 'status-success'
  if (code >= 300 && code < 400) return 'status-warning'
  return 'status-error'
}

function checkSSL() {
  testDomain.value = tmpDomain.value
  loading.value = true
  error.value = ''
  result.value = null

  axios.get(remoteAPI.value + 'v1/ssl/' + testDomain.value)
    .then(function (response) {
      console.log(response.data)
      result.value = response.data
    })
    .catch(function (error) {
      console.log(error)
      error.value = '请求失败，请检查域名或网络'
    })
    .finally(function () {
      loading.value = false
    })
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
      <h1>SSL 证书检查</h1>
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
              <span :class="result.ipv4!.is_expired ? 'expired' : 'valid'">
                {{ result.ipv4!.is_expired ? '已过期' : '有效' }}
              </span>
            </td>
            <td class="table-value">
              <span v-if="result.ipv6" :class="result.ipv6.is_expired ? 'expired' : 'valid'">
                {{ result.ipv6.is_expired ? '已过期' : '有效' }}
              </span>
              <span v-else>-</span>
            </td>
          </tr>
          <tr>
            <td class="table-label">常用名称</td>
            <td class="table-value">{{ result.ipv4!.subject_common_name }}</td>
            <td class="table-value">{{ result.ipv6?.subject_common_name || '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">签发者</td>
            <td class="table-value">{{ result.ipv4!.issuer_organization?.join(', ')|| '-' }}</td>
            <td class="table-value">{{ result.ipv6?.issuer_organization?.join(', ') || '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">证书有效期 (天)</td>
            <td class="table-value">{{ result.ipv4!.cert_validity_days }} 天</td>
            <td class="table-value">{{ result.ipv6?.cert_validity_days || '-' }} 天</td>
          </tr>
          <tr>
            <td class="table-label">证书开始时间</td>
            <td class="table-value">{{ formatDate(result.ipv4!.cert_start_time) }}</td>
            <td class="table-value">{{ result.ipv6?.cert_start_time ? formatDate(result.ipv6.cert_start_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">证书结束时间</td>
            <td class="table-value">{{ formatDate(result.ipv4!.cert_end_time) }}</td>
            <td class="table-value">{{ result.ipv6?.cert_end_time ? formatDate(result.ipv6.cert_end_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">HTTP 版本</td>
            <td class="table-value">{{ result.ipv4!.http_version }}</td>
            <td class="table-value">{{ result.ipv6?.http_version || '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">主机记录</td>
            <td class="table-value">{{ result.ipv4!.host_record }}</td>
            <td class="table-value">{{ result.ipv6?.host_record || '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">HTTPS 访问返回码</td>
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
            <td class="table-label">总耗时</td>
            <td class="table-value">{{ formatTime(result.ipv4!.total_time) }}</td>
            <td class="table-value">{{ result.ipv6?.total_time ? formatTime(result.ipv6.total_time) : '-' }}</td>
          </tr>
          <tr>
            <td class="table-label">下载速度</td>
            <td class="table-value">{{ formatSpeed(result.ipv4!.download_speed) }}</td>
            <td class="table-value">{{ result.ipv6?.download_speed ? formatSpeed(result.ipv6.download_speed) : '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="result && result.ipv4 && !result.ipv4.is_expired &&  !result.ipv6.is_expired && result.ipv4.is_reachable && result.ipv6.is_reachable">
      <h3>结论：<el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>网站{{ testDomain }} 证书有效 </h3>
      <p><el-icon><InfoFilled style="color: lightgreen;"/></el-icon>请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。</p>
        <img src="/ssl-s1.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s2.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s2.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s3.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s3.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s4.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s4.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s5.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s5.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s6.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s6.svg"&gt;&lt;/a&gt;</code></pre>

    </div>
    <div v-if="result && result.ipv4 && !result.ipv4.is_expired &&  result.ipv6.is_expired && result.ipv4.is_reachable && !result.ipv6.is_reachable">
      <h3>结论：<el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>网站{{ testDomain }} 证书有效,但不支持IPv6访问 </h3>
      <p><el-icon><InfoFilled style="color: lightgreen;"/></el-icon>请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。</p>
        <img src="/ssl-s1.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s2.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s2.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s3.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s3.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s4.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s4.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s5.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s5.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ssl-s6.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ssl/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ssl-s6.svg"&gt;&lt;/a&gt;</code></pre>

    </div>
    <div v-else-if="result && result.ipv4 && result.ipv4.is_expired && result.ipv4.is_reachable">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ testDomain }} 证书无效 </h3>
      <h2>都没有证书了这网站还活啥</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div>
    <div v-else-if="result && result.ipv4 && result.ipv6 && !result.ipv4.is_reachable && !result.ipv6.is_reachable">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ testDomain }} 不可达 </h3>
      <h2>...</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div> 
    <blockquote>
      网站不支持 IPv6 SSL 可能原因：<br/>
      <br/>
      1. 网站所在服务器未开启 IPv6，请参考 网站开启 IPv6 的三种方式<br/>
      2. 网站所在服务器已开启 IPv6，但防火墙未对源地址是 IPv6 地址(::/0)的 443（HTTPS）端口开放访问<br/>
      3. 网站所在服务器已开启 IPv6，但未开启SSL证书，请参考 Nginx 开启 IPv6 SSL<br/>
    </blockquote>
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

.result-table thead tr {
  background-color: #c0c4cc;
}

.result-table .table-header {
  padding: 12px 15px;
  font-weight: 600;
  color: #303133;
  text-align: left;
  border: 1px solid #dcdfe6;
}

.result-table tbody tr {
  border-bottom: 1px solid #dcdfe6;
}

.result-table tbody tr:last-child {
  border-bottom: none;
}

.result-table tbody tr:hover {
  background-color: #f5f7fa;
}

.result-table .table-label {
  padding: 12px 15px;
  font-weight: 600;
  color: #606266;
  width: 150px;
  text-align: left;
  border: 1px solid #dcdfe6;
}

.result-table .table-value {
  padding: 12px 15px;
  color: #303133;
  border: 1px solid #dcdfe6;
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
.el-icon{
  font-size: 1.3em;
}
</style>
