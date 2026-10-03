import { config } from '#lib/config/index.ts';

/**
 * 服务端公共工具（只在 SvelteKit 的 +server.ts / hooks.server.ts 里用）。
 * 对应旧站 h3 事件处理里的那几个公共片段：Origin/Referer 校验、每 IP 限流、取客户端 IP。
 */

/** 站点自身 Origin（用于拒绝跨域浏览器调用） */
export function siteOrigin(): string {
	return new URL(config.siteUrl).origin;
}

/**
 * Origin / Referer 校验：跨域浏览器调用一律拒绝。
 * 注意：无 Origin/Referer 的服务端调用与命令行请求无法区分，仅靠限流约束（旧站同此口径）。
 * 返回 403 响应；通过则返回 null。
 */
export function checkOrigin(request: Request): Response | null {
	const origin = request.headers.get('origin');
	const referer = request.headers.get('referer');
	const allow = siteOrigin();

	if (origin && origin !== allow) {
		return jsonResponse({ statusCode: 403, statusMessage: 'Forbidden: cross-origin request' }, 403);
	}
	if (referer && referer !== allow && !referer.startsWith(allow + '/')) {
		return jsonResponse({ statusCode: 403, statusMessage: 'Forbidden: invalid referer' }, 403);
	}
	return null;
}

/** 取客户端 IP：优先 Cloudflare 的 CF-Connecting-IP，其次 SvelteKit 的 getClientAddress()（部署在 Worker 上时它读的也是同一来源） */
export function clientIP(request: Request, getClientAddress: () => string): string {
	const cf = request.headers.get('cf-connecting-ip');
	if (cf) return cf.trim();
	try {
		return getClientAddress();
	} catch {
		// getClientAddress() 在非服务器环境（如 prerender）会抛错
		return request.headers.get('x-forwarded-for')?.split(',')[0]?.trim() || 'unknown';
	}
}

// ---- 简单限流（每 IP 每分钟次数上限，进程内内存计数；limit 0 表示不限流） ----
// 与旧站一致：进程内 Map，不做持久化。Cloudflare Workers 每个 isolate 各持一份，
// 因此实际额度是「每 isolate × limit」——旧站在 Vercel/Node 上同样只是单实例粒度。
const rateLimitMap = new Map<string, { count: number; resetAt: number }>();

export function checkRateLimit(ip: string, limit: number): boolean {
	const now = Date.now();
	const entry = rateLimitMap.get(ip);
	if (!entry || entry.resetAt <= now) {
		rateLimitMap.set(ip, { count: 1, resetAt: now + 60_000 });
		// 顺带清理过期条目，避免 Map 无限增长
		if (rateLimitMap.size > 1000) {
			for (const [k, v] of rateLimitMap) {
				if (v.resetAt <= now) rateLimitMap.delete(k);
			}
		}
		return true;
	}
	entry.count++;
	return entry.count <= limit;
}

/** 限流中间件：超出返回 429；通过返回 null */
export function checkRate(request: Request, getClientAddress: () => string): Response | null {
	const limit = config.rateLimitPerMinute || 0;
	if (limit > 0 && !checkRateLimit(clientIP(request, getClientAddress), limit)) {
		return jsonResponse({ statusCode: 429, statusMessage: 'Too Many Requests' }, 429);
	}
	return null;
}

export function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json; charset=utf-8' }
	});
}

export function textResponse(body: string, status = 200): Response {
	return new Response(body, {
		status,
		headers: { 'content-type': 'text/plain; charset=utf-8' }
	});
}
