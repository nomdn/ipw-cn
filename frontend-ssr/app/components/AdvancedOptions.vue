<script setup lang="ts">
import { ref } from 'vue'
import { isIPv6 } from 'is-ip'

/**
 * 高级选项：请求级解析参数（与 ipw-ak 的 AdvancedOptions 同源）
 *
 * - 展开后按地址族分列：IPv4 → dns4 / resolve4，IPv6 → dns6 / resolve6
 * - stack 非空时（单栈页）只渲染对应栈那一列，且不显示 dns / resolve 通用行：
 *   单栈页里两者与 dns4=/dns6= 等价，双份输入框只会让人纠结该填哪个；
 *   tcping / speed 的前端两个栈是分开的页面，另一列的参数永远吃不到。
 * - dns / resolve 是双栈页的两栈通用兜底，dns4/dns6、resolve4/resolve6 非空时覆盖它们。
 *
 * 用法：
 *   <AdvancedOptions ref="adv" />                         双栈页（dns4/dns6 + resolve4/resolve6）
 *   <AdvancedOptions ref="adv" stack="ipv4" />            单栈页，只给 IPv4 一列
 *   <AdvancedOptions ref="adv" :show-resolve="false" />    仅 dns4/dns6（dns、whois 页不支持强制解析）
 *
 * 页面里取参数：
 *   withQuery(baseUrl, adv.value?.query() ?? {})
 */
const props = withDefaults(
  defineProps<{
    /** 是否显示 resolve / resolve4 / resolve6（dns、whois 页不支持强制解析） */
    showResolve?: boolean
    /** 只显示某一栈的配置项；单栈页面（tcping / speed 的 v4、v6 页）传，双栈页面不传 */
    stack?: 'ipv4' | 'ipv6' | ''
  }>(),
  { showResolve: true, stack: '' },
)

const expanded = ref(false)
/** 两栈通用的 DNS 服务器：dns4/dns6 非空时覆盖它 */
const dns = ref('')
const dns4 = ref('')
const dns6 = ref('')
/** 两栈通用的强制解析：resolve4/resolve6 非空时覆盖它 */
const resolve = ref('')
const resolve4 = ref('')
const resolve6 = ref('')

/** 域名 HOST 判定：含点、标签为字母数字/连字符/下划线（宽松匹配，畸形值交给后端报错） */
const DOMAIN_RE = /^[a-z0-9_](?:[a-z0-9_-]*[a-z0-9_])?(?:\.[a-z0-9_](?:[a-z0-9_-]*[a-z0-9_])?)+$/i
const IPV4_RE = /^\d{1,3}(?:\.\d{1,3}){3}$/

/**
 * 把用户输入的单个 DNS 服务器条目规整成后端认可的形式：
 *  - 已带协议的 URL：路径为空或仅 "/" 时补上 DoH 标准路径 /dns-query
 *  - 无协议但含 "/"（缺了 scheme 的 DoH 地址）：补 https:// 前缀，路径按上条处理
 *  - 纯域名 HOST（无端口无路径）：包裹为 https://域名/dns-query
 *  - 无端口的裸 IP（v4 / 裸 v6）：补默认端口 53（v6 顺手加方括号）
 *  - 其余（host:port、[v6]:53 等）原样返回，交给后端校验
 */
