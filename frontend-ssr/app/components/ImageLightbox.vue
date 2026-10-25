<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'

/**
 * 全站文档图片灯箱：全局点击委托 + Element Plus 自带查看器（el-image-viewer）。
 *
 * 在 app.vue 挂载一次即覆盖所有 v-html 文档容器（.markdown / .markdown-body），
 * 页面无需任何接线。命中规则：容器内的 <img>，且不在 <a> 链接里（放行导航）。
 * 查看器自带 Esc 关闭、滚轮缩放、旋转工具栏；hide-on-click-modal 允许点遮罩关闭。
 */
const visible = ref(false)
const urlList = ref<string[]>([])
const initialIndex = ref(0)

function onClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  const img = target.closest?.<HTMLImageElement>('.markdown img, .markdown-body img')
  if (!img) return
  if (img.closest('a')) return // 图片套在链接里 → 放行导航，不开灯箱
  if (img.complete && img.naturalWidth === 0) return // 加载失败的裂图不进灯箱
  const src = img.currentSrc || img.src
  if (!src) return
  e.preventDefault()

  // 收集同一容器内的全部图片：EP 查看器由此获得左右箭头/键盘多图切换
  // （doc 页一篇文章带几十张截图，单图模式等于白瞎了查看器的切换能力）
  const container = img.closest('.markdown, .markdown-body')
  const imgs = container ? Array.from(container.querySelectorAll('img')) : [img]
  const list = imgs
    .filter(i => !(i.complete && i.naturalWidth === 0))
    .map(i => i.currentSrc || i.src)
    .filter(Boolean)
  if (!list.length) return

  urlList.value = list
  initialIndex.value = Math.max(0, list.indexOf(src))
  visible.value = true
}

// 打开期间锁 body 滚动（EP 查看器自带 Esc 关闭与缩放/旋转工具栏）
watch(visible, (v) => {
  document.body.style.overflow = v ? 'hidden' : ''
})

// 路由切换时强制关闭：灯箱开着按浏览器后退，查看器不能跨页面残留
const route = useRoute()
watch(
  () => route.path,
  () => {
    visible.value = false
  }
)

onMounted(() => document.addEventListener('click', onClick))
onBeforeUnmount(() => {
  document.removeEventListener('click', onClick)
  document.body.style.overflow = ''
})
</script>

<template>
  <el-image-viewer
    v-if="visible"
    :url-list="urlList"
    :initial-index="initialIndex"
    :z-index="3000"
    hide-on-click-modal
    @close="visible = false"
  />
</template>

<style>
/* 文档图片可点的视觉暗示（非 scoped：v-html 内容拿不到 scope id） */
.markdown img,
.markdown-body img {
  cursor: zoom-in;
}
</style>
