import { jsonResponse } from '#lib/server/http.ts';
import { renderDoc } from '#lib/server/docs.ts';

/**
 * GET /api/markdown/<path> → 渲染后的 HTML（对应旧站 `server/api/markdown/[...path].get.ts`）。
 *
 * 谁在用：`/doc/**` 页面已改为在 `+page.server.ts` 里直接调用 `renderDoc()`（SSR 一次到位，
 * 少一跳自请求）；这个接口保留下来是给 whois 页用的 —— 它在客户端按需取 `whois.md`
 * 渲染出来的说明块（旧站也是 `$fetch('/api/markdown/whois')`），同时它对外也是一个可取用的
 * 文档接口。返回体是 HTML 文本，与旧站 h3 直接 return string 的行为一致。
 */
export async function GET({ params }) {
	const path = (params.path ?? '').trim();

	if (!path) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Missing markdown path' }, 400);
	}
	// 防止目录穿越（与旧站同样的两个判断）
	if (path.includes('..') || path.includes('\\')) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Invalid path' }, 400);
	}

	const html = await renderDoc(path);
	if (html === null) {
		return jsonResponse({ statusCode: 404, statusMessage: 'Markdown not found' }, 404);
	}

	return new Response(html, {
		headers: { 'content-type': 'text/html; charset=utf-8' }
	});
}
