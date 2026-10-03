import { renderMarkdown } from '#lib/markdown.ts';

/**
 * 文档 Markdown 的取用层（对应旧站 `useStorage('assets:server')` 那一段）。
 *
 * 旧站的 markdown 放在 `server/assets/markdown/`，由 Nitro 的 `assets:server` 存储卷
 * 在**构建期**打包进产物；SvelteKit 没有等价的存储卷，这里改用 Vite 的
 * `import.meta.glob(..., { query: '?raw', eager: true })` 达成同样的效果：
 * 构建期把每个 .md 的原文取出来，编译成模块里的一个字符串常量。
 *
 * 为什么用 eager（而不是按需动态 import）：
 *   ① 一共 36 个文件、约 230KB 纯文本，全量内联对 Worker 体积可忽略；
 *   ② adapter-cloudflare 默认把 SSR 打成一个 `_worker.js`，动态 import 会被
 *      rollup 内联掉（inlineDynamicImports），eager 只是把这件事实写明；
 *   ③ 请求时不再有任何文件 IO —— 这正是旧站 assets 存储卷的行为。
 */

const rawSources = import.meta.glob('./markdown/**/*.md', {
	query: '?raw',
	import: 'default',
	eager: true
}) as Record<string, string>;

/** 建索引：'./markdown/doc/index.md' → 'doc/index' */
const sources = new Map<string, string>();
for (const [file, content] of Object.entries(rawSources)) {
	const key = file.replace(/^\.\/markdown\//, '').replace(/\.md$/, '');
	sources.set(key, content);
}

/** 该文档路径是否存在（不含扩展名，如 'doc/index'） */
export function hasDoc(path: string): boolean {
	return sources.has(path);
}

/**
 * 渲染指定文档为 HTML；路径不存在或非法时返回 null（调用方自行决定 404 还是别的）。
 * 目录穿越在这里也拦一道 —— 与旧站 `server/api/markdown/[...path].get.ts` 一致。
 */
export async function renderDoc(path: string): Promise<string | null> {
	if (!path || path.includes('..') || path.includes('\\')) return null;
	const markdown = sources.get(path);
	if (!markdown) return null;
	return renderMarkdown(markdown);
}
