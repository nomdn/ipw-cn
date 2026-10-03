import { renderDoc } from '#lib/server/docs.ts';

/** dns 页底部的说明块：内容来自 `markdown/dns.md`（旧站是 `$fetch('/api/markdown/dns')`）。 */
export async function load() {
	return { doc: (await renderDoc('dns')) ?? '' };
}
