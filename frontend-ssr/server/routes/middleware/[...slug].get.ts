import { defineEventHandler, getQuery, createError, setResponseStatus } from 'h3'
import { config } from '../../../config/index'
export default defineEventHandler(async (event) => {
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
    const query = getQuery(event)

    // 从运行时变量读取 APIKEYS (JSON字符串)，解析后按 backendID 查找 token
    const runtimeConfig = useRuntimeConfig(event)
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
    // 转发头：不透传 Origin，避免上游 CORS 误判；仅按需注入 Authorization
    const authHeaders: Record<string, string> = {}
    if (apiKey) {
        authHeaders['Authorization'] = `Bearer ${apiKey}`
        console.debug('[middleware debug] authHeaders SET Authorization:', `Bearer ${apiKey.slice(0, 1)}...`)
    } else {
        console.debug('[middleware debug] authHeaders NO Authorization header')
    }
    
    if (apiType === 'whois' || apiType === 'dns' || apiType === 'location' || apiType === 'ssl' || apiType === 'asn' || apiType === 'dnssec' || apiType === 'detail') {
        let apiBaseUrls: Array<{ id: string, url: string ,label: string}> = []
        switch (apiType) {
            case 'whois':
                apiBaseUrls = [...config.apiBaseUrls]
                break
            case 'dns':
                apiBaseUrls = [...config.NSLookup]
                break
            case 'location':
                apiBaseUrls = [...config.IPLocationAPIs]
                break
            case 'ssl':
                apiBaseUrls = [...config.apiBaseUrls]
                break
            case 'asn':
                apiBaseUrls = [...config.IPLocationAPIs]
                break
            case 'dnssec':
                apiBaseUrls = [...config.NSLookup]
                break
            case 'detail':
                apiBaseUrls = [...config.apiBaseUrls]
                break
            default:
                throw createError({ statusCode: 400, statusMessage: 'Invalid API type' })
        }
        const map = new Map<string, string>(apiBaseUrls.map(api => [api.id, api.url]))
        if (!map.has(backendID)) {
            throw createError({ statusCode: 400, statusMessage: 'Invalid backend ID' })
        }
        let apiBaseUrl = map.get(backendID)
        if (apiBaseUrl === undefined) {
            throw createError({ statusCode: 400, statusMessage: 'API base URL not found for the given backend ID' })
        }
        let data: any = {}

        if (apiBaseUrl.slice(-1) != '/') {
            apiBaseUrl = `${apiBaseUrl}/`
            }
            data = await $fetch(`${apiBaseUrl}v1/${apiType}/${raw}`, {
                method: 'GET',
                headers: authHeaders,
            }).catch((error: any) => {
                console.error(`Error fetching from ${apiBaseUrl}:`, error)
                const errStatus = error?.status ?? error?.statusCode
                if (errStatus) {
                    // 上游返回的错误响应（4xx/5xx），直接转发其状态码与错误体，不包装成 500
                    setResponseStatus(event, errStatus)
                    return error.data ?? {}
                }
                // 网络层错误（无法连接上游）
                setResponseStatus(event, 502)
                return { statusCode: 502, statusMessage: 'Backend unreachable' }
            })
        return data
    }else if (apiType === 'tcping' || apiType === 'udping' || apiType === 'speed') {
        let apiBaseUrls:Array<{ id: string, url: string ,label: string}> = []
        let apiTypeConfig: any = {}
        switch (apiType) {
            case 'tcping':
                apiTypeConfig = config.TCPing
                apiBaseUrls = [...apiTypeConfig.DualStack, ...apiTypeConfig.IPv4, ...apiTypeConfig.IPv6]
                break
            case 'speed':
                apiTypeConfig = config.SpeedTest
                apiBaseUrls = [...apiTypeConfig.DualStack, ...apiTypeConfig.IPv4, ...apiTypeConfig.IPv6]
                break
            default:
                throw createError({ statusCode: 400, statusMessage: 'Invalid API type' })

        }
        const map = new Map<string, string>(apiBaseUrls.map(api => [api.id, api.url]))
        if (!map.has(backendID)) {
            throw createError({ statusCode: 400, statusMessage: 'Invalid backend ID' })
        }
        let apiBaseUrl = map.get(backendID)
        if (apiBaseUrl === undefined) {
            throw createError({ statusCode: 400, statusMessage: 'API base URL not found for the given backend ID' })
        }
        let data: any = {}

        if (apiBaseUrl.slice(-1) != '/') {
            apiBaseUrl = `${apiBaseUrl}/`
            }

        const queryString = new URLSearchParams(
            Object.entries(query).filter(([_, v]) => v !== undefined) as [string, string][]
        ).toString()

        data = await $fetch(`${apiBaseUrl}v1/${apiType}/${raw}${queryString ? '?' + queryString : ''}`, {
                method: 'GET',
                headers: authHeaders,
            }).catch((error: any) => {
                console.error(`Error fetching from ${apiBaseUrl}:`, error)
                const errStatus = error?.status ?? error?.statusCode
                if (errStatus) {
                    // 上游返回的错误响应（4xx/5xx），直接转发其状态码与错误体，不包装成 500
                    setResponseStatus(event, errStatus)
                    return error.data ?? {}
                }
                // 网络层错误（无法连接上游）
                setResponseStatus(event, 502)
                return { statusCode: 502, statusMessage: 'Backend unreachable' }
            })
        return data

        
    }

    })
