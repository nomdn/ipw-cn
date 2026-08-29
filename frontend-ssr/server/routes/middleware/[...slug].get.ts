import { defineEventHandler, getQuery, createError, setResponseStatus, getRequestHeader, getRequestIP } from 'h3'
import { config } from '../../../config/index'

// ---- 方案 C：简单限流（每 IP 每分钟次数上限，进程内内存计数；次数取 config.rateLimitPerMinute，0 表示不限流） ----
const rateLimitMap = new Map<string, { count: number; resetAt: number }>()

function checkRateLimit(ip: string, limit: number): boolean {
    const now = Date.now()
    const entry = rateLimitMap.get(ip)
    if (!entry || entry.resetAt <= now) {
        rateLimitMap.set(ip, { count: 1, resetAt: now + 60_000 })
        // 顺带清理过期条目，避免 Map 无限增长
        if (rateLimitMap.size > 1000) {
            for (const [k, v] of rateLimitMap) {
                if (v.resetAt <= now) rateLimitMap.delete(k)
            }
        }
        return true
    }
    entry.count++
    return entry.count <= limit
}

// ---- 数据上报（协议见收集中心中间件 ipw-boce 的 report.go）：调一次上报一次 ----
// 配置经 runtimeConfig 从环境变量注入：
//   NUXT_BOCE_REPORT_URL       收集中心基址（如 https://collector.example.com）；空 = 不上报
//   NUXT_BOCE_REPORT_TOKEN     /report 鉴权 token（敏感，仅环境变量，不进仓库）
//   NUXT_BOCE_REPORT_INSTANCE  上报方标识（存收集库 probe_results.origin）；空 = 默认 frontend-ssr
// 语义：本入口只报"自己转发的请求"（fire-and-forget，失败丢弃不影响主链路）；
//       配置了上报时，转发请求会带 X-Boce-Reporter 标记头，节点据此跳过，避免双算。
const BOCE_PROBE_TYPES = new Set(['tcping', 'udping', 'speed'])
const BOCE_INSTANCE_DEFAULT = 'frontend-ssr'

function boceReportOnce(rt: any, backendID: string, apiType: string, raw: string, status: number, latencyMs: number, body?: any) {
    const base: string = rt.boceReportUrl || ''
    if (!base) return
    const payload: Record<string, any> = {
        instance: rt.boceReportInstance || BOCE_INSTANCE_DEFAULT,
        stats: [{
            nodeId: backendID,
            apiType,
            total: 1,
            errors: status >= 500 ? 1 : 0,
            latencySumMs: latencyMs,
            latencyMaxMs: latencyMs,
            minute: 0, // 0 = 收集器按自己时钟入桶（前端时钟不可信也不影响聚合口径）
        }],
    }
    if (BOCE_PROBE_TYPES.has(apiType)) {
        payload.probes = [{
            nodeId: backendID,
            apiType,
            raw,
            status,
            latencyMs,
            source: 'http',
            body: body === undefined || body === null ? undefined : body,
        }]
    }
    const base2 = base.replace(new RegExp('/+$'), '')
    $fetch(`${base2}/report`, {
        method: 'POST',
        headers: {
            authorization: `Bearer ${rt.boceReportToken || ''}`,
            'content-type': 'application/json',
        },
        body: payload,
        timeout: 5_000,
    }).catch(() => {}) // 上报失败静默：不影响转发响应
}

