import { isIPv6 } from 'is-ip';
import { config } from '#lib/config/index.ts';

/**
 * 访客公网 IP（走 config.DualStackAPI 的双栈探测）。
 *
 * 旧站每个工具页都在自己的 onMounted 里 `$fetch(config.DualStackAPI)` 取一次，
 * 7 个页面就是 7 份重复代码、切页还会各取一次。这里收成一个模块级单例：
 *   - 模块**只在工具页被 import**（文档页不引入），所以不会给文档页多加请求；
 *   - 首次 import 时（仅浏览器端）发一次请求，之后各页共用同一结果；
 *   - 用 `$state` 承载，模板里读到就会随结果到达自动更新。
 *
 * SSR 期返回空串 —— 与旧站一致（旧站也是挂载后才取，首屏 HTML 里没有访客 IP）。
 */
let ip = $state('');

if (typeof window !== 'undefined') {
	fetch(config.DualStackAPI)
		.then((r) => r.text())
		.then((text) => {
			ip = text.trim();
		})
		.catch(() => {
			// 取不到就保持空串；调用方按空串渲染（旧站同样只是把错误吞掉）
		});
}

/** 当前访客 IP（未取到时为空串） */
export function visitorIP(): string {
	return ip;
}

/** 访客网络是否为 IPv6 优先（IP 未取到时为 false，与旧站 `isIPv6('')` 一致） */
export function visitorIsIPv6(): boolean {
	return isIPv6(ip);
}
