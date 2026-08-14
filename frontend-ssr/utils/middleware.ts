import { ref, computed, watch, toValue, type MaybeRefOrGetter } from 'vue'
import { useFetch } from 'nuxt/app'
import { config } from '../config/index'

// 构建中间件候选 URL 数组：
// config.Middleware 外部节点（base URL + 相对路径）在前，
// 前端自带中间件（传过来的相对路径本身，如 /middleware/...，由 useFetch 自行解析、走 Nuxt Server 的 [...slug].get.ts）放最后一位。
// 未配置 Middleware 数组或为空时，直接返回 [path]（只用前端自带中间件）。
function buildCandidates(path: string): string[] {
    const external: string[] = Array.isArray(config.Middleware)
        ? config.Middleware.filter((u): u is string => !!u)
        : []
    if (external.length === 0) return [path]
    return [...external.map(u => u.replace(/\/$/, '') + path), path]
}

// 获取错误的状态码：优先 status（ofetch 2.x），兼容 statusCode（旧版 ofetch）
function getErrorStatus(e: any): number | undefined {
    return e?.status ?? e?.statusCode
}

// 判断是否为"节点无法连接"错误（网络层错误：无 HTTP 状态码）
function isConnectionError(e: any): boolean {
    return !!e && typeof getErrorStatus(e) !== 'number'
}

// 需要切换节点重试的上游状态码：
// 401 未授权、403 被 CDN/WAF 风控拦截、418 被反爬/限流标记 —— 换一个节点往往就能绕过
const RETRY_STATUS_CODES = new Set([401, 403, 418])

// useMiddlewareFetch：在 Vue 页面中调用中间件（替代直接 useFetch('/middleware/...')）。
// 外部中间件与前端自带中间件同级：依次尝试 config.Middleware 中的外部节点，
// 仅当"节点无法连接"或上游返回 401/403/418 时重试下一个候选（最后一位兜底为前端自带中间件）；
// 其余状态码错误不重试，直接返回给页面展示。
export function useMiddlewareFetch<T = any>(url: MaybeRefOrGetter<string>, options?: any) {
    const path = computed(() => toValue(url) || '')
    const candidates = computed(() => buildCandidates(path.value))
    // 当前尝试的候选下标
    const idx = ref(0)
    const currentUrl = computed(() => candidates.value[idx.value] ?? path.value)

    const result = useFetch<T>(currentUrl, options)

    // 仅"节点无法连接"或命中 RETRY_STATUS_CODES 时切换到下一个候选并重新请求
    watch(() => result.error.value, (err) => {
        if (!err) return
        if (idx.value >= candidates.value.length - 1) return
        const status = getErrorStatus(err)
        if (!isConnectionError(err) && !(typeof status === 'number' && RETRY_STATUS_CODES.has(status))) return
        idx.value++
        result.execute()
    })

    return result
}
