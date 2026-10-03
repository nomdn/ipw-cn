import { config } from '#lib/config/index.ts';

/**
 * 首屏「IPv4 / IPv6 / 优先级」三行的数据源（见 routes/+page.svelte 的 onMount）：
 * 三个接口分属三个独立源站，且都要等水合完成后才发起 —— 那时才做 DNS+TCP+TLS 太晚。
 * 把它们抽出来做 preconnect，让握手在 HTML 解析期就开始。
 * （这是旧站 nuxt.config.ts 里同一段逻辑的平移。）
 */
export const ipProbeOrigins: string[] = [
	config.DualStackAPI,
	config.v4OnlyAPI,
	config.v6OnlyAPI
].map((url) => new URL(url).origin);

/** 中间件（工具页 /middleware/* 的上游）：只在部分页面用到，DNS 预解析即可，不占连接。 */
export const middlewareOrigin: string | null = config.Middleware?.[0]
	? new URL(config.Middleware[0]).origin
	: null;
