import { fetchMiddleware } from '#lib/middleware.ts';

/**
 * 「同一件事问多个节点，用第一个成功的答案」——旧站每个单结果工具页
 * （location / asn / whois / ssl / ipv6webcheck）都手写了一遍的循环，这里收成一个函数。
 *
 * 两层重试的分工（与旧站一致，别把两层合并）：
 *   - **内层**（在 `fetchMiddleware` 里）：`config.Middleware` 外部节点 → 内置中间件兜底，
 *     解决的是"走哪条通道发出去"；
 *   - **外层**（本函数）：`IPLocationAPI` / `APIBaseURL.*` 里的拨测节点，
 *     解决的是"由哪台机器去拨测"。
 * 所以 URL 形如 `/middleware/<节点ID>/<接口>/<参数>`，节点 ID 由外层换、通道由内层换。
 *
 * 失败语义：全部节点都失败时返回 `{ ok:false }`，**由调用方决定最终给用户看什么**——
 * 旧站各页的兜底文案并不统一（"请检查网络或稍后重试" / "请检查域名或网络" /
 * "请检查IP或网络"），所以不在这里写死。
 */

export interface NodeRef {
	id: string;
	label?: string;
}

export interface NodePoolOptions {
	/** 切到下一个节点前回调，用于把「正在重试 X…」写进界面（旧站就是这么提示的） */
	onRetry?: (message: string) => void;
	/** 单次请求超时（毫秒）。不传则不设超时，与旧站一致 */
	timeoutMs?: number;
}

export type NodePoolOutcome<T> = { ok: true; data: T } | { ok: false; error: string };

export async function queryNodePool<T>(
	nodes: readonly NodeRef[],
	pathFor: (nodeId: string) => string,
	opts: NodePoolOptions = {}
): Promise<NodePoolOutcome<T>> {
	let lastError: unknown = null;

	for (let i = 0; i < nodes.length; i++) {
		const node = nodes[i];
		if (!node) continue;
		try {
			const data = await fetchMiddleware<T>(pathFor(node.id), { timeoutMs: opts.timeoutMs });
			return { ok: true, data };
		} catch (e: any) {
			lastError = e;
			const next = nodes[i + 1];
			if (next) {
				// 中间失败：把「…，正在重试 <下一个节点>…」显示出来（旧站同款提示）
				opts.onRetry?.(`${e?.message || '请求失败'}，正在重试 ${next.label || ''}...`);
			}
		}
	}

	return { ok: false, error: lastError instanceof Error ? lastError.message : String(lastError ?? '') };
}

/** 「每个节点各问一次、同时进行、谁先回来谁先上屏」的行状态 */
export interface NodeRow<D> {
	label: string;
	loading: boolean;
	error?: string;
	data?: D;
}

/**
 * 并发问所有节点（dns / dnssec / tcping / speed 这些多节点对比页用）。
 *
 * 与 `queryNodePool()` 的区别：那个是"逐节点兜底、只要一个成功结果"，
 * 这个是"每个节点都要一份结果"。重试仍然由每行自己的 `fetchMiddleware` 承担
 * （即每行都会先在外部中间件与内置中间件之间选通道）。
 *
 * 关于"连续点击"：调用方每次重新点击都会**换一个新的 rows 数组**（重新赋值 $state），
 * 于是上一次查询里那些尚未落地的 promise 拿着的是**旧数组**的引用，
 * 它们的写入不会再影响界面 —— 效果等同于旧站的 queryId 守卫，但不用维护计数器。
 *
 * @param rows 与 nodes 一一对应、且已置 loading=true 的响应式行数组
 * @param errorMessage 把异常转成该页要展示的错误文案（各页文案不同）
 */
export async function queryAllNodes<D>(
	nodes: readonly NodeRef[],
	pathFor: (nodeId: string, index: number) => string,
	rows: NodeRow<D>[],
	errorMessage: (e: unknown) => string
): Promise<void> {
	await Promise.all(
		nodes.map(async (node, index) => {
			const row = rows[index];
			if (!row) return;
			try {
				row.data = await fetchMiddleware<D>(pathFor(node.id, index));
			} catch (e) {
				row.error = errorMessage(e);
			} finally {
				row.loading = false;
			}
		})
	);
}
