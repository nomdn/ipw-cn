<script setup lang="ts">
import { computed, nextTick, onMounted, onBeforeUnmount, ref } from 'vue';
import { useDark, useToggle } from '@vueuse/core';
import { Moon, Sunny, Expand } from '@element-plus/icons-vue';
import { config } from '../config/index';

const isNarrow = ref(false);
let mediaQueryList: MediaQueryList | null = null;
const drawer = ref(false);

const isDark = useDark();
const toggleDark = useToggle(isDark);

// ==================== 抽屉里的文档菜单 ====================
// 窄屏下 /doc 页面自己的左侧侧栏会被隐藏（见 app/pages/doc/*.vue），
// 文档导航改由这里的抽屉承载；宽屏不渲染抽屉，一切照旧。
const route = useRoute()
const isDocRoute = computed(() => route.path === '/doc' || route.path.startsWith('/doc/'))

function cleanChineseCharacters(str:string) {
return str.replace(/[\u4e00-\u9fa5]/g, '');
}

// ==================== 响应式折叠菜单 ====================
// 映射表：菜单按显示顺序排在一个数组里，每项自带 minWidth（px）折叠阈值。
// 规则：视口宽度 >= minWidth 显示，低于则折叠进「更多」子菜单。
// 断点从右往左递减（最右侧 index 9 先折叠），数值可按需调整。
// divider 也带 minWidth（取其后第一项的值）：所在段全部折叠时，分隔线一并隐藏。
interface NavItem {
  kind: 'item'
  index: string
  label: string
  to: string
  minWidth: number
}
interface NavGroup {
  kind: 'group'
  index: string
  label: string
  minWidth: number
  children: { index: string; label: string; to: string }[]
}
interface NavDivider {
  kind: 'divider'
  minWidth: number
}
type MenuEntry = NavItem | NavGroup | NavDivider

const menu: MenuEntry[] = [
  // 断点设计基准（从右往左折叠）：全显示约 1480px，1366 屏可用约 1350px，
  // 故 1366 屏折叠 index 7/8/9（显示 1-6 +「更多」，约 1220px 宽裕）；1440 折叠 8/9；1500+ 全显示。
  { kind: 'item', index: '1', label: 'IPv6 网站检测', to: '/ipv6webcheck', minWidth: 980 },
  { kind: 'item', index: '2', label: 'IPv6/IPv4 地址查询', to: '/location', minWidth: 1040 },
  { kind: 'item', index: '3', label: 'IPv6 TCPing测试', to: '/ipv6tcping', minWidth: 1100 },
  { kind: 'divider', minWidth: 1160 }, // 与 index 4 相同：4-6 段全折叠时隐藏
  { kind: 'item', index: '4', label: 'IPv6 DNS解析', to: '/dns', minWidth: 1160 },
  { kind: 'item', index: '5', label: 'IPv6 SSL检查', to: '/ssl', minWidth: 1220 },
  { kind: 'item', index: '6', label: 'IPv6 网站测速', to: '/ipv6speedtest', minWidth: 1280 },
  { kind: 'divider', minWidth: 1380 }, // 与 index 7 相同：7-9 段全折叠时隐藏
  {
    kind: 'group', index: '7', label: 'IPv4工具箱', minWidth: 1380,
    children: [
      { index: '7-0', label: 'IPv4 网站测速', to: '/speedtest' },
      { index: '7-1', label: 'IPv4 TCPing测试', to: '/tcping' },
    ],
  },
  {
    kind: 'group', index: '8', label: '其他工具', minWidth: 1420,
    children: [
      { index: '8-0', label: '网站截图', to: '/screenshot' },
      { index: '8-1', label: 'Whois查询', to: '/whois' },
      { index: '8-3', label: 'ASN查询', to: '/asn' },
      { index: '8-4', label: 'DNSSEC验证', to: '/dnssec' },
    ],
  },
  { kind: 'item', index: '9', label: '文档', to: '/doc', minWidth: 1500 },
]

// SSR 阶段无 window，默认按宽屏输出全部菜单项；onMounted 后更新为真实视口宽度
const viewportWidth = ref(1920)

// 可见项（含分隔线）：视口宽度达标即显示
const visible = computed<MenuEntry[]>(() =>
  menu.filter((entry) => viewportWidth.value >= entry.minWidth),
)
// 折叠项（不含分隔线）：视口宽度不足即收进「更多」。
// 用类型谓词收窄为 NavItem | NavGroup，模板里才能安全访问 entry.index
const hidden = computed<Array<NavItem | NavGroup>>(
  () => menu.filter((entry): entry is NavItem | NavGroup =>
    entry.kind !== 'divider' && viewportWidth.value < entry.minWidth),
)

