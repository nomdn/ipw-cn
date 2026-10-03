/**
 * 从 frontend-ssr 的 @element-plus/icons-vue 编译产物里抽出图标几何数据，
 * 生成 Svelte 组件 —— 保证新前端用的图标与旧站**逐像素同源**（同一版本同一份路径）。
 *
 * 为什么要这么绕：icons-vue 是 Vue 专用包，dist 里没有原始 .svg，只有
 * `_createElementVNode("path", { d: "..." })` 形式的编译产物。好在这层包装很薄，
 * 把 name → 图元列表解出来即可；比去官网重新下载、或手抄路径都可靠。
 */
import fs from 'node:fs';
import path from 'node:path';

const SRC = 'F:/code/ipw-cn/frontend-ssr/node_modules/@element-plus/icons-vue/dist/index.js';
const OUT = 'F:/code/ipw-cn/frontend-svelte/src/lib/components/icons';

const WANT = [
	'Moon',
	'Sunny',
	'Expand',
	'CircleCheckFilled',
	'CircleCloseFilled',
	'InfoFilled',
	'Position',
	'Loading',
	'ArrowDown',
	'ArrowUp',
	'ArrowRight',
	'Close',
	'Check',
	'Minus',
	'SemiSelect'
];

const src = fs.readFileSync(SRC, 'utf8');

// 每个图标形如：  name: "Moon",\n  __name: "moon",\n  setup(__props) {\n ... _createElementVNode("path", {\n ... d: "..."\n
// 用 `name: "X"` 作为切分点，截到下一个 `name: "` 之前。
const names = [...src.matchAll(/name: "([A-Za-z0-9]+)",\n\s+__name: "([a-z0-9-]+)"/g)];
const blocks = new Map();
for (let i = 0; i < names.length; i++) {
	const start = names[i].index;
	const end = i + 1 < names.length ? names[i + 1].index : src.length;
	blocks.set(names[i][1], src.slice(start, end));
}

/** 把某个图标块里的 path / circle / ellipse / rect 图元抽成 JSX 属性字符串 */
function primitives(block) {
	const out = [];
	// path：d 属性（可能被换行/空白打断）
	for (const m of block.matchAll(/_createElementVNode\d*\("path",\s*\{([\s\S]*?)\}\s*\)/g)) {
		const body = m[1];
		const d = /d:\s*"([\s\S]*?)"\s*$|d:\s*"([\s\S]*?)",/m.exec(body);
		const dval = (d?.[1] ?? d?.[2] ?? '').replace(/\s+/g, ' ').trim();
		if (dval) out.push({ tag: 'path', attrs: `d="${dval}"` });
	}
	// circle / ellipse / rect：其余图元（EP 少数图标用得到）
	for (const m of block.matchAll(/_createElementVNode\d*\("(circle|ellipse|rect)",\s*\{([\s\S]*?)\}\s*\)/g)) {
		const tag = m[1];
		const attrs = [];
		for (const a of m[2].matchAll(/(\w[\w-]*):\s*"?([^",\n]+)"?/g)) {
			const k = a[1];
			const v = a[2].trim();
			if (k === 'fill') continue;
			attrs.push(`${k === 'className' ? 'class' : k}="${v}"`);
		}
		out.push({ tag, attrs: attrs.join(' ') });
	}
	return out;
}

fs.mkdirSync(OUT, { recursive: true });
const report = [];

for (const name of WANT) {
	const block = blocks.get(name);
	if (!block) {
		report.push(`${name}: ❌ 在包内未找到`);
		continue;
	}
	const prims = primitives(block);
	if (!prims.length) {
		report.push(`${name}: ❌ 未解析出图元`);
		continue;
	}
	const inner = prims.map((p) => `\t<${p.tag} ${p.attrs} />`).join('\n');
	const file = path.join(OUT, `${name}.svelte`);
	fs.writeFileSync(
		file,
		`<script lang="ts">
	/**
	 * Element Plus 图标 \`${name}\`（MIT）。
	 *
	 * 几何数据抽自 frontend-ssr 依赖的 @element-plus/icons-vue@2.3.2 编译产物，
	 * 与旧站用的是同一份路径 —— 迁移后图标观感不发生任何变化。
	 * 需要新图标时把名字加进 WANT，再跑 scripts/extract-element-plus-icons.mjs，别手抄路径。
	 */
	import type { SVGAttributes } from 'svelte/elements';

	let { class: className = '', ...rest }: SVGAttributes<SVGSVGElement> = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 1024 1024"
	fill="currentColor"
	class={className}
	aria-hidden="true"
	{...rest}
>
${inner}
</svg>
`,
		'utf8'
	);
	report.push(`${name}: ✅ ${prims.length} 个图元 -> ${path.basename(file)}`);
}

console.log(report.join('\n'));
