import { checkOrigin, checkRate, clientIP, textResponse, jsonResponse } from '#lib/server/http.ts';
import { locateText } from '#lib/server/location.ts';

/**
 * GET /ip → text/plain 输出 IP 归属地（`?ip=<ip>` 优先，否则查请求方 IP）。
 * 对应旧站 `server/routes/ip/index.get.ts`。
 *
 * 注意：这里**强制** text/plain（不是 JSON），旧站如此，也是这个接口对外承诺的格式；
 * 出错时同样用 text/plain 输出 `Error <status>: <message>`，只有参数类错误（403/429/400）
 * 才回 JSON（对齐旧站 createError 的行为）。
 */
export async function GET({ url, request, fetch, getClientAddress }) {
	// 1) Origin / Referer 校验：跨域浏览器调用一律拒绝
	const denied = checkOrigin(request) ?? checkRate(request, getClientAddress);
	if (denied) return denied;

	// 2) 确定要查询的 IP：?ip=<ip> 优先，否则查请求方 IP
	let ip = (url.searchParams.get('ip') ?? '').trim();
	if (!ip) {
		ip = clientIP(request, getClientAddress);
	}
	ip = ip.trim();
	if (!ip || ip === 'unknown') {
		return jsonResponse({ statusCode: 400, statusMessage: 'Unable to determine IP address' }, 400);
	}

	const { text, status } = await locateText(fetch, ip);
	return textResponse(text, status);
}
