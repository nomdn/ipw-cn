// 环境变量：Kit 3 里 `$env/dynamic/private` 已废弃（dev 下会打 deprecated 警告），
// 替代品是 `$app/env/private`；被 `src/env.ts` 声明过的键以**具名导出**的形式暴露。
import { API_KEYS, APIKEYS } from '$app/env/private';
import { config } from '#lib/config/index.ts';
import { checkOrigin, checkRate, jsonResponse } from '#lib/server/http.ts';

/**
 * 站点自带的中间件转发入口（对应旧站 server/routes/middleware/[...slug].get.ts）。
 *
 * 它既是「兜底候选」——`config.Middleware` 里的外部节点全部失败时，前端候选循环会落到
 * 相对路径 `/middleware/...`，也就是这里；也是没有独立中间件时唯一的转发通道。
 *
 * 与旧站逐条对齐的行为：
 *   1. 跨域浏览器调用（Origin/Referer 不属于本站）一律 403；
 *   2. 每 IP 每分钟限流（config.rateLimitPerMinute，0 = 不限流）→ 429；
 *   3. slug 形如 <backendID>/<apiType>/<raw>；raw 里带协议（https://…）时按 / 拆出的
 *      分段数会超过 4，需要把协议之后的部分重新拼回去；
 *   4. 阻断 raw 里的 `.` / `..` 分段 —— 否则上游 URL 规范化会把请求（连同节点的
 *      Authorization 头）带到 `/v1` 之外的任意路径；
 *   5. 按 apiType 选择节点池（whois/ssl/detail → DualStack；dns/dnssec → 三栈；
 *      location/asn → IPLocationAPI；tcping/udping/speed → 三栈平铺）；
 *   6. 按 backendID 从 API_KEYS（JSON 字符串）里取 token，注入 `Authorization: Bearer`；
 *      **不透传 Origin**，避免上游 CORS 误判；
 *   7. 上游返回 4xx/5xx 时原样透传状态码与响应体；网络层失败（连不上）才报 502。
 *
 * 数据上报（POST /report、X-Boce-Reporter 归属头）已在旧站移除，这里同样不实现。
 */

/** tcping / speed 这类拨测上游本身较慢，给更长超时 */
const TIMEOUT_NORMAL_MS = 15_000;
const TIMEOUT_SLOW_MS = 30_000;

// url 可以为空/缺省：config 里「只经独立中间件转发」的节点就是这样写的
// （如 DualStack 的 anyang-cu，没有 url 字段）。这类节点命中后走下面的 502 分支。
type NodeEntry = { id: string; url?: string; label?: string };

function poolFor(apiType: string): NodeEntry[] {
	switch (apiType) {
		case 'whois':
		case 'ssl':
		case 'detail':
			return [...config.APIBaseURL.DualStack];
		case 'dns':
		case 'dnssec':
			return [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4, ...config.APIBaseURL.IPv6];
		case 'location':
		case 'asn':
			return [...config.IPLocationAPI];
		case 'tcping':
		case 'udping':
		case 'speed':
			return [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4, ...config.APIBaseURL.IPv6];
		default:
			return [];
	}
}

/** apiKeys 从环境变量读（JSON：{ "<backendID>": "<token>" }）。
 *  变量必须在 src/env.ts 里声明过，否则构建期会被判定为 undefined（见那个文件的注释）。
 *  API_KEYS 与历史别名 APIKEYS 二选一，前者优先。 */
function apiKeyFor(backendID: string): string | undefined {
	const raw = API_KEYS || APIKEYS || '';
	if (!raw) return undefined;
	try {
		const map = JSON.parse(raw) as Record<string, string>;
		return map[backendID];
	} catch {
		console.error('[middleware] API_KEYS 不是合法 JSON，已忽略');
		return undefined;
	}
}

export async function GET({ params, request, url, getClientAddress }) {
	const denied = checkOrigin(request) ?? checkRate(request, getClientAddress);
	if (denied) return denied;

	const slugString = params.slug;
	if (!slugString) return jsonResponse({ statusCode: 400, statusMessage: 'Missing slug parameter' }, 400);

	let slug = slugString.split('/').filter((s) => s.length > 0);
	if (slug.length === 0) return jsonResponse({ statusCode: 400, statusMessage: 'Invalid slug' }, 400);

	// raw 里带协议时（https://example.com）按 / 拆会多出分段，重新拼回
	if (slug.length > 4) {
		const [backendID, apiType, protocol, ...restParts] = slug;
		const rest = restParts.join('/');
		if (protocol === 'https:' || protocol === 'http:') {
			slug = [backendID!, apiType!, `${protocol}//${rest}`];
		} else {
			return jsonResponse({ statusCode: 400, statusMessage: 'Invalid slug' }, 400);
		}
	}

	const [backendID, apiType, ...rawParts] = slug;
	const raw = rawParts.join('/');
	if (!backendID || !apiType || !raw) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Missing parameters in slug' }, 400);
	}

	// 阻断路径穿越
	if (raw.split('/').some((seg) => seg === '.' || seg === '..')) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Invalid path' }, 400);
	}

	const nodes = poolFor(apiType);
	if (nodes.length === 0) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Invalid API type' }, 400);
	}

	const node = nodes.find((n) => n.id === backendID);
	if (!node) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Invalid backend ID' }, 400);
	}
	// url 为空的节点无法转发（会变成对本站自身的请求），直接 502
	if (!node.url) {
		return jsonResponse({ statusCode: 502, statusMessage: 'Backend URL not configured' }, 502);
	}

	const base = node.url.endsWith('/') ? node.url : `${node.url}/`;
	const isSlow = apiType === 'tcping' || apiType === 'udping' || apiType === 'speed';
	const upstream = `${base}v1/${apiType}/${raw}${url.search}`;

	const authHeaders: Record<string, string> = { Accept: 'application/json, text/plain, */*' };
	const apiKey = apiKeyFor(backendID);
	if (apiKey) authHeaders['Authorization'] = `Bearer ${apiKey}`;

	try {
		const res = await fetch(upstream, {
			method: 'GET',
			headers: authHeaders,
			signal: AbortSignal.timeout(isSlow ? TIMEOUT_SLOW_MS : TIMEOUT_NORMAL_MS)
		});
		const body = await res.text();
		// 原样透传上游状态码与响应体（不要包装成 500）
		return new Response(body, {
			status: res.status,
			headers: {
				'content-type': res.headers.get('content-type') ?? 'application/json; charset=utf-8'
			}
		});
	} catch (e) {
		console.error(`[middleware] 上游不可达 ${upstream}:`, e);
		return jsonResponse({ statusCode: 502, statusMessage: 'Backend unreachable' }, 502);
	}
}
