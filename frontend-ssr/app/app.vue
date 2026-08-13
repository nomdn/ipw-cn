<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue';
import { useDark, useToggle } from '@vueuse/core';
import { Moon, Sunny, Expand } from '@element-plus/icons-vue';
import { config } from '../config/index';

const isNarrow = ref(false);
let mediaQueryList: MediaQueryList | null = null;
const drawer = ref(false);

const isDark = useDark();
const toggleDark = useToggle(isDark);

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
  { kind: 'item', index: '1', label: 'IPv6 网站检测', to: '/ipv6webcheck', minWidth: 950 },
  { kind: 'item', index: '2', label: 'IPv6/IPv4 地址查询', to: '/location', minWidth: 990 },
  { kind: 'item', index: '3', label: 'IPv6 TCPing测试', to: '/ipv6tcping', minWidth: 1030 },
  { kind: 'divider', minWidth: 1080 }, // 与 index 4 相同：4-6 段全折叠时隐藏
  { kind: 'item', index: '4', label: 'IPv6 DNS解析', to: '/dns', minWidth: 1080 },
  { kind: 'item', index: '5', label: 'IPv6 SSL检查', to: '/ssl', minWidth: 1130 },
  { kind: 'item', index: '6', label: 'IPv6 网站测速', to: '/ipv6speedtest', minWidth: 1180 },
  { kind: 'divider', minWidth: 1240 }, // 与 index 7 相同：7-9 段全折叠时隐藏
  {
    kind: 'group', index: '7', label: 'IPv4工具箱', minWidth: 1240,
    children: [
      { index: '7-0', label: 'IPv4 网站测速', to: '/speedtest' },
      { index: '7-1', label: 'IPv4 TCPing测试', to: '/tcping' },
    ],
  },
  {
    kind: 'group', index: '8', label: '其他工具', minWidth: 1300,
    children: [
      { index: '8-0', label: '网站截图', to: '/screenshot' },
      { index: '8-1', label: 'Whois查询', to: '/whois' },
      { index: '8-3', label: 'ASN查询', to: '/asn' },
      { index: '8-4', label: 'DNSSEC验证', to: '/dnssec' },
    ],
  },
  { kind: 'item', index: '9', label: '文档', to: '/doc', minWidth: 1360 },
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

// 实时同步视口宽度（resize 触发，驱动折叠）
function updateViewport() {
  viewportWidth.value = window.innerWidth
}
let umamiScript: HTMLScriptElement | null = null
useHead({
  meta: config.noindex
    ? [
        { name: 'robots', content: 'noindex, nofollow' },
        { name: 'googlebot', content: 'noindex, nofollow' },
        { name: 'bingbot', content: 'noindex, nofollow' },
      ]
    : [],
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
  
  <el-drawer v-if="isNarrow" v-model="drawer" direction="ltr" style="height: 100%;" size="50%">
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
      <a href="/ipv6speedtest"><p class="menu-item-text">IPv6 网站测速</p></a>
      <a href="/speedtest"><p class="menu-item-text">IPv4 网站测速</p></a>
      <a href="/tcping"><p class="menu-item-text">IPv4 TCPing</p></a>
      <a href="/screenshot"><p class="menu-item-text">网站截图</p></a>
      <a href="/whois"><p class="menu-item-text">Whois查询</p></a>
      <a href="/asn"><p class="menu-item-text">ASN查询</p></a>
      <a href="/dnssec"><p class="menu-item-text">DNSSEC验证</p></a>
  </el-drawer>
  <el-menu
      mode="horizontal"
      :ellipsis="false"
      
    >
    <el-menu-item index="0">
      <el-icon v-if="isNarrow" @click="drawer = !drawer"><Expand /></el-icon>
      <router-link to="/">
        <el-image src="/favicon.svg" style="margin-top: 20px;" /> 
        <h2 style="display: inline-block; margin-left: 10px" v-if="!isNarrow">柠檬味ipw.cn</h2>
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
      <el-divider v-else style="margin-top: 20px;height: 1.2em;" direction="vertical"/>
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
    <el-menu-item index="10">
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

  <footer>
    <div class="one-line">
      Copyright © nomdn & IP 查询 2026  | <img src="/ipv6-s1.svg" alt="IPv6 相关标识"/> | <img src="/ssl-s1.svg" alt="SSL 相关标识"/> | All right reserved
    </div>
    <div class="one-line">
      <a v-if="config.ICP" href="https://beian.miit.gov.cn/" target="_blank" rel="noreferrer" >{{ config.ICP }}</a>
      <span v-if="config.ICP">&nbsp;|&nbsp;</span>
      <el-image v-if="config.GongAn" style="height: 1em; width: 1em;" src="/备案图标.png" />
      <a :href="'https://beian.mps.gov.cn/#/query/webSearch?code=' + cleanChineseCharacters(config.GongAn)" target="_blank" rel="noreferrer" >{{ config.GongAn }}</a>
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
</style>
<style>
:root {
  --el-color-primary: #3EAF7C;
}
html.dark {
  --el-color-primary: #3EAF7C;
}

/* Drawer 内部链接占满一行 */
.el-drawer__body {
  overflow-x: hidden;
}
.el-drawer__body a {
  display: block !important;
  width: 100% !important;
  box-sizing: border-box;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 1em;
}
.el-drawer__body a p {
  display: block !important;
  width: 100% !important;
  box-sizing: border-box;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 窄屏（≤768px，与 script 中 isNarrow 的 matchMedia 断点一致）：
   在水合前由 CSS 兜底隐藏宽屏菜单项，避免窄屏设备闪现宽屏布局 */
@media (max-width: 768px) {
  .el-menu--horizontal > .el-divider {
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