// EP 的 Menu.init() 会在挂载时（以及菜单项集合变化时）遍历顶栏 <ul> 的**每个直接子元素**、
// 统一写 tabindex="0" —— 连纯装饰的分隔条也不放过。调用栈实测：
//   Menu.init → Array.forEach → new MenuItem(el).init → setAttribute('tabindex','0')
// 分隔条是 aria-hidden 的装饰元素，留着 tabindex="0" 会被 axe 判为 aria-hidden-focus
// （"Focusable content should have tabindex=-1 or be removed from the DOM"）——
// 这一条比它替换掉的 aria-required-attr 更靠后，所以必须显式改回去。
// 改的时机：子组件 mounted 早于父组件，onMounted 里 EP 已经 init 完；
// 但要等 DOM 更新落定 —— updateViewport() 会改 visible、让 el-menu 重渲染并再 init 一遍。
function fixDividerTabindex() {
  document.querySelectorAll('.menu-divider').forEach((el) => {
    el.setAttribute('tabindex', '-1')
  })
}

// 实时同步视口宽度（resize 触发，驱动折叠）
function updateViewport() {
  viewportWidth.value = window.innerWidth
  nextTick(fixDividerTabindex)
}
let umamiScript: HTMLScriptElement | null = null
useHead({
  meta: config.noindex
    ? [
        { name: 'robots', content: 'noindex, nofollow' },
        { name: 'googlebot', content: 'noindex, nofollow' },
        { name: 'bingbot', content: 'noindex, nofollow' },
        { name:'baidu-site-verification', content: 'codeva-xzdbvF8gQu'}
      ]
    : [{ name:'baidu-site-verification', content: 'codeva-xzdbvF8gQu'}],
});
onMounted(() => {
  mediaQueryList = window.matchMedia('(max-width: 768px)');
  isNarrow.value = mediaQueryList.matches;

  const handler = (e: MediaQueryListEvent) => {
    isNarrow.value = e.matches;
  };

  mediaQueryList.addEventListener('change', handler);

  updateViewport();
  window.addEventListener('resize', updateViewport);

  onBeforeUnmount(() => {
    mediaQueryList?.removeEventListener('change', handler);
    window.removeEventListener('resize', updateViewport);
  });

  if (!umamiScript && config.umamiScriptUrl) {
    umamiScript = document.createElement('script')
    umamiScript.src = config.umamiScriptUrl
    umamiScript.async = true
    umamiScript.setAttribute('data-website-id', config.umamiWebsiteId)
    document.head.appendChild(umamiScript)
  }
})
</script>

