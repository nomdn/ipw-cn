import { redirect } from '@sveltejs/kit';

/**
 * /ipv6 → /location 的兼容跳转（旧站 app/pages/ipv6.vue）。
 *
 * 旧站是在 onMounted 里 `navigateTo()`，也就是先渲染一屏"正在跳转…"再由客户端跳走。
 * 这里改成服务端 302：无 JS 也能跳、少一次白屏、SEO 上也不会把这条 URL 当成有效页收录。
 * `?ip=` 参数照旧带过去（缺失时不拼出 `?ip=undefined`，这是旧站特意修过的一个点）。
 */
export function load({ url }) {
	const ip = url.searchParams.get('ip');
	redirect(302, ip ? `/location?ip=${encodeURIComponent(ip)}` : '/location');
}
