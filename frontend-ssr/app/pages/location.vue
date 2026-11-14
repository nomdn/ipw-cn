<script setup lang="ts">
// 工具说明正文随 SSR 一起产出：可索引内容不依赖客户端二次取数。
const { data: pageDoc } = await useAsyncData('location-page-doc', () =>
  $fetch('/api/markdown/location')
)
const doc = pageDoc.value

import { ogImageMeta, siteUrl } from '../../utils/seo'
import { ref, onMounted, computed, nextTick, watch } from 'vue';
import { isIPv6 } from 'is-ip';
import {config} from '../../config/index';
import { useMiddlewareFetch } from '../../utils/middleware';
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue';
import { useRoute } from 'vue-router'
import { isIPv4 } from '../../utils/tools';
import { wgs84ToGcj02 } from '../../utils/gcj02';
const route = useRoute();
const loading = ref(false);

useHead({
  title: `IPv6地址查询 | IPv4查询、IP归属地定位与多数据源对比工具 | ${config.siteName}`,
  meta: [
    { name: 'description', content: 'IPv6 地址查询与 IPv4 地址查询工具：输入 IP 即可并列查看 BiliBili Live、GeoCN、IP2Region、Maxmind 等多个地理库与 ASN 库的归属地、运营商结果，支持中国大陆及境外地址，免费无需注册' },
    { property: 'og:title', content: `IPv6/IPv4地址归属地查询工具 - ${config.siteName}` },
    { property: 'og:description', content: '多数据源IP地址归属地查询,支持IPv4和IPv6,提供地理位置、运营商等详细信息' },
    ...ogImageMeta,
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      key: 'location-jsonld',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: 'IPv6/IPv4地址归属地查询工具',
        description: '专业的IPv6/IPv4地址归属地查询工具，支持BiliBili Live、GeoCN、IP2Region、Maxmind、纯真社区库等多数据源对比，精确定位IP地理位置、运营商信息。',
        url: siteUrl('location'),
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
});
interface IPLocationType {
  bilibili?: any;
  ip?: string;
  geocn?: any;
  lemondb?: any;
  lemondb_asn?: any;
  ip2region?: any;
  ip2location?: any;
  ip2location_asn?: any;
  maxmind_asn?: any;
  maxmind_city?: any;
  qqwry?: any;
  [key: string]: any;
}
const code = `
# 请勿用于商业用途，仅供个人测试学习之用，请遵守中国法律法规
# 查询本机外网 IPv4 地址
curl ${config.v4OnlyAPI}

# 查询本机外网 IPv6 地址
curl ${config.v6OnlyAPI}

# 测试网络是 IPv4 还是 IPv6 访问优先 
# (访问 IPv4/IPv6 双栈站点，如果返回 IPv6 地址，则 IPv6 访问优先)
curl ${config.DualStackAPI}
`.trim();

// 代码高亮放 SSR 期算（与首页同款做法）：HTML 里直接是带 <span> 的高亮结果，
// 客户端不必再下载/执行 shiki（引擎 + 语言 + 主题 ≈ 164KB），也不会出现
// "先渲染纯文本、稍后换成高亮"的块高变化。代码是构建期常量，结果可随 HTML 一起缓存。
// import 写在 handler 内部（动态 import 单独成 chunk），首屏不会预载它。
const { data: html } = await useAsyncData(
  'location-curl-code',
  async () => {
    const { highlightCode } = await import('../../utils/shiki');
    return await highlightCode(code, 'bash');
  },
  // 默认值给 code（不是空串）：这里模板只有单个 v-html，没有 fallback 分支，
  // 高亮失败时正好退回原始代码文本，别让代码块空着。
  // （首页那处模板有 v-else 分支，所以那边默认值必须留空，两处不可照抄。）
  { default: () => code },
);
const apiList = config.IPLocationAPI
const currentApiIndex = ref(0)
const backendID = computed(() => apiList[currentApiIndex.value]?.id || '')
const ipAddress = ref('');
const IPLocation = ref<IPLocationType>({});
const UserIP = ref('');
const error = ref('')

// —— 归属地地图：高德瓦片（GCJ-02）+ Leaflet，免 key ——
// 每个有坐标的来源各画一个点（bilibili / geocn 坐标已是 GCJ-02 直接用；
// maxmind / ip2location / dbip / lemondb 为 WGS84 → 转 GCJ-02 与底图对齐），
// 多点同时展示可直观看出各库定位的差异与离散度。
const mapContainer = ref<HTMLElement | null>(null)
let mapInstance: any = null
let markerGroup: any = null

const SOURCE_DEFS = [
  { key: 'lemondb', lat: 'latitude', lng: 'longitude', gcj: false, label: 'LemonDB', color: '#ffab00' },
  { key: 'bilibili', lat: 'latitude', lng: 'longitude', gcj: true, label: 'bilibili Live', color: '#299764' },
  { key: 'maxmind_city', lat: 'latitude', lng: 'longitude', gcj: false, label: 'MaxMind', color: '#1d4ed8' },
  { key: 'ip2location', lat: 'latitude', lng: 'longitude', gcj: false, label: 'IP2Location', color: '#f97316' },
  { key: 'dbip_city', lat: 'latitude', lng: 'longitude', gcj: false, label: 'DB-IP', color: '#8b5cf6' },
  { key: 'geocn', lat: 'latitude', lng: 'longitude', gcj: true, label: 'GeoCN', color: '#0ea5e9' },
]

const mapPoints = computed(() => {
  const r = IPLocation.value
  if (!r) return []
  const pts: Array<{ lat: number; lng: number; label: string; color: string }> = []
  for (const s of SOURCE_DEFS) {
    const lat = parseFloat(r[s.key]?.[s.lat])
    const lng = parseFloat(r[s.key]?.[s.lng])
    if (Number.isFinite(lat) && Number.isFinite(lng) && !(lat === 0 && lng === 0)) {
      // LemonDB 全库统一 WGS-84（coord_system 恒为 wgs84，这里留个动态判断作前向兼容，
      // 若哪天源库换坐标系标注会自动跟随）；其余源维持静态标记
      const gcj = s.gcj || r[s.key]?.coord_system === 'gcj02'
      const p = gcj ? { lat, lng } : wgs84ToGcj02(lat, lng)
      pts.push({ ...p, label: s.label, color: s.color })
    }
  }
  return pts
})

let leafletModule: any = null
async function renderLocationMap() {
  if (!import.meta.client || !mapPoints.value.length || !mapContainer.value) return
  const pts = mapPoints.value
  leafletModule = leafletModule || (await import('leaflet'))
  const L = leafletModule.default ?? leafletModule
  if (!mapInstance) {
    mapInstance = L.map(mapContainer.value, { attributionControl: true })
    L.tileLayer('https://webrd0{s}.is.autonavi.com/appmaptile?lang=zh_cn&size=1&scale=1&style=8&x={x}&y={y}&z={z}', {
      subdomains: ['1', '2', '3', '4'],
      maxZoom: 18,
      // 高德栅格瓦片原生层级上限约 17：部分区域（尤其县乡）z=18 会返回纯米色
      // "无数据"空瓦片。声明 maxNativeZoom 后，Leaflet 对更高缩放自动放大
      // 17 级瓦片（轻度模糊但绝不变空白）。
      maxNativeZoom: 17,
      attribution: '&copy; <a href="https://www.amap.com/" target="_blank" rel="noreferrer">高德地图</a>',
    }).addTo(mapInstance)
  }
  if (markerGroup) markerGroup.remove()
  markerGroup = L.featureGroup()
  for (const p of pts) {
    L.circleMarker([p.lat, p.lng], {
      radius: 8,
      color: p.color,
      fillColor: p.color,
      fillOpacity: 0.85,
      weight: 2,
    })
      .bindPopup(`<b>${p.label}</b><br>${ipAddress.value || UserIP.value}<br>${p.lat.toFixed(5)}, ${p.lng.toFixed(5)}`)
      .addTo(markerGroup)
  }
  markerGroup.addTo(mapInstance)
  // 多点：视野自适应包住全部点；单点：定位到 10 级
  if (pts.length > 1) {
    mapInstance.fitBounds(markerGroup.getBounds().pad(0.3))
  } else {
    mapInstance.setView([pts[0].lat, pts[0].lng], 10)
  }
}

watch(IPLocation, async () => {
  if (mapPoints.value.length) {
    await nextTick()
    await renderLocationMap()
  }
})
const locationUrl = computed(() => "/middleware/" + backendID.value + "/location/" + encodeURIComponent(ipAddress.value));

const { data: locationData, error: locationError, execute: executeLocation } = useMiddlewareFetch<IPLocationType>(locationUrl, {
  immediate: false,
  watch: false,
});

watch(locationData, (newData) => {
  if (newData) {
    IPLocation.value = newData;
    loading.value = false;
    error.value = '';
  }
});

watch(locationError, async (newError) => {
  if (newError) {
    console.error('Error fetching IP location:', newError);
    if (currentApiIndex.value < apiList.length - 1) {
      const nextIndex = currentApiIndex.value + 1
      error.value = `${(newError as any).message || '请求失败'}，正在重试 ${apiList[nextIndex]?.label || ''}...`
      currentApiIndex.value = nextIndex
      await nextTick()
      executeLocation()
    } else {
      error.value = '请求失败，请检查网络或稍后重试'
      loading.value = false;
    }
  }
});

function locateIP(IP: string){
  currentApiIndex.value = 0
  ipAddress.value = IP;
  error.value = ''
  loading.value = true;
  nextTick(() => {
    executeLocation();
  });
}

onMounted(async () => {
  // （代码高亮已移到 setup 顶部的 useAsyncData，随 SSR 一起产出，这里只剩定位查询。）

  const urlParam = route.query.ip;
  if (urlParam) {
    ipAddress.value = urlParam as string;
    locateIP(urlParam as string);
  }else{
    $fetch<string>(config.DualStackAPI).then(
      function(data) {
        ipAddress.value = data;

        UserIP.value = data;
      }
    ).catch(
      function(error) {
        console.error('Error fetching IP address:', error);
      }
    );

  }
});
</script>
<template>
  <div class="title">
    <header>
      <h1>IPv6/IPv4地址归属地查询</h1>
      <p>极简的IPv6地址查询工具，致力于普及 IPv6</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input 
        v-model="ipAddress" 
        placeholder="请输入IP地址"
        clearable
      />
      <el-button 
        @click="locateIP(ipAddress)" 
        type="primary" 
        :loading="loading"
      
      >
        查询
      </el-button>
    </div>
    <div v-if="error" class="error-message">
      {{ error }}
    </div>
    <div class="location">
      <div class="ip-info" style="height: 40px;">
        <b>IP</b>&nbsp;<span>{{ ipAddress }}</span>
      </div>
      <div v-if="IPLocation" class="result-section">
          <table class="result-table">
            <tbody>
              <tr v-if="IPLocation.lemondb && (IPLocation.lemondb.country || IPLocation.lemondb.administrative_area || IPLocation.lemondb.city)">
                <td class="table-label">LemonDB</td>
                <td class="table-value">
                  {{ IPLocation.lemondb?.country }}&nbsp;{{ IPLocation.lemondb?.administrative_area }}&nbsp;{{ IPLocation.lemondb?.city }}&nbsp;{{ IPLocation.lemondb?.district }}
                </td>
                <td class="table-value">
                  <span>
                    {{ IPLocation.lemondb?.isp }}&nbsp;{{ IPLocation.lemondb?.type }}
                  </span>
                </td>
              </tr>
              <tr v-if="IPLocation.bilibili && (IPLocation.bilibili.administrative_area || IPLocation.bilibili.city)">
                <td class="table-label">bilibili Live接口</td>
                <td class="table-value">
                  <span>
                    {{ IPLocation.bilibili?.country }}&nbsp;{{ IPLocation.bilibili?.administrative_area }}&nbsp;{{ IPLocation.bilibili?.city }}
                  </span>
                </td>
                <td class="table-value">
                  <span>
                    {{ IPLocation.bilibili?.isp }}
                  </span>
                </td>
              </tr>
              <tr v-if="IPLocation.ip2region && typeof IPLocation.ip2region === 'object' && (IPLocation.ip2region.administrative_area || IPLocation.ip2region.city)">
                <td class="table-label">IP2Region</td>
                <td class="table-value">{{ IPLocation.ip2region?.country }}&nbsp;{{ IPLocation.ip2region?.administrative_area }}&nbsp;{{ IPLocation.ip2region?.city }}</td>
                <td class="table-value">{{ IPLocation.ip2region?.isp }}</td>
              </tr>
              <tr v-if="IPLocation.ip2location && typeof IPLocation.ip2location === 'object' && (IPLocation.ip2location.administrative_area || IPLocation.ip2location.city)">
                <td class="table-label">IP2Location</td>
                <td class="table-value">{{ IPLocation.ip2location?.country }}&nbsp;{{ IPLocation.ip2location?.administrative_area }}&nbsp;{{ IPLocation.ip2location?.city }}</td>
                <td class="table-value">{{ IPLocation.ip2location_asn?.as }}</td>
              </tr>
              <tr v-if="IPLocation.geocn && (IPLocation.geocn.administrative_area || IPLocation.geocn.city || IPLocation.geocn.district)">
                <td class="table-label">GeoCN(仅中国大陆)</td>
                <td class="table-value">{{ IPLocation.geocn?.administrative_area }}&nbsp;{{ IPLocation.geocn?.city }}&nbsp;{{ IPLocation.geocn?.district }}</td>
                <td class="table-value">{{ IPLocation.geocn?.isp }}&nbsp;{{IPLocation.geocn?.type}}</td>
              </tr>
              <tr v-if="IPLocation.maxmind_city && IPLocation.maxmind_asn && (IPLocation.maxmind_city.country || IPLocation.maxmind_city.city)">
                <td class="table-label">Maxmind GEOLite2 City</td>
                <td class="table-value">{{ IPLocation.maxmind_city?.country }}&nbsp;{{ IPLocation.maxmind_city?.administrative_area }}&nbsp;{{ IPLocation.maxmind_city?.city }}</td>
                <td class="table-value">{{ IPLocation.maxmind_asn?.org }}</td>
              </tr>
              <tr v-if="IPLocation.qqwry && (IPLocation.qqwry.country || IPLocation.qqwry.administrative_area || IPLocation.qqwry.city)">
                <td class="table-label">纯真社区库</td>
                <td class="table-value">{{ IPLocation.qqwry?.country }}&nbsp;{{ IPLocation.qqwry?.administrative_area }}&nbsp;{{ IPLocation.qqwry?.city }}</td>
                <td class="table-value">{{ IPLocation.qqwry?.isp }}</td>
              </tr>
              <tr v-if="IPLocation.dbip_city && (IPLocation.dbip_city.administrative_area || IPLocation.dbip_city.city)">
                <td class="table-label">DB-IP City</td>
                <td class="table-value">{{ IPLocation.dbip_city?.country }}&nbsp;{{ IPLocation.dbip_city?.administrative_area }}&nbsp;{{ IPLocation.dbip_city?.city }}</td>
                <td class="table-value">{{IPLocation.dbip_asn?.org}}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <!-- 归属地地图：高德瓦片（GCJ-02）+ Leaflet，每个有坐标的来源各画一点 -->
        <div v-if="mapPoints.length" ref="mapContainer" class="location-map"></div>
    </div>
    <div style="font-size: 1.5em;">
      <h3 v-if="isIPv6(UserIP)"><el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>您的网络IPv6优先</h3>
      <h3 v-else-if="isIPv4(UserIP)"><el-icon><CircleCloseFilled style="color: red;"/></el-icon>您的网络IPv4优先</h3>
    </div>
    <blockquote>
      精度参照表：<br>
      中国大陆:BiliBili Live > GeoCN > IP2Region > 纯真社区库 > Maxmind GEOLite2 City ≈ DB-IP<br>
      中国大陆 IPv6 地址: BiliBili Live > GeoCN > IP2Region > Maxmind GEOLite2 City ≈ DB-IP > 纯真社区库<br>
      境外及港澳台地址: Maxmind GEOLite2 City ≈ DB-IP > BiliBili Live >  > IP2Region > 纯真社区库<br>
      
      手机默认开启 IPv6，宽带开启 IPv6 请自行搜索<br>
      访客IP: {{UserIP}}，<span v-if="isIPv4(UserIP)">您的网络IPv4优先</span><span v-else-if="isIPv6(UserIP)">您的网络IPv6优先</span>
    </blockquote>

    <div v-html="html"></div>

    <div v-if="doc" class="markdown">
      <div v-html="doc"></div>
    </div>
    </div>



</template>
<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";
/* leaflet 基础样式必须非 scoped 引入（动态创建的窗格拿不到 data-v 属性） */
</style>
<style>
@import "leaflet/dist/leaflet.css";
.location-map {
  height: 320px;
  margin-top: 16px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  z-index: 0; /* 压住 leaflet 内部窗格的层级，避免盖过下拉/弹窗 */
}
</style>
<style>
@import "../style.css";
:root {
  --el-color-primary: #299764;
}
.el-icon{
  font-size: 1.3em;
}
.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
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
html.dark .result-table {
  width: 100%;
  border-collapse: collapse;
  background: #2e2d2d;
  border: 1px solid #2e2e2e;
}
.result-table thead tr {
  background-color: #c0c4cc;
}
html.dark .result-table thead tr {
    background: #2e2d2d;
}

.result-table .table-header {
  padding: 12px 15px;
  font-weight: 600;
  color: #303133;
  text-align: left;
  border: 1px solid #dcdfe6;
}
html.dark .result-table .table-header {
  padding: 12px 15px;
  font-weight: 600;
  color: #cfcfcf;
  text-align: left;
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
  width: 150px;
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
}
html.dark .result-table .table-value {
  color: #cfcfcf;
  border: 1px solid #1a1919;
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
html.dark pre {
  background: #303133;
}

html.dark pre code {
  font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  color: #f8f9fa;
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
  color: #299764;
  font-size: 1.2em;
}

.badge-item img {
  display: block;
  margin-bottom: 15px;
  max-width: 200px;
}
</style>
