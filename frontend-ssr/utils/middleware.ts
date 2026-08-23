import { ref, shallowRef, computed, toValue, type MaybeRefOrGetter } from 'vue'
import { config } from '../config/index'

// 构建中间件候选 URL 数组：
// config.Middleware 外部节点（base URL + 相对路径）在前，
// 前端自带中间件（传过来的相对路径本身，如 /middleware/...，由 Nuxt Server 的 [...slug].get.ts 处理）放最后一位；
// 若 config.EnableInternalMiddleware 为 false，候选列表不含内置中间件。
// 未配置 Middleware 数组或为空时，仅用内置中间件（若同时禁用则为空候选，请求将全部失败）。
function buildCandidates(path: string): string[] {
    const external: string[] = Array.isArray(config.Middleware)
        ? config.Middleware.filter((u): u is string => !!u)
        : []
    const useInternal = config.EnableInternalMiddleware !== false
    if (external.length === 0) return useInternal ? [path] : []
    const candidates = external.map(u => u.replace(/\/$/, '') + path)
    if (useInternal) candidates.push(path)
    return candidates
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
// 401 未授权、403 被 CDN/WAF 风控拦截、418 被反爬/限流标记、502 中间件/上游不可用 —— 换一个节点往往就能恢复
const RETRY_STATUS_CODES = new Set([401, 403, 418, 502])

// 过滤掉 useFetch 专属选项（对 $fetch 无意义），其余（method/query/params/headers/body/timeout 等）原样透传
function toFetchOptions(options?: any) {
    if (!options) return undefined
    const { immediate, watch, key, deep, server, lazy, default: _default, transform, pick, getCachedData, dedupe, ...rest } = options
    return rest
}

// useMiddlewareFetch：在 Vue 页面中调用中间件（替代直接 useFetch('/middleware/...')）。
//
// 候选切换在 execute() 内部以串行重试循环完成（不用 useFetch 的响应式 URL/key 切换——
// Nuxt useAsyncData 在 key 变化时会重新绑定 data/error 状态，导致切换后的数据无法可靠渲染）：
//   - 依次尝试 config.Middleware 中的外部节点 + 前端自带中间件兜底；
//   - 仅当"节点无法连接"或上游返回 401/403/418 时切下一个候选；
//   - 成功后 data 立即更新为最终节点的数据，调用方 await execute() 后可直接读取并渲染；
//   - 全部候选失败时 error 为最终错误。
export function useMiddlewareFetch<T = any>(url: MaybeRefOrGetter<string>, options?: any) {
    const path = computed(() => toValue(url) || '')
    const candidates = computed(() => buildCandidates(path.value))
    const fetchOptions = toFetchOptions(options)

    const data = shallowRef<T | undefined>(undefined)
    const error = shallowRef<any>(null)
    const pending = ref(false)
    // 请求序号：execute 被新调用取代时（如用户连续点击），旧循环直接退出，避免旧结果覆盖新结果
    let requestId = 0

    async function execute(): Promise<T | undefined> {
        const id = ++requestId
        pending.value = true
        error.value = null
        try {
            for (let i = 0; i < candidates.value.length; i++) {
                if (id !== requestId) return undefined // 已被更新的请求取代
                const candidate = candidates.value[i]
                // noUncheckedIndexedAccess 下索引访问为 string | undefined，防御性跳过
                if (candidate === undefined) continue
                try {
                    // $fetch 为 Nuxt 全局注入；候选失败会抛 FetchError（含 status）
                    const res = await $fetch<T>(candidate, fetchOptions)
                    if (id !== requestId) return undefined
                    data.value = res
                    return res
                } catch (e: any) {
                    if (id !== requestId) return undefined
                    const status = getErrorStatus(e)
                    // 不可重试的错误（如 400/500）直接暴露，不浪费候选
                    if (!isConnectionError(e) && !(typeof status === 'number' && RETRY_STATUS_CODES.has(status))) {
                        error.value = e
                        return undefined
                    }
                    // 可重试：继续下一个候选
                }
            }
            error.value = new Error('all middleware candidates failed')
            return undefined
        } finally {
            if (id === requestId) pending.value = false
        }
    }

    return { data, error, pending, execute, refresh: execute }
}
