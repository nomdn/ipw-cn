<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { CircleCheckFilled, CircleCloseFilled,InfoFilled,Position } from '@element-plus/icons-vue';

const route = useRoute()

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
// 等蛋饺给我双栈运行容器
const remoteAPI = ref('https://api-ipw.wsmdn.dpdns.org/')
const tmpDomain = ref('https://zakoflare.com')
const testDomain = ref('')
const loading = ref(false)
const error = ref('')
const result = ref<PerformanceCheckResponse | null>(null)



function formatTime(ms: number): string {
  if (ms < 1000) {
    return `${ms} ms`
  }
  return `${(ms / 1000).toFixed(2)} s`
}

function formatSpeed(speed: number): string {
  return `${speed.toFixed(2)} KB/s`
}

function formatSize(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(2)} KB`
  }
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
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

  axios.get(remoteAPI.value + 'v1/detail/' + testDomain.value)
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
      <h3>结论：<el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>网站{{ testDomain }} 支持IPv6访问 </h3>
      <p><el-icon><InfoFilled style="color: lightgreen;"/></el-icon>请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。</p>
        <img src="/ipv6-s1.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s2.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s2.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s3.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s3.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s4.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s4.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s5.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s5.svg"&gt;&lt;/a&gt;</code></pre>
        <img src="/ipv6-s6.svg"/>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-s6.svg"&gt;&lt;/a&gt;</code></pre>
        <div class="one-line">
            <img src="/ipv6-certified-lite-s1.svg"/>
            <img src="/ipv6-certified-lite-s2.svg"/>
            <img src="/ipv6-certified-lite-s3.svg"/>
            <img src="/ipv6-certified-lite-s4.svg"/>
            <img src="/ipv6-certified-lite-s5.svg"/>
            <img src="/ipv6-certified-lite-s6.svg"/>
        </div>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-certified-lite-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <div class="one-line">
            <img src="/ipv6-certified-s1.svg"/>
            <img src="/ipv6-certified-s2.svg"/>
            <img src="/ipv6-certified-s3.svg"/>
            <img src="/ipv6-certified-s4.svg"/>
            <img src="/ipv6-certified-s5.svg"/>
            <img src="/ipv6-certified-s6.svg"/>
        </div>
        <pre><code>&lt;a href="https://ipw.wsmdn.dpdns.org/ipv6webcheck/?site=ipw.cn" title="本站支持 SSL 安全访问" target='_blank'&gt;&lt;img style='display:inline-block;vertical-align:middle' alt="本站支持 SSL 安全访问" src="https://ipw.wsmdn.dpdns.org/ipv6-certified-s1.svg"&gt;&lt;/a&gt;</code></pre>
        <p>提示：修改IPv6徽标文件名，可修改对应样式</p>
    </div>
    <div v-else-if="result && result.ipv4 && result.ipv6 && result.ipv4.is_reachable && !result.ipv6.is_reachable">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ testDomain }} 不支持IPv6访问 </h3>
      <h2>国家正在支持IPv6发展，我建议你赶紧想办法给IPv6适配</h2>
      <el-image src="/jingya.jpg"></el-image>
    </div>
    <div v-else-if="result && (!result.ipv6?.is_reachable && !result.ipv4?.is_reachable)">
      <h3>结论：<CircleCloseFilled style="width: 1.3em;color: red;"/>网站{{ testDomain }} 不可达 </h3>
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
html.dark {
  --el-color-primary: #3EAF7C;
}
.el-icon{
  font-size: 1.3em;
}
</style>
