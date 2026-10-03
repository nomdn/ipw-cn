import adapter from '@sveltejs/adapter-cloudflare';
import { sveltekit } from '@sveltejs/kit/vite';
import type { Config } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';
import { readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { config } from './src/lib/config/index.ts';

// Kit 的 CSP source 类型是个模板字面量联合（`https://x.y` / 'self' / `sha384-…` …），
// 但它所在的 `Csp` 命名空间只声明不导出（`declares 'Csp' locally, but it is not exported`）。
// 这里从导出的 `Config.csp.directives` 反推出同一个类型，避免自己手抄一份联合。
type CspSource = NonNullable<NonNullable<NonNullable<Config['csp']>['directives']>['connect-src']>[number];

/**
 * 把配置对象里出现过的所有 https:// 域名抽出来（旧站 nuxt.config.ts 里的同款做法）。
 * 这些域名要进 CSP 的 script-src / connect-src：umami 上报、三个 IP 探测源站、
 * 各拨测节点等等。**新增外部服务时不用手动维护 CSP 白名单**，写进 config 即可。
 *
 * 返回类型标成 `Csp.Source[]`：Kit 的 csp.directives 收的是模板字面量联合类型
 * （`${string}://${string}.${string}` 之类），不是裸 `string[]`，否则 svelte-check 会报
 * "Type 'string' is not assignable to type 'Source'"。
 */
const extractDomains = (obj: unknown): CspSource[] => {
	const urls = JSON.stringify(obj).match(/https?:\/\/[^"\/\\\s]+/g) || [];
	return [...new Set(urls.map((u) => new URL(u).origin as CspSource))];
};

const allowedDomains = extractDomains(config);

/**
 * app.html 里的**内联 <script>** 必须自己往 CSP 里领一个 hash。
 *
 * Kit 只给「组件渲染出来的内联脚本」算 hash（内部走
 * `csp.add_script_hashes(rendered.hashes.script)`，来自 Svelte SSR 的产物），
 * 而 app.html 是纯模板字符串、不经过组件渲染 —— 于是：
 *   - nonce 模式（SSR 页）：Kit 会往 app.html 的 <script> 上补 nonce，没问题；
 *   - hash 模式（预渲染页，也就是 /doc/**）：**没有任何 nonce/hash**，会被直接拦掉。
 * script-src 里还有 'strict-dynamic'，按 CSP3 宿主白名单（含 'self'）一律失效，
 * 只有 nonce/hash 能让内联脚本生效 —— 所以这条不是"可选优化"，
 * 少了它 app.html 里那段「首屏防闪」深色模式脚本在全部 /doc/** 页都不执行
 * （深色用户进文档站会先白闪一下，而且类永远补不回来）。
 *
 * 构建期从 app.html 现算，脚本内容改了 hash 自动跟着变，不需要人工同步。
 */
const extractInlineScriptHashes = (html: string): CspSource[] => {
	const hashes: CspSource[] = [];
	// 只取**内联**的：带 src 的 <script> 不需要 hash
	for (const match of html.matchAll(/<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/gi)) {
		const content = match[1] ?? '';
		if (!content.trim()) continue;
		hashes.push(
			`sha256-${createHash('sha256').update(content, 'utf8').digest('base64')}` as CspSource
		);
	}
	return hashes;
};

const appHtmlScriptHashes = extractInlineScriptHashes(
	readFileSync(new URL('./src/app.html', import.meta.url), 'utf8')
);

export default defineConfig({
	plugins: [
		// Tailwind v4 走 Vite 插件形态（不再需要 postcss 配置与 tailwind.config.js）。
		// 必须排在 sveltekit() 前面：SvelteKit 要在编译期就看到已展开的 Tailwind 指令，
		// 否则 `@import "tailwindcss"` 会被当成普通 css 原样透传。
		tailwindcss(),
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// 注意：SvelteKit 3 已移除 `$lib`，统一走 package.json 的 `imports` 子路径
			// （`#lib` / `#lib/*`）。**不要**再补 `alias: { $lib: 'src/lib' }`：
			// 该选项已标记 deprecated（构建时会告警），且会让生成出来的 tsconfig
			// 覆盖 `paths`，`$lib` 反而拿不到类型检查。
			// shadcn-svelte 的别名因此也配成 `#lib/...`（见 components.json）。

			// 部署目标与 frontend-ssr 一致：Cloudflare Workers。
			// adapter-cloudflare 产出 .svelte-kit/cloudflare/_worker.js 与静态资源目录，
			// 再由 wrangler.jsonc 的 assets 绑定分发。
			adapter: adapter(),

			// 内容安全策略：用 Kit 内建的 CSP 而不是 nuxt-security。
			// kit 会自己把 nonce（SSR 页）/ hash（预渲染页）拼进 script-src —— 见下方 mode。
			// 指令口径与旧站 nuxt-security 的配置逐条对齐：
			//   - script-src 保留 'strict-dynamic'：允许被信任脚本动态插入自己的子脚本
			//     （Nuxt/Kit 的 hydration、我们的 umami 注入都靠这条）；
			//   - 'wasm-unsafe-eval' 给 shiki 的 WASM 正则引擎（迁移后仍走 JS 引擎，
			//     但留着这条以免将来换 oniguruma-wasm 时被拦）；
			//   - connect-src 里的外部域名供 umami 上报与三个 IP 探测源站使用。
			// 注意 'strict-dynamic' 存在时，宿主白名单（下面的全部域名）在支持 CSP3 的
			// 浏览器里会被忽略 —— 这是旧站就有的既有取舍，保持一致。
			csp: {
				// auto：预渲染页面（/doc/**，SSG）用 hash，其余动态页面用逐请求 nonce。
				mode: 'auto',
				directives: {
					'script-src': [
						'self',
						'strict-dynamic',
						'wasm-unsafe-eval',
						// app.html 里的内联脚本（构建期现算，见 extractInlineScriptHashes）
						...appHtmlScriptHashes,
						...allowedDomains
					],
					'connect-src': ['self', ...allowedDomains],
					'style-src': ['self', 'https:', 'unsafe-inline'],
					'img-src': ['self', 'https://s0.wp.com', 'data:', 'https:'],
					'font-src': ['self', 'https:', 'data:']
				}
			},

			// 预渲染：/doc/** 走 SSG（见 src/routes/doc/+layout.ts）。
			// 默认 entries 是 ['*']，会自己顺着页面里的 <a href> 爬（DocMenu 把 32 条
			// 文档链接都渲染进 HTML，所以整棵子树都能被抓到）。
			// handleMissingId 保持默认的 'fail'：任何 #锚点 对不上都直接中断构建。
			// 首次开启时确实报了 16 处对不上（历史手写目录的问题，旧站同样存在但没人校验），
			// 已逐条修正/清理，现在它是防止目录再跑偏的守卫 —— 别改成 'warn' 把守卫关掉。
			prerender: {
				entries: ['*']
			}
		})
	]
});
