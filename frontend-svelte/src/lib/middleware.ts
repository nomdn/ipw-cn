import { config } from '#lib/config/index.ts';

/**
 * 中间件取数层（替代 Nuxt 版的 `utils/middleware.ts` 里的 `useMiddlewareFetch`）。
 *
 * 候选策略与旧站完全一致：
 *   - `config.Middleware` 里的外部节点（base URL + 相对路径）在前；
 *   - 站点自带的转发路由（就是传进来的相对路径本身，如 `/middleware/...`，
 *     由 `src/routes/middleware/[...slug]/+server.ts` 处理）放最后一位兜底；
 *   - `config.EnableInternalMiddleware === false` 时，候选列表不含自带中间件。
 *
 * 重试策略（旧站的语义，逐条照搬）：
 *   - 只切节点重试两类失败：① 网络层错误（拿不到 HTTP 状态码）；
 *     ② 上游返回 401 / 403 / 418 / 502 —— 换一个节点往往就能恢复；
 *   - 其它状态码（400 / 404 / 500 ...）直接抛错，不浪费候选；
 *   - **同一个候选只打一次**。旧版这么做是为了绕开 ofetch 的隐式重试
 *     （ofetch 对 GET 默认重试 1 次、且把"连不上"按 500 计，会让最贵的失败
 *     在同一个节点上无退避地重打一遍）；这里用原生 fetch，本来就没有隐式重试，
 *     但这条纪律仍然保留：重试统一由本循环承担。
 */

/** 需要切换节点重试的上游状态码 */
const RETRY_STATUS_CODES = new Set([401, 403, 418, 502]);

/** 构造中间件候选 URL 数组（导出出来便于单测 / 调试） */
export function buildCandidates(path: string): string[] {
	const external: string[] = Array.isArray(config.Middleware)
		? config.Middleware.filter((u): u is string => !!u)
		: [];
	const useInternal = config.EnableInternalMiddleware !== false;
	if (external.length === 0) return useInternal ? [path] : [];
	const candidates = external.map((u) => u.replace(/\/$/, '') + path);
	if (useInternal) candidates.push(path);
	return candidates;
}

/** 带 HTTP 状态的取数错误（对齐 ofetch 的 FetchError：error.status） */
export class MiddlewareError extends Error {
	status?: number;
	constructor(message: string, status?: number) {
		super(message);
		this.name = 'MiddlewareError';
		this.status = status;
	}
}

export interface MiddlewareFetchOptions extends RequestInit {
	/** 超时（毫秒）。不传则完全交给浏览器/上游，与旧站行为一致（旧站也没设超时）。 */
	timeoutMs?: number;
	/** 调用方的取消信号（与 timeoutMs 合并，任一触发即中止） */
	signal?: AbortSignal;
}

/** 取错误里的 HTTP 状态码；没有状态码 = 网络层错误（可重试） */
function getErrorStatus(e: unknown): number | undefined {
	return e instanceof MiddlewareError ? e.status : undefined;
}

/** 一次候选请求；失败一律抛出 MiddlewareError（带 status，网络层错误则无 status） */
async function fetchOnce<T>(url: string, options: MiddlewareFetchOptions): Promise<T> {
	const { timeoutMs, signal, headers, ...rest } = options;

	const controller = new AbortController();
	const signals: AbortSignal[] = [controller.signal];
	if (signal) signals.push(signal);
	if (timeoutMs && timeoutMs > 0) signals.push(AbortSignal.timeout(timeoutMs));
	// 合并多个信号：任意一个 abort 都能中止本次请求
	const merged = signals.length === 1 ? controller.signal : AbortSignal.any(signals);

	try {
		const res = await fetch(url, {
			...rest,
			signal: merged,
			headers: { Accept: 'application/json, text/plain, */*', ...headers }
		});
		if (!res.ok) {
			throw new MiddlewareError(`Request failed with status ${res.status}`, res.status);
		}
		const text = await res.text();
		if (!text) return undefined as T;
		// 上游有的是 JSON、有的是纯文本（例如 /ip 返回裸 IP），按内容嗅探
		const ct = res.headers.get('content-type') ?? '';
		if (ct.includes('json') || text.startsWith('{') || text.startsWith('[')) {
			try {
				return JSON.parse(text) as T;
			} catch {
				return text as unknown as T;
			}
		}
		return text as unknown as T;
	} catch (e) {
		if (e instanceof MiddlewareError) throw e;
		// fetch 抛出的 TypeError（连不上）/ AbortError（超时）都没有状态码 ⇒ 可重试
		throw new MiddlewareError(e instanceof Error ? e.message : String(e));
	} finally {
		controller.abort(); // 释放监听，避免超时计时器挂着
	}
}

/**
 * 依次尝试候选节点，返回第一个成功的结果。
 * 全部失败时抛出最后一个错误（与原 `useMiddlewareFetch` 的 `error.value` 一致）。
 */
export async function fetchMiddleware<T = unknown>(
	path: string,
	options: MiddlewareFetchOptions = {}
): Promise<T> {
	const candidates = buildCandidates(path);
	let lastError: unknown = new MiddlewareError('all middleware candidates failed');

	for (const candidate of candidates) {
		try {
			return await fetchOnce<T>(candidate, options);
		} catch (e) {
			lastError = e;
			const status = getErrorStatus(e);
			// 不可重试的错误（如 400 / 500）直接暴露，不浪费候选
			const retryable =
				status === undefined || (typeof status === 'number' && RETRY_STATUS_CODES.has(status));
			if (!retryable) throw e;
		}
	}

	throw lastError;
}

/**
 * 只走站点自带中间件（不轮候选节点）。
 * 用于 `/ip/*` 这类"必须由本机出口发起"的查询 —— 它们不是拨测，不需要多节点。
 */
export async function fetchInternal<T = unknown>(
	path: string,
	options: MiddlewareFetchOptions = {}
): Promise<T> {
	return fetchOnce<T>(path, options);
}
