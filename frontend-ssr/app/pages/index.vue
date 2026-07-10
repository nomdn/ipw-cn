<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import { isIPv6 } from 'is-ip';
import { config } from '../../config/index';
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue';
import { highlightCode } from '../../utils/shiki';
const route = useRoute();
const canonicalUrl = computed(() => new URL(route.path, config.siteUrl).toString());

useHead({
  title: '柠檬味ipw.cn | IP查询工具 | IPv4/IPv6地址查询与网络测试平台',
  titleTemplate: '%s',
  link: [
    { rel: 'canonical', href: canonicalUrl.value }
  ],
  meta: [
    { name: 'description', content: '柠檬味ipw.cn提供专业的IP查询服务,支持IPv4和IPv6地址在线查询、归属地定位、网络测速、DNS解析、SSL证书检测、TCPing测试等多种网络工具,致力于推进IPv6规模部署和应用,打造去中心化的IP查询平台' },
    { name: 'keywords', content: 'ipv6,ipv4,ip查询,ipv6查询,ipv4查询,ipv6地址查询,ipv4地址查询,网络测速,DNS查询,SSL检测,TCPing,IP归属地,IPv6优先' },
    { property: 'og:title', content: '柠檬味ipw.cn - 专业IP查询与网络测试工具平台' },
    { property: 'og:description', content: '提供IPv4/IPv6地址查询、网络测速、DNS解析、SSL检测等全方位网络诊断工具,助力IPv6普及与部署' },
    { property: 'og:image', content: `${config.siteUrl}favicon.svg` },
    { property: 'og:type', content: 'website' },
    { property: 'og:url', content: canonicalUrl.value },
    { name: 'twitter:card', content: 'summary_large_image' },
  ],
  script: [
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: '柠檬味ipw.cn',
        description: '柠檬味ipw.cn提供专业的IP查询服务,支持IPv4和IPv6地址在线查询、归属地定位、网络测速、DNS解析、SSL证书检测、TCPing测试等多种网络工具,致力于推进IPv6规模部署和应用,打造去中心化的IP查询平台',
        url: canonicalUrl.value,
        applicationCategory: 'DeveloperApplication',
        operatingSystem: 'Any',
        offers: {
          '@type': 'Offer',
          price: '0',
          priceCurrency: 'CNY'
        },
        featureList: 'IPv4地址查询,IPv6地址查询,IP归属地定位,网络测速,DNS解析,SSL证书检测,TCPing测试,IPv6优先检测',
        about: [
          {
            '@type': 'Thing',
            name: 'IP地址查询',
            description: '通过特定的IP地址获取相关的地理位置、运营商、网络类型等信息的技术服务。IPv4地址是32位地址格式,IPv6地址是128位地址格式,IPv6能够提供更大的地址空间,解决IPv4地址枯竭问题。'
          },
          {
            '@type': 'Thing',
            name: 'IPv6优先检测',
            description: '当访问一个同时支持IPv4和IPv6的双栈网站时,如果网络IPv6优先,系统会优先使用IPv6地址进行连接。通过访问双栈测试域名来判断网络优先级。'
          },
          {
            '@type': 'Thing',
            name: 'IPv6部署',
            description: 'IPv6是全球下一代互联网协议标准,相比IPv4具有更大的地址空间、更好的安全性、更高的网络效率。国家正在大力推进IPv6规模部署和应用,以适应未来互联网发展需求。'
          }
        ]
      })
    }
  ]
});

const code = `
# 请勿用于商业用途，仅供个人测试学习之用，请遵守中国法律法规
# 查询本机外网 IPv4 地址
curl 4.wsmdn.dpdns.org

# 查询本机外网 IPv6 地址
curl 6.wsmdn.dpdns.org

# 测试网络是 IPv4 还是 IPv6 访问优先
# (访问 IPv4/IPv6 双栈站点，如果返回 IPv6 地址，则 IPv6 访问优先)
curl test.wsmdn.dpdns.org
`.trim(); // 关键：去掉首尾多余的空行
const highlightedCode = ref('');

const ipAddress = ref('');
const yourIPv4 = ref('');
const yourIPv6 = ref('');

onMounted(async () => {
  try {
    highlightedCode.value = await highlightCode(code, 'bash');
  } catch {
    highlightedCode.value = '';
  }

  const [dualStack, ipV4, ipV6] = await Promise.allSettled([
    $fetch<string>(config.DualStackAPI),
    $fetch<string>(config.v4OnlyAPI),
    $fetch<string>(config.v6OnlyAPI)
  ]);

  if (dualStack.status === 'fulfilled') {
    ipAddress.value = dualStack.value;
  }
  if (ipV4.status === 'fulfilled') {
    yourIPv4.value = ipV4.value;
  }
  if (ipV6.status === 'fulfilled') {
    yourIPv6.value = ipV6.value;
  }
});

function isIPv4(ip: string): boolean {
  const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/;
  return ipRegex.test(ip);
}
</script>

<template>
  <div class="title">
    <header>
      <h1>IP查询</h1>
      <p>致力于IP查询去中心化,推进 IPv6 规模部署和应用</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line">
      <b>IPv4</b>&nbsp<p>{{ yourIPv4 }} </p>&nbsp<a :href="`/ipv6?ip=${yourIPv4}`" target="_blank">查询归属地</a>
    </div>
    <div class="one-line">
      <b>IPv6</b>&nbsp<p v-if="yourIPv6">{{ yourIPv6 }}</p><a :href="`/ipv6?ip=${yourIPv6}`" target="_blank" v-if="yourIPv6">&nbsp查询归属地</a><a v-else href="https://www.bing.com/search?q=%E5%AE%B6%E5%AE%BD%E5%BC%80ipv6" target="_blank">没有IPv6地址,查看如何开启IPv6</a>
    </div>
    <div style="font-size: 1.5em;">
      <h3 v-if="ipAddress && isIPv6(ipAddress)"><el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>您的网络IPv6优先</h3>
      <h3 v-else-if="ipAddress && isIPv4(ipAddress)"><el-icon><CircleCloseFilled style="color: red;"/></el-icon>您的网络IPv4优先</h3>
      <h3 v-else><el-icon><CircleCloseFilled /></el-icon>查询中，请稍后</h3>
    </div>
     <blockquote>
      手机默认开启 IPv6，宽带开启 IPv6 请自行搜索 (我们没有文档)
    </blockquote>

    <div v-if="highlightedCode" v-html="highlightedCode" class="code-block"></div>
    <div v-else class="code-block code-block--fallback">{{ code }}</div>
  </div>

</template>
<style scoped>
@import "../style.css";
.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

.code-block {
  margin-top: 1rem;
  padding: 1rem;
  border-radius: 0.75rem;
  overflow-x: auto;
  white-space: pre-wrap;
  word-break: break-word;
}

.code-block--fallback {
  background: rgb(48, 46, 46);
  border: 1px solid rgba(62, 175, 124, 0.18);
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace !important;
  color: rgb(255, 255, 255);
}

</style>
<style>
:root {
  --el-color-primary: #3EAF7C;
}
</style>