export default defineEventHandler(async (event) => {
    // 1) Origin / Referer 校验：跨域浏览器调用一律拒绝。
    //    注意：无 Origin/Referer 的服务端 SSR 调用与命令行请求无法区分，仅靠下方限流约束。
    const siteOrigin = new URL(config.siteUrl).origin
    const origin = getRequestHeader(event, 'origin')
    const referer = getRequestHeader(event, 'referer')
    if (origin && origin !== siteOrigin) {
        throw createError({ statusCode: 403, statusMessage: 'Forbidden: cross-origin request' })
    }
    if (referer && referer !== siteOrigin && !referer.startsWith(siteOrigin + '/')) {
        throw createError({ statusCode: 403, statusMessage: 'Forbidden: invalid referer' })
    }
    // 2) 每 IP 每分钟限流（次数取 config.rateLimitPerMinute，0 表示不限流）
    const rateLimit = config.rateLimitPerMinute || 0
    if (rateLimit > 0 && !checkRateLimit(getRequestIP(event) || 'unknown', rateLimit)) {
        throw createError({ statusCode: 429, statusMessage: 'Too Many Requests' })
    }

    const slugString: string = event.context.params?.slug as any
    if (!slugString) {
        throw createError({ statusCode: 400, statusMessage: 'Missing slug parameter' })
    } 
    // 将路径参数转为数组
    let slug: string[] = slugString ? slugString.split('/') : []
    console.debug('[middleware debug] raw slugString:', slugString, '| split:', slug)
    // 验证 slug 数组的长度是否在允许范围内
    if (!slug || Object.keys(slug).length === 0) {
        throw createError({ statusCode: 400, statusMessage: 'Invalid slug' })
    }
    // 如果分段超过4个，可能是 raw 部分包含协议（如 https://example.com），
    // 按 / 分割后会产生额外分段，需要重新拼回
    if (slug.length > 4) {
        const backendID = slug[0]!
        const apiType = slug[1]!
        const protocol = slug[2]!
        const rest = slug.slice(3).filter(Boolean).join('/')
        if (protocol === 'https:' || protocol === 'http:') {
            slug = [backendID, apiType, `${protocol}//${rest}`]
        } else {
            throw createError({ statusCode: 400, statusMessage: 'Invalid slug' })
        }
    }
    console.log('[middleware debug] final slug:', slug)
    // 分割参数
    const backendID = slug[0]
    const apiType = slug[1]
    const raw = slug.slice(2).join('/')
    if (!backendID || !apiType || !raw) {
        throw createError({ statusCode: 400, statusMessage: 'Missing parameters in slug' })
    }
    // 阻断路径穿越：raw 里的 ".." 会被上游 URL 规范化解析掉，
    // 把请求（连同节点的 Authorization 头）带到 /v1 之外的任意路径
    if (raw.split('/').some(seg => seg === '.' || seg === '..')) {
        throw createError({ statusCode: 400, statusMessage: 'Invalid path' })
    }
    const query = getQuery(event)
    const runtimeConfig = useRuntimeConfig(event)

    // 从运行时变量读取 APIKEYS (JSON字符串)，解析后按 backendID 查找 token
    // （runtimeConfig 同时提供数据上报配置 boceReport*，见文件头说明）
    let apiKey: string | undefined
    try {
        const apiKeysMap: Record<string, string> = (runtimeConfig.apiKeys || {}) as Record<string, string>
        apiKey = apiKeysMap[backendID]
        console.debug('[middleware debug] runtimeConfig.apiKeys raw:', runtimeConfig.apiKeys ? 'set (len=' + runtimeConfig.apiKeys.length + ')' : 'NOT SET')
        console.debug('[middleware debug] backendID:', backendID, '| apiKeysMap keys:', Object.keys(apiKeysMap))
        console.debug('[middleware debug] matched apiKey:', apiKey ? 'FOUND (' + apiKey.slice(0, 1) + '...)' : 'NOT FOUND')
    } catch (e) {
        console.debug('[middleware debug] JSON parse FAILED:', e)
    }
    // 转发头：不透传 Origin，避免上游 CORS 误判；仅按需注入 Authorization。
    // 配置了数据上报时注入 X-Boce-Reporter 标记头（本入口会计数上报，节点据此跳过防双算；
    // 未配置上报则不打标记，由节点兜底记账）
    const authHeaders: Record<string, string> = {}
    if (apiKey) {
        authHeaders['Authorization'] = `Bearer ${apiKey}`
        console.debug('[middleware debug] authHeaders SET Authorization:', `Bearer ${apiKey.slice(0, 1)}...`)
    } else {
        console.debug('[middleware debug] authHeaders NO Authorization header')
    }
    const boceInstance = (runtimeConfig.boceReportInstance || BOCE_INSTANCE_DEFAULT) as string
    if (runtimeConfig.boceReportUrl) {
        authHeaders['X-Boce-Reporter'] = boceInstance
    }
    
    if (apiType === 'whois' || apiType === 'dns' || apiType === 'location' || apiType === 'ssl' || apiType === 'asn' || apiType === 'dnssec' || apiType === 'detail') {
        let apiBaseUrls: Array<{ id: string, url: string ,label: string}> = []
        switch (apiType) {
            case 'whois':
            case 'ssl':
            case 'detail':
                apiBaseUrls = [...config.APIBaseURL.DualStack]
                break
            case 'dns':
            case 'dnssec':
                apiBaseUrls = [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4, ...config.APIBaseURL.IPv6]
                break
            case 'location':
            case 'asn':
                apiBaseUrls = [...config.IPLocationAPI]
                break
            default:
                throw createError({ statusCode: 400, statusMessage: 'Invalid API type' })
        }
        const map = new Map<string, string>(apiBaseUrls.map(api => [api.id, api.url]))
        if (!map.has(backendID)) {
            throw createError({ statusCode: 400, statusMessage: 'Invalid backend ID' })
        }
        let apiBaseUrl = map.get(backendID)
        // url 为空的节点无法转发（会变成对前端自身的请求），直接报 502 而不是浪费一次自请求
        if (!apiBaseUrl) {
            throw createError({ statusCode: 502, statusMessage: 'Backend URL not configured' })
        }
        let data: any = {}

        if (apiBaseUrl.slice(-1) != '/') {
            apiBaseUrl = `${apiBaseUrl}/`
            }
        const boceStart = Date.now()
        let upstreamStatus = 200
            data = await $fetch(`${apiBaseUrl}v1/${apiType}/${raw}`, {
                method: 'GET',
                headers: authHeaders,
                timeout: 15_000,
            }).catch((error: any) => {
                console.error(`Error fetching from ${apiBaseUrl}:`, error)
                const errStatus = error?.status ?? error?.statusCode
                if (errStatus) {
                    // 上游返回的错误响应（4xx/5xx），直接转发其状态码与错误体，不包装成 500
                    upstreamStatus = errStatus
                    setResponseStatus(event, errStatus)
                    return error.data ?? {}
                }
                // 网络层错误（无法连接上游）
                upstreamStatus = 502
                setResponseStatus(event, 502)
                return { statusCode: 502, statusMessage: 'Backend unreachable' }
            })
        boceReportOnce(runtimeConfig, backendID, apiType, raw, upstreamStatus, Date.now() - boceStart)
        return data
    }else if (apiType === 'tcping' || apiType === 'udping' || apiType === 'speed') {
        // tcping/udping/speed 统一走 APIBaseURL 节点池（平铺三栈）
        const apiBaseUrls: Array<{ id: string, url: string ,label: string}> = [
            ...config.APIBaseURL.DualStack,
            ...config.APIBaseURL.IPv4,
            ...config.APIBaseURL.IPv6
        ]
        const map = new Map<string, string>(apiBaseUrls.map(api => [api.id, api.url]))
        if (!map.has(backendID)) {
            throw createError({ statusCode: 400, statusMessage: 'Invalid backend ID' })
        }
        let apiBaseUrl = map.get(backendID)
        if (!apiBaseUrl) {
            throw createError({ statusCode: 502, statusMessage: 'Backend URL not configured' })
        }
        let data: any = {}

        if (apiBaseUrl.slice(-1) != '/') {
            apiBaseUrl = `${apiBaseUrl}/`
            }

        const queryString = new URLSearchParams(
            Object.entries(query).filter(([_, v]) => v !== undefined) as [string, string][]
        ).toString()

        const boceStart = Date.now()
        let upstreamStatus = 200
        data = await $fetch(`${apiBaseUrl}v1/${apiType}/${raw}${queryString ? '?' + queryString : ''}`, {
                method: 'GET',
                headers: authHeaders,
                timeout: 30_000, // tcping/测速上游本身较慢，给更长超时
            }).catch((error: any) => {
                console.error(`Error fetching from ${apiBaseUrl}:`, error)
                const errStatus = error?.status ?? error?.statusCode
                if (errStatus) {
                    // 上游返回的错误响应（4xx/5xx），直接转发其状态码与错误体，不包装成 500
                    upstreamStatus = errStatus
                    setResponseStatus(event, errStatus)
                    return error.data ?? {}
                }
                // 网络层错误（无法连接上游）
                upstreamStatus = 502
                setResponseStatus(event, 502)
                return { statusCode: 502, statusMessage: 'Backend unreachable' }
            })
        boceReportOnce(runtimeConfig, backendID, apiType, raw, upstreamStatus, Date.now() - boceStart, data)
        return data

        
    }

    })
