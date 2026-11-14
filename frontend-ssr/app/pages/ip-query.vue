<script setup lang="ts">
// IP 查询词族的独立说明页：原挂在首页末尾的正文迁到这里，首页 body 恢复原版。
// 正文随 SSR 一起产出，可索引内容不依赖客户端二次取数。
const { data: pageDoc } = await useAsyncData('ip-query-page-doc', () =>
  $fetch('/api/markdown/ip-query')
)
const doc = pageDoc.value

import { ogImageMeta, siteUrl } from '../../utils/seo'
import { config } from '../../config/index'

const description = 'IP 查询说明：一次看懂本机公网 IPv4 地址、IPv6 地址与网络优先级，解释 IPv4 查询与 IPv6 查询的区别、为什么出口 IP 不等于本机地址。附 IP 归属地、DNS 等工具入口，免费无需注册。'

useHead({
  // 三段 `|` 结构、品牌末尾，与全站一致；首段用报告里卡在 7~9 位的精确词「IP查询」。
  title: `IP查询说明 | IPv4与IPv6地址查询、出口IP与网络优先级解读 | ${config.siteName}`,
  meta: [
    { name: 'description', content: description },
    { property: 'og:title', content: `IP查询说明 | IPv4与IPv6地址查询 | ${config.siteName}` },
    { property: 'og:description', content: description },
    ...ogImageMeta,
    { property: 'og:type', content: 'article' },
  ],
  script: [
    {
      key: 'ip-query-jsonld',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'BreadcrumbList',
        itemListElement: [
          { '@type': 'ListItem', position: 1, name: '首页', item: siteUrl() },
          { '@type': 'ListItem', position: 2, name: 'IP 查询说明', item: siteUrl('/ip-query') }
        ]
      })
    }
  ]
})
</script>

<template>
  <div class="title">
    <header>
      <h1>IP 查询说明：IPv4 与 IPv6 地址查询</h1>
      <p>读懂出口 IP、网络优先级与 IPv4 / IPv6 查询结果</p>
    </header>
  </div>
  <div class="content">
    <div v-if="doc" class="markdown">
      <div v-html="doc"></div>
    </div>
    <IpProbeQuote />
  </div>
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";
</style>
