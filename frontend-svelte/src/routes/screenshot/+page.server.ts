import { renderDoc } from '#lib/server/docs.ts';

/** screenshot 页的说明块：内容来自 `markdown/screenshot.md`。 */
export async function load() {
	return { doc: (await renderDoc('screenshot')) ?? '' };
}
