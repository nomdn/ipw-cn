<script setup lang="ts">
import { getDocMeta } from '../../../config/doc'
import { config } from '../../../config/index'

// 窄屏下这条侧栏被 CSS 隐藏（导航在顶栏抽屉里），但照样要花钱水合 ——
// 详见 app/pages/doc/index.vue 的同名注释。
const isNarrow = useShellNarrow()

const route = useRoute();

const path = computed(() => route.path)
const { data: page } = await useFetch(() => `/api/markdown${path.value}`)

const meta = computed(() => getDocMeta(route.path))
// <title>/meta 优先取 seoXxx 加长版（Bing 审计判文档页标题与描述过短）；
// 侧栏菜单与卡片仍读原字段展示，不受影响。
const seoTitle = computed(() => meta.value.seoTitle ?? meta.value.title)
const seoDescription = computed(() => meta.value.seoDescription ?? meta.value.description)
const docUrl = computed(() => new URL(route.path, config.siteUrl).href)
useHead({
  title: seoTitle,
  titleTemplate: computed(() => `%s | ${config.siteName}`),
  meta: [
    { name: 'description', content: seoDescription },
    { property: 'og:title', content: computed(() => `${seoTitle.value} | ${config.siteName}`) },
    { property: 'og:description', content: seoDescription },
    { property: 'og:image', content: config.siteUrl + 'favicon.svg' },
    { property: 'og:type', content: 'article' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: computed(() => JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'BreadcrumbList',
        itemListElement: [
          { '@type': 'ListItem', position: 1, name: '首页', item: config.siteUrl },
          { '@type': 'ListItem', position: 2, name: '文档', item: new URL('/doc', config.siteUrl).href },
          { '@type': 'ListItem', position: 3, name: meta.value.title, item: docUrl.value }
        ]
      }))
    }
  ]
})
</script>
<template>
    <div class="box">
        <!-- 文档导航只有一份实现（app/components/DocMenu.vue）。
             宽屏：就是这条左侧固定侧栏；窄屏：本侧栏隐藏，改由顶部抽屉菜单承载 -->
        <DocMenu v-if="!isNarrow" class="sidebar-menu" />
        <div class="content">
            <div class="markdown-body" v-if="page">
                <div v-html="page" />
            </div>
        </div>
    </div>
</template>
<style scoped>
/* 核心布局容器 */
.box {
    display: flex;
    flex-direction: row; /* 改为 row，实现左右排列 */
    height: 100vh;       /* 占满整个屏幕高度，防止出现页面级滚动条 */
    width: 100%;
    background-color: #fff; /* 可选：设置整体背景色 */
}
html.dark .box {
    background-color: #242424; /* 可选：暗黑模式下的整体背景色 */
}
/* 左侧侧边栏样式 */
.sidebar-menu {
    width: 260px;        /* 固定侧边栏宽度 */
    height: 100%;        /* 高度撑满父容器 */
    flex-shrink: 0;      /* 防止被右侧内容挤压变形 */
    overflow-y: auto;    /* 菜单项过多时，侧边栏内部出现滚动条 */
    border-right: 1px solid #e4e7ed; /* 添加右侧分割线（可选） */
}

.content {
    flex: 1;             /* 右侧内容区域占据剩余空间 */
    padding: 20px;       /* 内边距，避免内容贴边 */
    overflow-y: auto;    /* 内容过多时，右侧区域出现滚动条 */
    background-color: #fff; /* 可选：设置右侧内容区域背景色 */
}
html.dark .content {
    background-color: #242424; /* 可选：暗黑模式下的背景色 */
}

@media (max-width: 768px) {
    /* 窄屏不再挤一条窄侧栏（原先 100px，长标题全被截断）：
       文档导航已并入左侧抽屉菜单，这里直接撤掉，正文铺满整屏 */
    .sidebar-menu {
        display: none;
    }
    .content {
        padding: 10px;
    }
}
</style>
<style>
@import "github-markdown-css/github-markdown-light.css";
@import "../../github-markdown-dark.css";
</style>
