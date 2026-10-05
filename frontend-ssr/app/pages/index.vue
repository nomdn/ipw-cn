<script setup lang="ts">
import { computed, ref, onMounted } from 'vue';
import { isIPv6 } from 'is-ip';
import { config } from '../../config/index';
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue';
import { isIPv4 } from '../../utils/tools';
const route = useRoute();
const canonicalUrl = computed(() => new URL(route.path, config.siteUrl).toString());

useHead({
  title: `${config.siteName} | IP查询工具 | IPv4/IPv6地址查询与网络测试平台`,
  titleTemplate: '%s',
  link: [
    { rel: 'canonical', href: canonicalUrl.value }
  ],
  meta: [
    { name: 'description', content: `${config.siteName}提供专业的IP查询服务,支持IPv4和IPv6地址在线查询、归属地定位、网络测速、DNS解析、SSL证书检测、TCPing测试等多种网络工具,致力于推进IPv6规模部署和应用,打造去中心化的IP查询平台` },
    { name: 'keywords', content: 'ipv6,ipv4,ip查询,ipv6查询,ipv4查询,ipv6地址查询,ipv4地址查询,网络测速,DNS查询,SSL检测,TCPing,IP归属地,IPv6优先' },
    { property: 'og:title', content: `${config.siteName} - 专业IP查询与网络测试工具平台` },
    { property: 'og:description', content: '提供IPv4/IPv6地址查询、网络测速、DNS解析、SSL检测等全方位网络诊断工具,助力IPv6普及与部署' },
    { property: 'og:image', content: `${config.siteUrl}favicon.svg` },
    { property: 'og:type', content: 'website' },
    { property: 'og:url', content: canonicalUrl.value },
    { name: 'twitter:card', content: 'summary_large_image' },
  ],
  script: [
    {
      // —— IP 探测提前到 HTML 解析期（LCP 优化）——
      // 原来三个探测请求在 onMounted（水合完成后）才发出，模拟链路是
      // 「水合 → 发请求 → 等服务端 → 回填渲染」，IPv6 行作为 LCP 元素被整段串行拖住
      // （mobile 模拟 LCP 2275ms，满分线约 1600ms）。
      // 这里在 head 里用内联脚本立刻发起，与 render-blocking CSS 的下载、
      // hydration 三路并行；结果挂 window.__ipProbe，完成时派发 ip-probe 事件，
      // onMounted 只负责消费（见下方），不再等水合才起步。
      // 用原生 fetch：此刻 ofetch 还没加载；不带 retry 的语义与旧版 retry:false 一致
      // （底层 fetch 本来就不会重试）。
      innerHTML: [
        'window.__ipProbe = {};',
        '(function () {',
        `  function emit(key) {`,
        `    return function (text) {`,
        `      window.__ipProbe[key] = text;`,
        `      document.dispatchEvent(new CustomEvent('ip-probe', { detail: key }));`,
        `    };`,
        `  }`,
        // 与 onMounted 的语义一致：三接口各自独立、失败静默（保持「查询中」占位）
        `  fetch(${JSON.stringify(config.v6OnlyAPI)}).then(function (r) { return r.text(); }).then(emit('v6')).catch(function () {});`,
        `  fetch(${JSON.stringify(config.v4OnlyAPI)}).then(function (r) { return r.text(); }).then(emit('v4')).catch(function () {});`,
        `  fetch(${JSON.stringify(config.DualStackAPI)}).then(function (r) { return r.text(); }).then(emit('dual')).catch(function () {});`,
        '})();',
      ].join('\n'),
    },
    {
      type: 'application/ld+json',
      innerHTML: JSON.stringify({
        '@context': 'https://schema.org',
        '@type': 'WebApplication',
        name: config.siteName,
        description: `${config.siteName}提供专业的IP查询服务,支持IPv4和IPv6地址在线查询、归属地定位、网络测速、DNS解析、SSL证书检测、TCPing测试等多种网络工具,致力于推进IPv6规模部署和应用,打造去中心化的IP查询平台`,
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
curl ${config.v4OnlyAPI}

# 查询本机外网 IPv6 地址
curl ${config.v6OnlyAPI}

# 测试网络是 IPv4 还是 IPv6 访问优先
# (访问 IPv4/IPv6 双栈站点，如果返回 IPv6 地址，则 IPv6 访问优先)
curl ${config.DualStackAPI}
`.trim(); // 关键：去掉首尾多余的空行

// 代码高亮放在 SSR 期算：HTML 下发时就已经是带 <span> 的高亮结果，
// 于是 ①客户端不必再下载/执行 shiki（引擎 + 语言 + 主题 ≈ 164KB）；
// ②不会再有"先出纯文本兜底、若干秒后才换成高亮"的那次块高变化（可见重排）。
//
// 这段代码是构建期常量（只依赖 config，不随请求变化），所以结果可以安全地随 HTML 一起缓存。
//
// import 仍然写在 handler 内部（而不是文件顶层）：动态 import 会把它切成独立 chunk，
// 只在真正要高亮时加载，不会像顶层静态 import 那样被塞进首屏的 modulepreload 列表。
// 首次进入是 SSR 执行的；客户端水合直接读 payload（Nuxt 会序列化 useAsyncData 的结果），
// 只有客户端软导航到本页时才在浏览器里跑一次。
const { data: highlightedCode } = await useAsyncData(
  'home-curl-code',
  async () => {
    const { highlightCode } = await import('../../utils/shiki');
    return await highlightCode(code, 'bash');
  },
  // 默认值必须留空：模板是 v-if="highlightedCode" / v-else 的双分支，
  // 给成 code 会让 v-if 命中、把纯文本当 HTML 塞进 v-html（fallback 分支永远走不到）。
  // 留空则失败时自然落到 v-else 的 code-block--fallback，与改动前 catch 里清空的行为一致。
  { default: () => '' },
);

const ipAddress = ref('');
const yourIPv4 = ref('');
const yourIPv6 = ref('');

onMounted(() => {
  // （代码高亮已移到 setup 顶部的 useAsyncData，随 SSR 一起产出。）

  // —— 三个 IP 探测：请求已由 head 内联脚本提前发出（见 useHead），这里只消费 ——
  // 内联脚本把结果挂 window.__ipProbe[key] 并派发 ip-probe 事件。三种时序都要覆盖：
  //   ① 值已就位（事件在 onMounted 之前派发过）→ 同步读 store 直接上屏；
  //   ② 值未到 → 挂监听等事件（收到匹配 key 才移除，不能 once——
  //      三个 key 共用同一事件名，once 会把别人的事件当自己的吃掉）；
  //   ③ window.__ipProbe 不存在（内联脚本被 CSP 拦掉/异常）→ 退回自己发请求，行为同旧版。
  //
  // retry: false —— ofetch 默认会对 GET 重发一次：网络层失败时它拿不到 response，就按 500 计
  // （500 在默认重试白名单里），且默认延迟为 0，立即重发。于是"这个节点连不上"会被原样重做一遍，
  // 用户白等一倍时间（实测 ofetch 1.5.1：裸调用触发 2 次真实 fetch，传 retry:false 后只剩 1 次；
  // 浏览器自身不会重试，底层 fetch 对同一请求只被调用过一次）。仅 ③ 的兜底路径会走到 $fetch。
  const ipFetchOptions = { retry: false } as const;
  const probeStore = (window as any).__ipProbe as Record<string, string> | undefined;

  const consumeProbe = (key: string, apply: (ip: string) => void): boolean => {
    if (!probeStore) return false; // ③ 内联脚本没跑起来
    if (typeof probeStore[key] === 'string') {
      apply(probeStore[key]); // ① 值已就位
      return true;
    }
    const onProbe = (e: Event) => {
      if ((e as CustomEvent).detail !== key) return;
      document.removeEventListener('ip-probe', onProbe);
      apply(probeStore[key]); // ② 事件到达
    };
    document.addEventListener('ip-probe', onProbe);
    return true;
  };

  // 三个地址各自独立请求、各自渲染：谁先回来谁先上屏。
  // 不用 Promise.allSettled 包起来等齐——那样只要有一个慢（例如纯 IPv4 网络下 v6 接口要等超时），
  // 已经拿到的 IPv4 与双栈结果也得一起干等。
  if (!consumeProbe('dual', (ip) => { ipAddress.value = ip; })) {
    $fetch<string>(config.DualStackAPI, ipFetchOptions)
      .then((ip) => {
        ipAddress.value = ip;
      })
      .catch(() => {
        // 单个接口失败不影响其它两行展示，保持"查询中"占位
      });
  }
  if (!consumeProbe('v4', (ip) => { yourIPv4.value = ip; })) {
    $fetch<string>(config.v4OnlyAPI, ipFetchOptions)
      .then((ip) => {
        yourIPv4.value = ip;
      })
      .catch(() => {});
  }
  if (!consumeProbe('v6', (ip) => { yourIPv6.value = ip; })) {
    $fetch<string>(config.v6OnlyAPI, ipFetchOptions)
      .then((ip) => {
        yourIPv6.value = ip;
      })
      .catch(() => {});
  }
});
</script>


<template>
  <div class="title">
    <header>
      <h1>IP查询</h1>
      <p>致力于IP查询去中心化,推进 IPv6 规模部署和应用</p>
    </header>
  </div>
  <div class="content">
    <div class="one-line ip-row">
      <b>IPv4</b>&nbsp<p>{{ yourIPv4 }} </p>&nbsp<RouterLink :to="`/location?ip=${yourIPv4}`" target="_blank">查询归属地</RouterLink>
    </div>
    <div class="one-line ip-row ip-row-v6">
      <b>IPv6</b>&nbsp<p v-if="yourIPv6">{{ yourIPv6 }}</p><RouterLink :to="`/location?ip=${yourIPv6}`" target="_blank" v-if="yourIPv6">&nbsp查询归属地</RouterLink><RouterLink v-else to="/doc/user/enable_ipv6" target="_blank">没有IPv6地址,查看如何开启IPv6</RouterLink>
    </div>
    <div class="ip-priority">
      <h2 v-if="ipAddress && isIPv6(ipAddress)"><el-icon><CircleCheckFilled style="color: lightgreen;"/></el-icon>您的网络IPv6优先</h2>
      <h2 v-else-if="ipAddress && isIPv4(ipAddress)"><el-icon><CircleCloseFilled style="color: red;"/></el-icon>您的网络IPv4优先</h2>
      <h2 v-else><el-icon><CircleCloseFilled /></el-icon>查询中，请稍后</h2>
    </div>
     <blockquote>
      手机默认开启 IPv6，宽带开启 IPv6 请参阅<a href="/doc/user/enable_ipv6" target="_blank">文档</a>
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

/* 网络优先级提示：原来是「div 1.5em × h3」，但 h1 之后直跳 h3 违反标题层级（heading-order），
   语义上应为 h2。style.css 给 h3 显式定了 1.3em / 窄屏 1.1em，h2 只有 1.5em 一档，
   所以这里把字号按原样补回，避免语义修正顺带改了视觉（含 margin：UA 的 h3 是 1em，h2 是 0.83em）。 */
.ip-priority {
  font-size: 1.5em;
}
.ip-priority h2 {
  font-size: 1.3em;
  margin: 1em 0;
}

/* 两行 IP 记录预留确定高度，消除 CLS。
   实测（CDP 逐帧采样 + CSS 变体对照，填值后量终态几何）：
   这两个 div 的文本是 onMounted 之后才由 $fetch 填上的。IPv6 地址（39 字符）
   在窄屏下把 .one-line 从 1 行撑到 3 行，下面 .ip-priority / 代码块 / blockquote /
   footer 被整体顶下 50px —— Lighthouse 记的 `div.code-block` 那条 CLS 0.0359
   （占全页 88%）就是这么来的，而满分门槛只有 0.0396，差 3% 就被扣分。
   修法是让高度与「填完之后」一致。终态高度按视口分档（CDP 实测，不是估算）：
     移动 412 宽 → ip2 终态 75px（3 行）；桌面 1350 宽 → 30px（1 行，.content 宽 60%）
   所以必须分媒体查询给值：一刀切 75px 会让桌面凭空多出 45px 空白。
   代价是接口 pending 时这里空着，但换来 CLS 归零。 */
.ip-row {
  min-height: 30px;
}

.code-block {
  margin-top: 1rem;
  padding: 1rem;
  border-radius: 0.75rem;
  overflow-x: auto;
  max-width: 100%;
  white-space: pre-wrap;
  overflow-wrap: break-word;
}

.code-block--fallback {
  background: rgb(48, 46, 46);
  border: 1px solid rgba(62, 175, 124, 0.18);
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace !important;
  color: rgb(255, 255, 255);
}

@media (max-width: 768px) {
    .ip-priority h2 {
      font-size: 1.1em;
    }
    .code-block {
      padding: 0.75rem;
      font-size: 0.8em;
    }
    /* 窄屏下 IPv6 地址换 3 行（CDP 实测 412 宽终态 h=75px），见上面 .ip-row 的注释。
       75px 只给 IPv6 行：IPv4 地址窄屏仍是 1 行（~30px），一刀切 75px 会让
       IPv4 行下部空出 ~45px，两块 IP 之间出现一大段空白。
       IPv4 回填前后都是 1 行 30px，高度不变 → 不引入新的 CLS。 */
    .ip-row {
      /* style.css 窄屏给 .one-line 加了 0.5em 上下 margin（两块之间叠 16px），
         IP 行的间距由行高本身决定，这里归零。scoped 属性选择器特异性更高，稳赢。 */
      margin: 0;
    }
    .ip-row-v6 {
      min-height: 75px;
      align-content: flex-start;
    }
}
</style>
<style>
:root {
  --el-color-primary: #299764;
}
</style>