function normalizeDnsEntry(raw: string): string {
  const v = raw.trim()
  if (!v) return ''

  // 1) 已带协议的 URL
  if (/^[a-z][a-z0-9+.-]*:\/\//i.test(v)) return completeDohPath(v)
  // 2) 无协议但含 "/"：缺 scheme 的 DoH 地址
  if (v.includes('/')) return completeDohPath('https://' + v)
  // 3) 带方括号的 v6（含/含端口均原样，端口要求交后端校验）
  if (v.startsWith('[')) return v
  // 4) host:port：IP 或域名都按 UDP 语义原样（写了端口就尊重端口）
  if (/^[^:]+:\d+$/.test(v)) return v
  // 5) 无方括号的裸 v6：补 [..]:53
  if (isIPv6(v)) return `[${v}]:53`
  // 6) 裸 IPv4：补 :53
  if (IPV4_RE.test(v)) return `${v}:53`
  // 7) 纯域名 HOST：包裹为标准 DoH URL
  if (DOMAIN_RE.test(v)) return `https://${v}/dns-query`

  return v
}

/** 补齐 DoH URL 的标准路径：空路径/根路径 → /dns-query；自带路径（如 /custom）原样保留 */
function completeDohPath(url: string): string {
  try {
    const u = new URL(url)
    if (!u.hostname) return url
    if (u.pathname === '/' || u.pathname === '') u.pathname = '/dns-query'
    return u.toString()
  } catch {
    return url
  }
}

/** 逗号分隔列表逐项规整（空项丢弃） */
function normalizeDnsList(value: string): string {
  return value
    .split(',')
    .map(p => normalizeDnsEntry(p))
    .filter(Boolean)
    .join(',')
}

/** 组装请求级解析参数（空值不下发；DNS 服务器逐项规整） */
function query(): Record<string, string> {
  const q: Record<string, string> = {}
  if (dns.value.trim()) q.dns = normalizeDnsList(dns.value)
  if (dns4.value.trim()) q.dns4 = normalizeDnsList(dns4.value)
  if (dns6.value.trim()) q.dns6 = normalizeDnsList(dns6.value)
  if (props.showResolve) {
    if (resolve.value.trim()) q.resolve = resolve.value.trim()
    if (resolve4.value.trim()) q.resolve4 = resolve4.value.trim()
    if (resolve6.value.trim()) q.resolve6 = resolve6.value.trim()
  }
  return q
}

defineExpose({ query })
</script>

<template>
  <div class="adv">
    <!-- 用原生 button：页面的全局 .el-button{width:165px} 会把 el-button 撑成实心大按钮 -->
    <button type="button" class="adv__toggle" :aria-expanded="expanded" @click="expanded = !expanded">
      <span class="adv__caret" :class="{ 'is-open': expanded }">▸</span>
      <span>高级选项</span>
    </button>

    <div v-if="expanded" class="adv__body">
      <!-- 两栈通用的 DNS 服务器（dns4/dns6 非空时覆盖它）。仅双栈页显示：
           单栈页里 dns= 与 dns4=/dns6= 对该栈请求效果等价（后端按栈取值），
           再给一个"两栈通用"行只会让人纠结该填哪个 —— 单栈只留对应栈那一列。 -->
      <div v-if="!stack" class="adv__field adv__field--wide">
        <span class="adv__label">DNS 服务器（dns，两栈通用）</span>
        <el-input v-model="dns" clearable placeholder="如 1.1.1.1:53 / doh.pub / dns.google，逗号分隔" />
      </div>

      <!-- 两栈通用的强制解析（resolve4/6 非空时覆盖它），同上仅双栈页显示 -->
      <div v-if="!stack && showResolve" class="adv__field adv__field--wide">
        <span class="adv__label">强制解析（resolve，两栈通用）</span>
        <el-input v-model="resolve" clearable placeholder="如 example.com:1.2.3.4，逗号分隔" />
      </div>

      <div class="adv__grid" :class="{ 'is-single': !!stack }">
        <section v-if="!stack || stack === 'ipv4'" class="adv__col">
          <h4 class="adv__col-title">IPv4</h4>
          <div class="adv__field">
            <span class="adv__label">DNS 服务器（dns4）</span>
            <el-input v-model="dns4" clearable placeholder="如 1.1.1.1:53 / doh.pub，逗号分隔" />
          </div>
          <div v-if="showResolve" class="adv__field">
            <span class="adv__label">强制解析（resolve4）</span>
            <el-input v-model="resolve4" clearable placeholder="如 example.com:1.2.3.4" />
          </div>
        </section>

        <section v-if="!stack || stack === 'ipv6'" class="adv__col">
          <h4 class="adv__col-title">IPv6</h4>
          <div class="adv__field">
            <span class="adv__label">DNS 服务器（dns6）</span>
            <el-input v-model="dns6" clearable placeholder="如 [2400:3200::1]:53，逗号分隔" />
          </div>
          <div v-if="showResolve" class="adv__field">
            <span class="adv__label">强制解析（resolve6）</span>
            <el-input v-model="resolve6" clearable placeholder="如 example.com:[::1]" />
          </div>
        </section>
      </div>

      <p class="adv__hint">
        DNS 服务器可填 <code>ip:port</code>、DoH 地址或纯域名（自动补全为
        <code>https://域名/dns-query</code>），逗号分隔多个；强制解析格式
        <code>域名:IP</code> 或 <code>域名:CNAME</code>，IPv6 目标写 <code>[::1]</code>。{{
          stack ? '参数直接作用于本栈解析；双栈页的 dns / resolve 通用项在此页不展示。'
                : 'dns4 / dns6、resolve4 / resolve6 非空时覆盖 dns / resolve，分别只作用于对应栈。'
        }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.adv {
  margin-top: 0.25rem;
}

.adv__toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  width: auto;
  height: auto;
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  color: var(--el-color-primary, #299764);
  font: inherit;
  font-size: 0.85rem;
  letter-spacing: 0.03em;
  cursor: pointer;
}

.adv__caret {
  display: inline-block;
  font-size: 0.7rem;
  transition: transform 150ms ease;
}

.adv__caret.is-open {
  transform: rotate(90deg);
}

.adv__body {
  margin-top: 0.5rem;
  padding: 0.75rem 0.9rem;
  border: 1px solid var(--el-border-color-lighter, #ebeef5);
  border-radius: 6px;
  background: var(--el-fill-color-lighter, #fafafa);
}

.adv__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 1rem;
}

/* 单栈页只渲染一列：撑满，别再留半幅空白 */
.adv__grid.is-single {
  grid-template-columns: 1fr;
}

.adv__col-title {
  margin: 0 0 0.4rem;
  font-size: 0.8rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary, #909399);
}

.adv__field {
  margin-bottom: 0.5rem;
}

.adv__field--wide {
  margin-bottom: 0.75rem;
}

.adv__label {
  display: block;
  margin-bottom: 0.25rem;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary, #909399);
}

.adv__hint {
  margin: 0.35rem 0 0;
  font-size: 0.75rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary, #909399);
}

.adv__hint code {
  padding: 0 0.2rem;
  border-radius: 2px;
  background: var(--el-fill-color, #f0f2f5);
}

@media (max-width: 768px) {
  .adv__grid {
    grid-template-columns: 1fr;
  }
}
</style>
