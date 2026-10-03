// SvelteKit 3 把 `Handle`（以及 `sequence`）从包根挪到了 `@sveltejs/kit/hooks`。
// 从 '@sveltejs/kit' 导入会在 svelte-check 里报 "has no exported member 'Handle'"。
import type { Handle } from '@sveltejs/kit/hooks';

/**
 * 安全响应头（CSP 由 vite.config.ts 的 `csp` 配置负责注入，这里只补其余几条，
 * 口径对齐旧站 nuxt-security 的默认值）。
 *
 * 之所以 CSP 交给 Kit 内建能力：它知道每个页面里到底有哪些内联脚本，
 * 能自动给出 nonce（SSR）或 hash（预渲染），不需要自己拼。
 */
const SECURITY_HEADERS: Record<string, string> = {
	'x-content-type-options': 'nosniff',
	'x-frame-options': 'SAMEORIGIN',
	'referrer-policy': 'strict-origin-when-cross-origin',
	'permissions-policy': 'geolocation=(), microphone=(), camera=()',
	'cross-origin-opener-policy': 'same-origin'
};

export const handle: Handle = async ({ event, resolve }) => {
	const response = await resolve(event);

	for (const [key, value] of Object.entries(SECURITY_HEADERS)) {
		response.headers.set(key, value);
	}

	return response;
};
