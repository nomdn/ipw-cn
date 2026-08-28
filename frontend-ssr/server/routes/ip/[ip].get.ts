import { defineEventHandler, createError, setResponseStatus, setResponseHeader, getRequestHeader, getRequestIP } from 'h3'
import { config } from '../../../config/index'
import { useMiddlewareFetch } from '../../../utils/middleware'

// GET /ip/<ip> → text/plain 输出指定 IP 的归属地（支持 IPv4 / IPv6，如 /ip/2001:db8::1）
//
//   IP: 2001:db8::1
//   bilibili: 中国 江苏 南京 电信
//   geocn: 江苏省 南京市 江宁区 电信 宽带
//   ip2region: 中国 江苏 南京 电信
//   maxmind_city: 中国 江苏 南京
//   maxmind_asn: AS4134 中国电信
//   qqwry: 中国 江苏省 南京市 电信
//
// 复用 utils/middleware.ts 的 useMiddlewareFetch()：候选依次为 config.Middleware 外部中间层 + 前端自带中间件兜底，
// 仅当节点无法连接或上游 401/403/418/502 时切换；IPLocationAPI 配置了多个后端时逐个重试。

// ---- 简单限流（每 IP 每分钟次数上限，进程内内存计数；limit 0 表示不限流） ----
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

// ---- 文本美化：把单个数据源的归属地对象拼成可读字符串 ----
function objectToText(obj: any): string {
    const parts: string[] = []
    // 位置字段（按国家 → 省/区域 → 城市 → 区县）
    for (const k of ['country', 'administrative_area', 'region', 'city', 'district', 'province']) {
        const v = obj[k]
        if (typeof v === 'string' && v) parts.push(v)
    }
    // ASN 编号（如 "AS4134"）
    if (typeof obj.asn === 'string' && obj.asn) {
        parts.push(obj.asn.startsWith('AS') ? obj.asn : `AS${obj.asn}`)
    }
    // 运营商 / 组织 / 类型
    for (const k of ['isp', 'org', 'type']) {
        const v = obj[k]
        if (typeof v === 'string' && v) parts.push(v)
    }
    return [...new Set(parts)].join(' ')
}

// ---- 文本美化：多数据源逐行输出（跳过后端未加载的库） ----
function formatLocationText(data: any, ip: string): string {
    const lines: string[] = [`IP: ${ip}`]
    const sources: Array<[string, any]> = [
        ['bilibili', data?.bilibili],
        ['geocn', data?.geocn],
        ['ip2region', data?.ip2region],
        ['ip2location', data?.ip2location],
        ['maxmind_city', data?.maxmind_city],
        ['maxmind_asn', data?.maxmind_asn],
        ['dbip_city', data?.dbip_city],
        ['dbip_asn', data?.dbip_asn],
        ['qqwry', data?.qqwry],
    ]
    for (const [name, value] of sources) {
        if (value === undefined || value === null) continue
        if (typeof value === 'string') {
            // 纯字符串结果（error 等）原样输出，"not loaded" 跳过
            if (value && value !== 'not loaded') {
                lines.push(`${name}: ${value}`)
            }
            continue
        }
        if (typeof value === 'object') {
            const text = objectToText(value)
            if (text) lines.push(`${name}: ${text}`)
        }
    }
    return lines.join('\n')
}

// ---- 核心：查询指定 IP 的归属地（IPLocationAPI 多后端逐个重试） ----
async function locate(event: any, ip: string) {
    const backends = config.IPLocationAPI
    if (!backends || backends.length === 0) {
        throw createError({ statusCode: 500, statusMessage: 'No location backend configured' })
    }
    let lastError: any = null
    for (const backend of backends) {
        const path = `/middleware/${backend.id}/location/${ip}`
        const { data, error, execute } = useMiddlewareFetch(path)
        await execute()
        if (error.value) {
            lastError = error.value
            console.error(`[ip] backend "${backend.label}" failed for ${ip}:`, error.value?.message || error.value)
            continue
        }
        // 成功：location 响应必含 ip 字段；缺失视为失败（如误返回错误体），换下一个后端
        const res = data.value
        if (res && typeof res === 'object' && (res as any).ip) {
            return formatLocationText(res, ip)
        }
        lastError = new Error('invalid location response')
    }

    // 全部后端失败：透传最后一次错误（text/plain）
    const errStatus = lastError?.status ?? lastError?.statusCode ?? 502
    const status = typeof errStatus === 'number' ? errStatus : 502
    setResponseStatus(event, status)
    return `Error ${status}: ${lastError?.message || 'Backend unreachable'}`
}

export default defineEventHandler(async (event) => {
    // 0) 统一输出 text/plain（不是 JSON）
    setResponseHeader(event, 'content-type', 'text/plain; charset=utf-8')

    // 1) Origin / Referer 校验：跨域浏览器调用一律拒绝
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

    // 3) 取路径参数 IP
    const ipParam: string = event.context.params?.ip as any
    let ip = ipParam ? ipParam.trim() : ''
    if (!ip) {
        throw createError({ statusCode: 400, statusMessage: 'Missing IP parameter' })
    }

    return locate(event, ip)
})