<template>
  
  <el-drawer v-if="isNarrow" v-model="drawer" direction="ltr" style="height: 100%;" :size="isDocRoute ? '80%' : '60%'">
      <router-link to="/ipv6webcheck">
        <p class="menu-item-text">IPv6 网站检测</p>
      </router-link>
      <router-link to="/location">
        <p class="menu-item-text">IPv6/IPv4 地址查询</p>
      </router-link>
      <router-link to="/ipv6tcping">
        <p class="menu-item-text">IPv6 TCPing</p>
      </router-link>
      <router-link to="/dns"><p class="menu-item-text">IPv6 DNS解析</p></router-link>
      <router-link to="/ssl">
        <p class="menu-item-text">IPv6 SSL检查</p>
      </router-link>
      <router-link to="/ipv6speedtest"><p class="menu-item-text">IPv6 网站测速</p></router-link>
      <router-link to="/speedtest"><p class="menu-item-text">IPv4 网站测速</p></router-link>
      <router-link to="/tcping"><p class="menu-item-text">IPv4 TCPing</p></router-link>
      <router-link to="/screenshot"><p class="menu-item-text">网站截图</p></router-link>
      <router-link to="/whois"><p class="menu-item-text">Whois查询</p></router-link>
      <router-link to="/asn"><p class="menu-item-text">ASN查询</p></router-link>
      <router-link to="/dnssec"><p class="menu-item-text">DNSSEC验证</p></router-link>
      <!-- 窄屏唯一的文档入口：非文档页面抽屉里不挂整份文档菜单（太长），
           只留这一个链接，对应宽屏顶栏 index 9 的「文档」项。
           进入文档页后，抽屉里才会出现完整的文档导航。 -->
      <router-link v-if="!isDocRoute" to="/doc" @click="drawer = false"><p class="menu-item-text">文档</p></router-link>
      <!-- 文档导航并进同一个左侧菜单：与 /doc 页面用的是同一个组件。
           只在文档路由下出现（非文档页面抽屉里就只留工具链接）；
           文档路由下分组默认展开（手机上没 hover，省一次点击）。
           :key 是为了让分组开合重新初始化 —— el-menu 只在挂载时读一次 default-openeds，
           抽屉一旦打开过就不会重挂，不加 key 会停在首次打开时的开合状态。 -->
      <DocMenu
        v-if="isDocRoute"
        :key="'doc-expanded'"
        expand-all
        @select="drawer = false"
      />
  </el-drawer>
  <el-menu
      mode="horizontal"
      :ellipsis="false"
      
    >
    <!-- 宽屏不给 aria-label：这个 <li> 里有可见文本（右侧的 h2 站名），而 axe 的
         label-content-name-mismatch 要求 aria-label 必须把可见文本包含进去 ——
         写死「返回首页」只会被判不匹配（Lighthouse 桌面独有项，权重 0 不扣分，但语义是错的）。
         窄屏 h2 被 v-if 隐藏、img 的 alt 留空、el-icon 也没有文本 ⇒ 那时才需要
         aria-label 兜底，否则承载 role="menuitem" 的 <li> 就没有可访问名（aria-command-name）。
         里面那个 <router-link> 同理（它自己也是一个可访问名有冲突的目标），
         宽屏靠 h2 的「柠檬味ipw.cn」当名字，窄屏才补「返回首页」。 -->
    <el-menu-item index="0" :aria-label="isNarrow ? '返回首页' : undefined">
      <el-icon v-if="isNarrow" @click="drawer = !drawer"><Expand /></el-icon>
      <router-link to="/" :aria-label="isNarrow ? '返回首页' : undefined">
        <!-- alt 留空是有意的：logo 属装饰图，可访问名由外层链接提供。
             若把 alt 写成站名，宽屏下会与旁边的 h2 一起被读两遍。
             这里用原生 <img> 而不是 el-image：el-image 默认 lazy，SSR 只输出一个
             `.el-image__placeholder` 空壳，真正的 <img> 要等客户端水合后由
             IntersectionObserver 插进去 ⇒ logo 在首屏 HTML 里不存在（无 JS 时永不显示，
             有 JS 时也要闪一下）。原生 <img> 直接进 HTML，还顺带省掉 el-image
             及其 image-viewer 的样式。宽高属性声明固有比例，显示宽度由下文
             :deep(.el-menu-item a img) 定为 50px。 -->
        <img src="/favicon.svg" alt="" width="50" height="50" style="margin-top: 20px;" />
        <h2 style="display: inline-block; margin-left: 10px" v-if="!isNarrow">{{ config.siteName }}</h2>
      </router-link>
    </el-menu-item>
    
    <template v-if="!isNarrow">
    <!-- 可见菜单项：按映射表顺序渲染 item / group / divider -->
    <template v-for="entry in visible" :key="entry.kind === 'divider' ? 'div-' + entry.minWidth : entry.index">
      <el-menu-item v-if="entry.kind === 'item'" :index="entry.index">
        <router-link :to="entry.to"><p class="menu-item-text">{{ entry.label }}</p></router-link>
      </el-menu-item>
      <el-sub-menu v-else-if="entry.kind === 'group'" :index="entry.index">
        <template #title>{{ entry.label }}</template>
        <el-menu-item v-for="child in entry.children" :key="child.index" :index="child.index">
          <router-link :to="child.to"><p class="menu-item-text">{{ child.label }}</p></router-link>
        </el-menu-item>
      </el-sub-menu>
      <!-- 分组之间的竖分隔条。原来是 <el-divider direction="vertical">，但它在客户端水合后
           会被 EP 补上 tabindex="0"（SSR HTML 里干净、没有这个属性），于是 axe 把
           role="separator" 当成「可聚焦控件」并要求它带 aria-valuenow —— 这就是桌面端
           aria-required-attr 那一条（Lighthouse 权重 10，是桌面 a11y 掉 12 分里的最大单项）。
           换成 aria-hidden 的原生 <span>：装饰元素不参与焦点，也不会被 EP 改写属性。
           视觉与 .el-divider--vertical 等价，数值见下方 .menu-divider 的注释。 -->
      <span v-else class="menu-divider" aria-hidden="true"></span>
    </template>
    <!-- 折叠项：统一收进「更多」子菜单（分组用 el-menu-item-group 保留组名） -->
    <el-sub-menu v-if="hidden.length" index="more">
      <template #title>更多</template>
      <template v-for="entry in hidden" :key="entry.index">
        <el-menu-item v-if="entry.kind === 'item'" :index="'m-' + entry.index">
          <router-link :to="entry.to"><p class="menu-item-text">{{ entry.label }}</p></router-link>
        </el-menu-item>
        <el-menu-item-group v-else :title="entry.label">
          <el-menu-item v-for="child in entry.children" :key="child.index" :index="'m-' + child.index">
            <router-link :to="child.to"><p class="menu-item-text">{{ child.label }}</p></router-link>
          </el-menu-item>
        </el-menu-item-group>
      </template>
    </el-sub-menu>
    </template>
    <!-- 纯图标菜单项：无可访问名，补 aria-label（图标本身不带文字） -->
    <el-menu-item index="10" aria-label="切换深色/浅色模式">
      <ClientOnly>
      <el-icon @click="toggleDark()" v-if="isDark" style="cursor: pointer;"><Moon style="height: 20px; width: 20px;"/></el-icon>
      <el-icon @click="toggleDark()" v-else style="cursor: pointer;"><Sunny style="height: 20px; width: 20px;"/></el-icon>
      </ClientOnly>
    </el-menu-item>


  </el-menu>
  
  <NuxtLoadingIndicator />
  <main id="main-content" role="main">
    <NuxtPage />
  </main>

  <!-- 文档路由不渲染页脚：文档页是阅读页，正文底部加备案/版权块会打断阅读，
       且窄屏下与侧栏抽屉的可滚动区域互相挤压。 -->
  <footer v-if="!isDocRoute">
    <div class="one-line">
      <!-- 这两张 svg 补上 width/height 属性（= 各自的固有尺寸 190×36 / 85×36）：
           浏览器要在图下载完之前就知道占位比例，否则会先把整行按 0 高度排一遍、
           图到了再重排（Lighthouse 的 unsized-images 就是这条）。
           属性只声明固有比例，不改变显示尺寸：桌面端本来就是按固有尺寸渲染，
           窄屏由 style.css 的 `footer .one-line img { height:1.2em; width:auto }` 接管。 -->
      Copyright © nomdn & IP 查询 2026  | <img src="/ipv6-s1.svg" alt="IPv6 相关标识" width="190" height="36"/> | <img src="/ssl-s1.svg" alt="SSL 相关标识" width="85" height="36"/> | All right reserved
    </div>
    <div class="one-line">
      <a v-if="config.ICP" href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer" >{{ config.ICP }}</a>
      <span v-if="config.ICP">&nbsp;|&nbsp;</span>
      <!-- 同上：原生 <img> 而非 el-image（1em 见方的小图标，用不着 lazy/预览器）。
           显示尺寸由行内 style 定死；width/height 属性只声明固有比例（36×40），防抖动。 -->
      <img v-if="config.GongAn" src="/备案图标.png" alt="" width="36" height="40" style="height: 1em; width: 1em;" />
      <a v-if="config.GongAn" :href="'https://beian.mps.gov.cn/#/query/webSearch?code=' + cleanChineseCharacters(config.GongAn)" target="_blank" rel="noreferrer" >{{ config.GongAn }}</a>
      <span v-if="config.GongAn">&nbsp;|&nbsp;</span>
      <a href="https://www.china-ipv6.cn/" target="_blank" rel="noreferrer" >国家IPv6发展监测平台</a>
      &nbsp;|&nbsp;请遵守中国法律法规&nbsp;|&nbsp;
      <a href="https://github.com/nomdn/ipw-cn" target="_blank" rel="noreferrer" >Github</a>&nbsp;|&nbsp;
      <a href="https://qm.qq.com/q/E1CGjkqgG6" target="_blank" rel="noreferrer" >QQ用户交流群</a>
   </div>
   <div class="one-line">
      致力于普及IPv6，推进IPv6规模部署和应用，以全面推进IPv6技术创新与融合应用为主线，以提升应用广度深度为主攻方向
  </div>
  </footer>

