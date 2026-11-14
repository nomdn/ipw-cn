<script setup lang="ts">
/**
 * 页面底部「你的 IP / 网络优先级」引用框，供新建的说明页、聚合页复用。
 *
 * 数据取自首页 index.vue 的探测结果：首页 head 里的内联脚本会把双栈接口的返回
 * 挂在 window.__ipProbe.dual 上（并派发 ip-probe 事件）。从首页软导航过来时直接
 * 复用这份结果，不再重复发请求；直接打开本页、内联脚本被拦或首页探测失败时，
 * 才回退到 config.DualStackAPI 自己取一次（与工具页引用框同一接口）。
 *
 * 优先级的判定方式与首页一致：双栈接口返回的地址族就是系统实际选择的协议栈。
 */
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { isIPv6 } from 'is-ip'
import { config } from '../../config/index'
import { isIPv4 } from '../../utils/tools'

const userIP = ref('')
let onProbe: ((e: Event) => void) | null = null

const removeProbe = () => {
  if (onProbe) {
    document.removeEventListener('ip-probe', onProbe)
    onProbe = null
  }
}

onMounted(() => {
  const store = (window as any).__ipProbe as Record<string, string> | undefined

  // 首页已探测完成：直接取用，不额外发请求
  if (typeof store?.dual === 'string' && store.dual) {
    userIP.value = store.dual
    return
  }

  // 谁先返回用谁；重复结果不覆盖，避免两次渲染抖动
  const apply = (ip: string) => {
    if (ip && !userIP.value) {
      userIP.value = ip
      removeProbe()
    }
  }

  // 首页探测还在跑：等它的 dual 事件（三个 key 共用一个事件名，不能 once）
  if (store) {
    onProbe = (e: Event) => {
      if ((e as CustomEvent).detail !== 'dual') return
      apply(store.dual)
    }
    document.addEventListener('ip-probe', onProbe)
  }

  // 直接打开本页 / 首页探测失败时的兜底
  $fetch<string>(config.DualStackAPI, { retry: false })
    .then(apply)
    .catch(() => {
      // 取不到就保持「获取中...」，不影响页面其它内容
    })
})

onBeforeUnmount(removeProbe)
</script>

<template>
  <blockquote class="ip-probe-quote">
    你的IP是：{{ userIP || '获取中...' }}，你的网络<span v-if="isIPv6(userIP)">IPv6优先</span><span v-else-if="isIPv4(userIP)">IPv4优先</span><span v-else>优先级查询中</span><br/>
  </blockquote>
</template>

<style scoped>
/* 与 app/style.css 的 blockquote 保持一致（组件有自己的 scoped 作用域，
   页面里 @import 的那份样式选择器带页面 scope，作用不到本组件内）。 */
.ip-probe-quote {
  background: #f9f9f9;
  border-left: 10px solid #ccc;
  margin: 1.5em 10px;
  padding: 0.5em 10px;
  letter-spacing: 0.5px;
  line-height: 2;
  word-spacing: 1px;
}

@media (max-width: 768px) {
  .ip-probe-quote {
    font-size: 0.9em;
  }
}
</style>
