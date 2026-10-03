import { buildCandidates } from '#lib/middleware.ts';

/**
 * 服务端版的中间件取数（用于 `/ip/*` 这类"由服务端自己发起"的查询）。
 *
 * 为什么不能直接用 `#lib/middleware.ts` 里的 `fetchMiddleware()`：
 * 那个是给浏览器用的，`fetch` 收到相对路径 `/middleware/...` 会自动补上当前站点 origin；
 * 而服务端（Cloudflare Worker / Node）里的全局 fetch 要求绝对 URL，不能直接吃相对路径。
 * 所以这里把「候选循环 + 重试判定」照搬一遍，把 `fetch` 作为参数传进来 ——
 * 传 SvelteKit 事件的 `event.fetch`，它支持相对路径（内部直接命中本站的
 * `/middleware/[...slug]` 路由，不发真实网络请求，与旧站 Nitro 内部 fetch 的行为一致）。
 */

/** 需要切换节点重试的上游状态码（与客户端同一套口径） */
const RETRY_STATUS_CODES = new Set([401, 403, 418, 502]);

/** SvelteKit 事件上那个支持相对路径的 fetch */
export type ServerFetch = (input: string, init?: RequestInit) => Promise<Response>;

/** 取错误里的 HTTP 状态码；没有状态码 = 网络层错误（可重试） */
function statusOf(e: unknown): number | undefined {
	return e && typeof e === 'object' && 'status' in e ? ((e as any).status as number) : undefined;
}

export class ServerFetchError extends Error {
	status?: number;
	constructor(message: string, status?: number) {
		super(message);
		this.status = status;
	}
}

/** 单次候选请求：非 2xx 抛 ServerFetchError（带 status）；解析出的内容按 JSON/文本嗅探 */
async function fetchOnce<T>(fetchFn: ServerFetch, url: string, init: RequestInit = {}): Promise<T> {
	let res: Response;
	try {
		res = await fetchFn(url, {
			...init,
			headers: { Accept: 'application/json, text/plain, */*', ...(init.headers as any) }
		});
	} catch (e) {
		// 连不上 / 超时：没有状态码 ⇒ 可重试
		throw new ServerFetchError(e instanceof Error ? e.message : String(e));
	}

	if (!res.ok) {
		throw new ServerFetchError(`Request failed with status ${res.status}`, res.status);
	}

	const text = await res.text();
	if (!text) return undefined as T;
	const ct = res.headers.get('content-type') ?? '';
	if (ct.includes('json') || text.startsWith('{') || text.startsWith('[')) {
		try {
			return JSON.parse(text) as T;
		} catch {
			return text as unknown as T;
		}
	}
	return text as unknown as T;
}

/** 依次尝试候选节点（外部中间件在前、本站内置中间件兜底），返回第一个成功的结果 */
export async function fetchMiddlewareServer<T = unknown>(
	fetchFn: ServerFetch,
	path: string,
	init: RequestInit = {}
): Promise<T> {
	const candidates = buildCandidates(path);
	let lastError: unknown = new ServerFetchError('all middleware candidates failed');

	for (const candidate of candidates) {
		try {
			return await fetchOnce<T>(fetchFn, candidate, init);
		} catch (e) {
			lastError = e;
			const status = statusOf(e);
			const retryable =
				status === undefined || (typeof status === 'number' && RETRY_STATUS_CODES.has(status));
			if (!retryable) throw e;
		}
	}

	throw lastError;
}