</template>
<style scoped>
@import "~/style.css";
/* 菜单项统一文本样式（替代原内联 style="display: inline-block; margin-left: 10px"） */
.menu-item-text {
  display: inline-block;
  margin-left: 10px;
}
:deep(.shiki span) {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New', monospace !important;
  word-wrap:break-word;
}
:deep(.shiki){
  padding: 20px;
  border-radius: 10px;
}

:deep(.el-menu-item a) {
  font-size: 1em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:deep(.el-menu-item a p) {
  font-size: 1em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
:deep(.el-menu-item a img) {
  width: 50px;
  margin-bottom: 20px;
  
}
/* 顶栏分组之间的竖分隔条（template 里那个 aria-hidden 的 span）。
   数值抄自 element-plus 的 .el-divider--vertical：1px 左边框、1.2em 高、
   margin 左右各 8px、vertical-align: middle；再加上原来行内 style 的 margin-top: 20px
   与 height: 1.2em，保证与替换前的排布一致。 */
.menu-divider {
  display: inline-block;
  position: relative;
  width: 1px;
  height: 1.2em;
  margin: 20px 8px 0;
  vertical-align: middle;
  border-left: 1px solid var(--el-border-color);
}
</style>
<style>
:root {
  --el-color-primary: #3EAF7C;
}
html.dark {
  /* 暗色模式下根元素的文字色与页面底色。
     这两条必须在【非 scoped】块里：<html> 永远不会带上组件 scope 属性，
     若写在 <style scoped>（含被 scoped @import 进来的 style.css）里，会被改写成
     `html.dark[data-v-xxxx]` —— 整条规则永不匹配，底色只能靠浏览器的 color-scheme 兜底。 */
  color: rgba(255, 255, 255, 0.87);
  background-color: #242424;
  --el-color-primary: #3EAF7C;
}
/* Drawer 内部链接占满一行。
   注意用直接子选择器（> a）：文档菜单（DocMenu）里的 <a> 嵌在 el-menu-item 里，
   不能被这套「整行大按钮」规则吃掉，否则菜单项会变成上下带 1em 内边距的块。
   间距三层叠加会挤没文字（body 20px + a 1em + p margin-left 10px），
   所以 body 归零、a 只留一层、p 的 margin 清掉，字号定死 15px 保证最长的
   「IPv6/IPv4 地址查询」在 320px 屏的 60% 抽屉里也放得下。 */
.el-drawer__body {
  overflow-x: hidden;
  padding: 6px 0;
  font-size: 15px;
}
.el-drawer__body > a {
  display: block !important;
  width: 100% !important;
  box-sizing: border-box;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 12px 16px;
  font-size: 15px;
}
.el-drawer__body > a p.menu-item-text {
  display: block !important;
  width: 100% !important;
  box-sizing: border-box;
  margin: 0;
  font-size: 15px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 并进抽屉的文档菜单：与上面的工具链接视觉上分段，并去掉菜单自带底色与右边框
   （.el-menu 默认 border-right，竖排菜单是给页面侧栏用的，在抽屉里会多出一条竖线） */
.el-drawer__body .el-menu {
  background-color: transparent;
  border-right: none;
  border-top: 1px solid var(--el-border-color-lighter);
  padding-top: 8px;
}
/* 手机上条目多，行高压到 44px 让一屏能多看几项（默认 56px）。
   文档标题可能很长（如「Windows 10/11 设置 IPv4/IPv6 访问优先级」），
   截断会丢信息 → 允许换行，高度改 auto 只保 44px 最小行高。 */
/* 手机上条目多，行高压到 44px 让一屏能多看几项（默认 56px）。
   注意：element-plus 竖排菜单规则是
   `.el-menu--vertical:not(..)×3 .el-menu-item`（特异性 5 个类），
   这里必须 !important 才压得住 height / line-height / white-space。
   文档标题可能很长（如「Windows 10/11 设置 IPv4/IPv6 访问优先级」），
   截断会丢信息 → 允许换行，高度改 auto 只保 44px 最小行高。 */
.el-drawer__body .el-menu .el-menu-item,
.el-drawer__body .el-menu .el-sub-menu__title {
  height: auto !important;
  min-height: 44px;
  line-height: 1.45 !important;
  padding-top: 6px;
  padding-bottom: 6px;
  white-space: normal !important; /* 长标题裸文本在 li 上，必须解掉 nowrap */
}
.el-drawer__body .el-menu .el-menu-item a,
.el-drawer__body .el-menu .el-menu-item p {
  white-space: normal;
  word-break: break-word;
}

/* 窄屏（≤768px，与 script 中 isNarrow 的 matchMedia 断点一致）：
   在水合前由 CSS 兜底隐藏宽屏菜单项，避免窄屏设备闪现宽屏布局 */
@media (max-width: 768px) {
  .el-menu--horizontal > .menu-divider {
    display: none !important;
  }
  .el-menu--horizontal > .el-menu-item[index="1"],
  .el-menu--horizontal > .el-menu-item[index="2"],
  .el-menu--horizontal > .el-menu-item[index="3"],
  .el-menu--horizontal > .el-menu-item[index="4"],
  .el-menu--horizontal > .el-menu-item[index="5"],
  .el-menu--horizontal > .el-menu-item[index="6"],
  .el-menu--horizontal > .el-sub-menu[index="7"],
  .el-menu--horizontal > .el-sub-menu[index="8"],
  .el-menu--horizontal > .el-menu-item[index="9"],
  .el-menu--horizontal > .el-sub-menu[index="more"] {
    display: none !important;
  }
}
.el-menu--horizontal {
  --el-menu-hover-bg-color: transparent !important;
  --el-menu-active-color: var(--el-text-color-primary) !important;
  --el-menu-bg-color: transparent !important;
}

/* 顶栏菜单项的点击目标：让内层 <a> 撑满整个 <li>。
   axe 的 target-size（WCAG 2.2 的 2.5.8，24×24 最小目标）会把 <li role="menuitem">
   减掉内层 <a> 之后剩下的「侧翼」算成被遮挡的空间：EP 默认给 li 左右各 20px 内边距，
   而 <a> 只占中间 97px 宽，于是报
   "partially obscured (smallest space is 20px by 59px)" —— 桌面 7 条、权重 7。
   有两条路：① 把 --el-menu-base-level-padding 提到 >24px —— 顶栏整体就变宽，
   1350px 下直接横向溢出 45px（各档折叠阈值还得跟着重算）；② 让 <a> 撑满 li —— 本条。
   选②：li 宽度、文字位置、分隔条位置与改动前【逐像素一致】（CDP 量 rect 比对过），
   可点区域却从 97×59 变成整项 137×59，远超 24×24。
   - flex + align-items:center 是为了复现原来 inline 盒子靠 line-height 得到的垂直居中
     （logo 的 img 顶边仍在 li 顶 +4.5px 处，改完不位移）；
   - 负 margin 抵消 li 的左右内边距，与 padding 取同一个变量，改内边距时不会脱节；
   - height:100% 成立是因为 EP 给 .el-menu-item 写了确定的 height(--el-menu-item-height)。
   必须写在这个【非 scoped】块里：el-menu 是 fragment 根，scope 属性在它自己身上而不是
   祖先，<style scoped> 里的 :deep(.el-menu--horizontal) 选不到它。
   只作用于顶栏这条 ul —— sub-menu 的弹层被 teleport 到 body，跑不到这个选择器下。 */
.el-menu--horizontal > .el-menu-item > a {
  display: flex;
  align-items: center;
  box-sizing: border-box;
  height: 100%;
  margin: 0 calc(-1 * var(--el-menu-base-level-padding));
  padding: 0 var(--el-menu-base-level-padding);
}

/* logo 那个 <a> 单独放开 overflow（选择器多一层 :nth-child(1)，免得靠书写顺序
   去压 scoped 块里 :deep(.el-menu-item a) 的 overflow:hidden）。
   它的 img 带 20px 上下外边距、总高 90px 塞在 59px 的 li 里：原来 <a> 是 inline
   元素、overflow 本来就不生效才没被裁；上面改成 flex 后若不显式放开，logo 会被切掉。 */
.el-menu--horizontal > .el-menu-item:nth-child(1) > a {
  overflow: visible;
}

.el-menu--horizontal > .el-menu-item:nth-child(1) {
  margin-right: auto;
}

/* 去除选中强调和下划线 */
.el-menu--horizontal > .el-menu-item.is-active {
  color: var(--el-text-color-primary) !important;
  background-color: transparent !important;
}

.el-menu--horizontal > .el-menu-item:hover {
  color: var(--el-text-color-primary) !important;
  background-color: transparent !important;
}

/* 去除所有可能的边框和下划线 */
.el-menu--horizontal::after {
  display: none !important;
}

.el-menu--horizontal > .el-menu-item {
  border-bottom: none !important;
  transition: none !important;
}

.el-menu--horizontal > .el-menu-item.is-active::after {
  display: none !important;
}
/* 覆盖选中、悬停和聚焦状态的高亮样式 */


</style>
