<script setup lang="ts">
// IPv6 检测词族的聚合页：把「本机 / 网站 / 方法与文档」三支意图收在一个入口，
// 让 23 篇 IPv6 文档与 6 个工具页形成 hub-and-spoke，而不是各自孤立。
// 正文随 SSR 一起产出，可索引内容不依赖客户端二次取数。
const { data: pageDoc } = await useAsyncData('ipv6check-page-doc', () =>
  $fetch('/api/markdown/ipv6check')
)
const doc = pageDoc.value

import { ogImageMeta, siteUrl } from '../../utils/seo'
import { config } from '../../config/index'

const description = 'IPv6 检测与 IPv6 测试入口：一键查本机是否支持 IPv6、网站能否通过 IPv6 访问，按场景分流到 AAAA 记录、端口连通性、双栈证书与网站测速工具，附 IPv6 改造验收自查与常见问题。免费、无需注册，可反复检测。'

useHead({
  // 三段 `|` 结构、品牌末尾，与全站一致；本页主打「方法 / 工具大全」信息型词，
  // 第二段补「IPv6 测试」长尾，不与首页（本机）、/ipv6webcheck（网站）互抢工具型大词。
  title: `IPv6检测方法与工具大全 | IPv6测试、连通性与改造验收自查指南 | ${config.siteName}`,
  meta: [
    { name: 'description', content: description },
    { property: 'og:title', content: `IPv6检测方法与工具大全 | IPv6测试指南 | ${config.siteName}` },
    { property: 'og:description', content: description },
    ...ogImageMeta,
    { property: 'og:type', content: 'website' },
  ],
  script: [
    {
      key: 'ipv6check-jsonld',
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'BreadcrumbList',
        itemListElement: [
          { '@type': 'ListItem', position: 1, name: '首页', item: siteUrl() },
          { '@type': 'ListItem', position: 2, name: 'IPv6 检测方法与工具', item: siteUrl('/ipv6check') }
        ]
      })
    },
    {
      key: 'ipv6check-faq',
      type: 'application/ld+json',
      // FAQ 内容必须与 server/assets/markdown/ipv6check.md 的「常见问题」逐字一致：
      // Google 要求 FAQPage 的问答在页面上可见，只写进 JSON-LD 属于结构化数据违规。
      // 注：Google 已大幅收窄 FAQ 富媒体的展示范围，这里主要价值是让爬虫
      // 准确理解页面覆盖的问题，不承诺一定出富媒体。
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'FAQPage',
        mainEntity: [
          {
            '@type': 'Question',
            name: '怎么知道自己的网络支持 IPv6？',
            acceptedAnswer: {
              '@type': 'Answer',
              text: '打开首页即可看到本机的 IPv4 与 IPv6 出口地址，并通过双栈测试域名判定当前网络是 IPv6 优先还是 IPv4 优先。如果 IPv6 一行为空，通常是运营商未下发地址或光猫、路由器未开启 IPv6，与本机的网站访问无关。'
            }
          },
          {
            '@type': 'Question',
            name: '网站怎样才算真正支持 IPv6？',
            acceptedAnswer: {
              '@type': 'Answer',
              text: '需要同时满足三个条件：域名有 AAAA 记录、Web 服务在 IPv6 地址上监听了 80/443 端口、防火墙或云安全组对源地址 ::/0 放行了这些端口。只添加 AAAA 记录而服务未监听，是最常见的「假支持」。'
            }
          },
          {
            '@type': 'Question',
            name: '本机检测和网站检测有什么区别？',
            acceptedAnswer: {
              '@type': 'Answer',
              text: '本机检测回答「我这条宽带和操作系统能不能用 IPv6」，取决于运营商、路由器和系统前缀策略；网站检测回答「某个域名能不能被 IPv6 用户访问」，取决于站长的 DNS 记录、服务监听和防火墙。两者互相独立，一侧通过不代表另一侧通过。'
            }
          },
          {
            '@type': 'Question',
            name: '已经加了 AAAA 记录，为什么 IPv6 还是访问不了？',
            acceptedAnswer: {
              '@type': 'Answer',
              text: '优先检查服务端是否在 IPv6 地址上监听（如 Nginx 的 listen [::]:443）以及安全组是否放行 ::/0，再检查 IPv6 虚拟主机是否单独部署了证书。发布一个无法连通的 AAAA 记录会让客户端等待超时后才回退 IPv4，比不加更慢。'
            }
          },
          {
            '@type': 'Question',
            name: '不同平台的 IPv6 检测结果为什么不一致？',
            acceptedAnswer: {
              '@type': 'Answer',
              text: '各平台的探测节点分布在不同地域和运营商，CDN 调度、互联互通路径与安全组策略都会造成差异。个别节点失败多为中间链路问题，所有节点都失败才更像目标站点自身配置有误。建议用多节点工具看整体趋势而不是单点结论。'
            }
          },
          {
            '@type': 'Question',
            name: '备案或验收时提示「网站不支持 IPv6」怎么办？',
            acceptedAnswer: {
              '@type': 'Answer',
              text: '先按三项自查：用 DNS 查询确认 AAAA 记录、用网站 IPv6 检测确认双栈可访问、用 SSL 证书检测确认 IPv6 侧证书链与 SAN 正常。最常见的是只发布了 AAAA 记录却没有在 IPv6 上监听 443 端口，或 IPv6 虚拟主机漏配证书。官方与第三方平台通常有每日检测量上限，改造期间可用本站多节点工具反复自查，最终以对应平台的检测结果为准。'
            }
          }
        ]
      })
    }
  ]
})
</script>

<template>
  <div class="title">
    <header>
      <h1>IPv6 检测方法与工具</h1>
      <p>分清本机检测与网站检测，选对工具再排查</p>
    </header>
  </div>
  <div class="content">
    <div class="quick-links">
      <a href="/" class="quick-card">
        <b>本机 IPv6 检测</b>
        <span>查看本机 IPv4 / IPv6 出口地址与网络优先级，判断宽带是否支持 IPv6</span>
      </a>
      <a href="/ipv6webcheck" class="quick-card">
        <b>网站 IPv6 检测</b>
        <span>检查域名 AAAA 记录、双栈访问状态码与耗时，确认网站是否真正支持 IPv6</span>
      </a>
      <a href="/ipv6tcping" class="quick-card">
        <b>IPv6 连通性与测速</b>
        <span>用 TCPing 与网站测速定位端口不通、链路绕行、首字节慢等具体问题</span>
      </a>
    </div>

    <div v-if="doc" class="markdown">
      <div v-html="doc"></div>
    </div>
  </div>
</template>

<style scoped>
@import "../style.css";
@import "../../assets/css/tool-common.css";

.quick-links {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin: 16px 0 4px;
}

.quick-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border: 1px solid rgba(128, 128, 128, .28);
  border-radius: 8px;
  background: rgba(128, 128, 128, .06);
  text-decoration: none;
  color: inherit;
  line-height: 1.6;
}

.quick-card b {
  font-size: 1.05em;
  color: #299764;
}

.quick-card span {
  font-size: .88em;
  opacity: .8;
}

.quick-card:hover {
  border-color: #299764;
}

@media (max-width: 768px) {
  .quick-links {
    grid-template-columns: 1fr;
  }
}
</style>
