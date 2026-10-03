import { error } from '@sveltejs/kit';
import { getDocMeta } from '#lib/config/doc.ts';
import { renderDoc } from '#lib/server/docs.ts';

/**
 * 文档首页 /doc（内容来自 `markdown/doc/index.md`）。
 * 旧站在页面里 `$fetch('/api/markdown/doc/index')`，这里直接在 load 里渲染 ——
 * 少一次自请求，且 HTML 一定跟着页面一起出来（不会被 CDN 缓存成"空壳 + 稍后补"）。
 */
export async function load() {
	const html = await renderDoc('doc/index');
	if (html === null) {
		error(404, 'Markdown not found');
	}
	return { html, meta: getDocMeta('/doc') };
}
