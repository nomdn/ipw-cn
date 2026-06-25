<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import axios from 'axios'
import { config } from '../config/index';

const route = useRoute()





const tmpDomain = ref('')
const domain = ref('')
const recordType = ref('a')
const loading = ref(false)
const results = ref<any>([])
const isloading = ref(false)
function formatTime(ms: number): string {
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

async function queryDNS(){ 
    isloading.value = true
    domain.value = tmpDomain.value
    let PromiseArray = []
    for (let i = 0; i < config.NSLookup.length; i++){
        PromiseArray.push(axios.get(
            config.NSLookup[i].url +'v1/dns/' + recordType.value + "/"+domain.value)
            .then(function (response) { 
                return {
                    server: config.NSLookup[i].label,
                    data: response.data
                }
            }).catch(
                function (err) {
                    return {
                        server: config.NSLookup[i].label,
                        error: err
                    }
                }
            )
        )
    }
    const PeomiseResults = await Promise.all(PromiseArray)
    console.log(PeomiseResults)
    results.value = PeomiseResults
    isloading.value = false
    
    return PeomiseResults
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
    results.value = queryDNS().then(res => {
      results.value = res
    })
  }
})
</script>

<template>
  <div class="title">
    <header>
      <h1>DNS 查询</h1>
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
            <th class="table-header">耗时 (S)</th>
            <th class="table-header">TTL (S)</th>
            
          </tr>
        </thead>
        <tbody>
          <tr v-for="(result) in results" :key="result.server">
            <td class="table-label">{{result.server}}</td>
            <td class="table-value">{{recordType}}</td>
            
            <td class="table-value" style="text-align: center;">
              <template v-if="result && result.data?.record">
                <div v-for="(ip, index) in result.data.record.slice(0, 5)" :key="index" class="ip-address">
                  {{ ip }}
                </div>
              </template>

              <span v-else class="status-code" style="color: #F56C6C; background: #fef0f0;">
                失败 {{ result.error }}
              </span>
            </td>
            
            <td class="table-value">{{result.data?.record?.length || 0}}</td>
            <td class="table-value">{{formatTime(result.data?.duration)}}</td>
            <td class="table-value">{{result.data?.ttl}}</td>
          </tr>
        </tbody>
        
        </table>
    </div>
    </div>

</template>

<style scoped>
@import "../style.css";

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
  padding-bottom: 0 !important;
  /* 保持原有的边框样式 */
  box-shadow: 0 0 0 1px var(--el-border-color) inset !important;
  border-radius: var(--el-border-radius-base) !important;
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
