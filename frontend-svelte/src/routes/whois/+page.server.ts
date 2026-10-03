import { renderDoc } from '#lib/server/docs.ts';

/**
 * whois 页底部的说明块：内容来自 `markdown/whois.md`。
 * 旧站在页面里 `$fetch('/api/markdown/whois')` 取（`useAsyncData` 所以也是 SSR 出来的），
 * 这里直接在 load 里渲染好一起下发。
 * `/api/markdown/whois` 这条接口仍然保留（另有用途），两处走的是同一个 renderDoc()。
 */
export async function load() {
	return { doc: (await renderDoc('whois')) ?? '' };
}
