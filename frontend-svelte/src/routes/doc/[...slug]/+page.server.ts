import { error } from '@sveltejs/kit';
import { getDocMeta } from '#lib/config/doc.ts';
import { renderDoc } from '#lib/server/docs.ts';

/**
 * 文档详情页 /doc/<slug>（如 /doc/user/enable_ipv6 → `markdown/doc/user/enable_ipv6.md`）。
 *
 * 与旧站 `app/pages/doc/[...slug].vue` 一致：文档路径就是 URL 路径（去掉开头斜杠），
 * 直接映射到 markdown 文件；meta 从 config/doc.ts 的映射表取。
 */
export async function load({ params }) {
	const slug = params.slug ?? '';
	const path = `doc/${slug}`;

	const html = await renderDoc(path);
	if (html === null) {
		error(404, `文档不存在: ${slug}`);
	}

	return { html, meta: getDocMeta(`/doc/${slug}`) };
}
