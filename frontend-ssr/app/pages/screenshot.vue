<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { config } from '../../config/index'
import { Loading } from '@element-plus/icons-vue'
import { extractHost } from '../../utils/tools'
import { isIPv6 } from 'is-ip'

const route = useRoute()
const url = ref('')

useHead({
  title: '网站截图工具 - '+(route.query.site || config.siteName),
  meta: [
    { name: 'description', content: '在线网站截图工具，输入网址即可获取网页快照，支持所有公开网站的实时截图，方便查看网站布局和设计效果' },
    { name: 'keywords', content: '网站截图,网页快照,在线截图,网页预览,网站布局查看,截图工具' },
    { property: 'og:title', content: '网站截图工具 - 在线网页快照' },
    { property: 'og:description', content: '输入网址即可获取网页快照，查看网站布局和设计效果' },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'website' },
  ]
})

const tmpDomain = ref('https://www.zakoflare.com')
const loading = ref(false)
const error = ref('')
const screenshotUrl = ref('')
const imgLoaded = ref(false)


let detectionImg: HTMLImageElement | null = null
let detectionTimeout: ReturnType<typeof setTimeout> | null = null
let currentRequestId = 0
const userIP = ref('')
// WordPress mshots 默认占位图（生成失败/重定向目标）的尺寸是 1200x900，
// 与下方注释及判定逻辑一致；真实网站截图可以是任意尺寸
const DEFAULT_MSHOTS_WIDTH = 1200
const DEFAULT_MSHOTS_HEIGHT = 900
const LOADING_TIMEOUT = 15000 // 15 seconds

async function getUserIP(){
  try {
    userIP.value = await $fetch<string>(config.DualStackAPI)
  } catch {
    // 获取失败保持为空，避免未处理的 Promise 拒绝
  }
  return userIP.value
}
function cleanupDetection() {
  if (detectionImg) {
    detectionImg.onload = null
    detectionImg.onerror = null
    detectionImg = null
  }
  if (detectionTimeout) {
    clearTimeout(detectionTimeout)
    detectionTimeout = null
  }
}

function showError(message: string) {
  cleanupDetection()
  error.value = message
  loading.value = false
  screenshotUrl.value = ''
  imgLoaded.value = false
}

function takeScreenshot() {
  url.value = tmpDomain.value.trim()
  if (!url.value) {
    error.value = '请输入网址'
    return
  }

  if (!url.value.startsWith('http://') && !url.value.startsWith('https://')) {
    url.value = 'https://' + url.value
  }

  cleanupDetection()
  error.value = ''
  imgLoaded.value = false
  loading.value = true
  screenshotUrl.value = ''

  const requestId = ++currentRequestId

  // Set the screenshot URL on next tick so Vue can update the displayed <img>
  nextTick(() => {
    // Check if this request was superseded
    if (requestId !== currentRequestId) return

    screenshotUrl.value = `https://s0.wp.com/mshots/v1/${encodeURIComponent(url.value)}`

    // Create a hidden detection image to check for default WordPress error image
    // The mshots API returns a 302 redirect to a default image (1200x900) when
    // the target site is unreachable or generation times out.
    detectionImg = new Image()

    detectionImg.onload = () => {
      if (requestId !== currentRequestId) return

      // 307 Temporary Redirect to mshots default — target is still generating
      if (detectionImg!.currentSrc?.endsWith('/mshots/v1/default')) {
        cleanupDetection()
        loading.value = false
        error.value = '正在生成截图中，请稍后...'
        return
      }

      // WordPress mshots default error images are exactly 1200x900.
      // Real website screenshots can be any dimensions.
      if (
        detectionImg!.naturalWidth === DEFAULT_MSHOTS_WIDTH &&
        detectionImg!.naturalHeight === DEFAULT_MSHOTS_HEIGHT
      ) {
        showError('截图加载失败，可能是目标站点无法访问或生成超时')
        return
      }

      // Passed detection - screenshot is valid
      cleanupDetection()
      imgLoaded.value = true
      loading.value = false
    }

    detectionImg.onerror = () => {
      if (requestId !== currentRequestId) return
      showError('截图加载失败，请检查网址是否正确')
    }

    detectionImg.src = screenshotUrl.value

    // Timeout: if detection doesn't complete within LOADING_TIMEOUT ms
    detectionTimeout = setTimeout(() => {
      if (requestId !== currentRequestId) return
      showError('截图加载失败，可能是目标站点无法访问或生成超时')
    }, LOADING_TIMEOUT)
  })
}

// The displayed <img> may or may not fire its own load event depending on
// browser behavior with 302 redirects. The hidden detection image above
// handles the primary success/failure determination.
function onImageLoad() {
  if (detectionImg && detectionImg.currentSrc?.endsWith('/mshots/v1/default')) {
    error.value = '正在生成截图中，请稍后...'
    loading.value = false
  }
}

function onImageError() {
  showError('截图加载失败，请检查网址是否正确')
}

onMounted(() => {
  const urlParam = route.query.site as string
  if (urlParam) {
    tmpDomain.value = urlParam
    takeScreenshot()
  }
  getUserIP()
})

const { data: page } = await useAsyncData('screenshot-doc', () =>
  $fetch('/api/markdown/screenshot')
)
const doc = page.value
</script>

<template>
  <div class="title">
    <header>
      <h1>网站截图</h1>
      <p>输入网址，自动截图网站页面</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <el-input
        v-model="tmpDomain"
        placeholder="请输入网址（如：https://example.com）"
        @keyup.enter="takeScreenshot"
      />
      <el-button
        @click="takeScreenshot"
        type="primary"
        :loading="loading"
      >
        网站截图
      </el-button>
    </div>

    <div v-if="error" class="error-message">
      {{ error }}
    </div>

    <div v-if="screenshotUrl" class="result-section">
      <div class="screenshot-container">
        <div v-if="loading && !imgLoaded" class="loading-overlay">
          <el-icon class="is-loading" :size="40">
            <Loading />
          </el-icon>
          <p>正在截图中...</p>
        </div>
        <img
          :src="screenshotUrl"
          alt="网站截图"
          class="screenshot-img"
          :class="{ 'img-loaded': imgLoaded }"
          @load="onImageLoad"
          @error="onImageError"
        />
      </div>
    </div>

    <div v-if="doc" class="markdown">
      <div v-html="doc"></div>
    </div>
    <blockquote>

        访客IP: {{userIP }}，您的网络 {{ isIPv6(userIP) ? 'IPv6' : 'IPv4'}} 访问优先<br/>
    </blockquote>
  </div>
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";

.screenshot-container {
  position: relative;
  display: inline-block;
  width: 100%;
  max-width: 100%;
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  background: #f5f5f5;
  min-height: 200px;
}

.screenshot-img {
  display: block;
  max-width: 100%;
  height: auto;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.screenshot-img.img-loaded {
  opacity: 1;
}

html.dark .screenshot-container {
  background: #1a1919;
}

.loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
  z-index: 1;
}

html.dark .loading-overlay {
  background: #1a1919;
  color: #adbac7;
}

.loading-overlay p {
  margin-top: 12px;
  color: #606266;
  font-size: 1.1em;
}

html.dark .loading-overlay p {
  color: #c0c4cc;
}

.markdown :deep(a) {
  color: #3EAF7C !important;
  font-size: 1.3em;
  text-decoration: none
}
</style>
