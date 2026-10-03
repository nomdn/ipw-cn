import { checkOrigin, checkRate, textResponse, jsonResponse } from '#lib/server/http.ts';
import { locateText } from '#lib/server/location.ts';

/**
 * GET /ip/<ip> → text/plain 输出指定 IP 的归属地（支持 IPv4 / IPv6，如 /ip/2001:db8::1）。
 * 对应旧站 `server/routes/ip/[ip].get.ts`。
 *
 * 行为与 /ip 完全一致，只是 IP 来自路径参数而非查询串/请求方地址。
 */
export async function GET({ params, request, fetch, getClientAddress }) {
	const denied = checkOrigin(request) ?? checkRate(request, getClientAddress);
	if (denied) return denied;

	const ip = (params.ip ?? '').trim();
	if (!ip) {
		return jsonResponse({ statusCode: 400, statusMessage: 'Missing IP parameter' }, 400);
	}

	const { text, status } = await locateText(fetch, ip);
	return textResponse(text, status);
}
