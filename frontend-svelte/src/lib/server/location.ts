import { config } from '#lib/config/index.ts';
import { fetchMiddlewareServer, type ServerFetch } from '#lib/server/middleware.ts';

/**
 * IP 归属地文本聚合（`/ip` 与 `/ip/<ip>` 两个路由共用；对应旧站同名文件里那份实现）。
 * 输出是多行 text/plain，形如：
 *
 *   IP: 1.2.3.4
 *   bilibili: 中国 江苏 南京 电信
 *   geocn: 江苏省 南京市 江宁区 电信 宽带
 *   ...
 *
 * 单个数据源：按「国家 → 省/区域 → 城市 → 区县」拼位置字段，再补 ASN 与运营商/组织。
 * 后端未加载的库会返回 'not loaded'（或直接缺席），这类值不输出。
 */

/** 把单个数据源的归属地对象拼成可读字符串 */
function objectToText(obj: any): string {
	const parts: string[] = [];
	// 位置字段（按国家 → 省/区域 → 城市 → 区县）
	for (const k of ['country', 'administrative_area', 'region', 'city', 'district', 'province']) {
		const v = obj[k];
		if (typeof v === 'string' && v) parts.push(v);
	}
	// ASN 编号（如 "AS4134"）
	if (typeof obj.asn === 'string' && obj.asn) {
		parts.push(obj.asn.startsWith('AS') ? obj.asn : `AS${obj.asn}`);
	}
	// 运营商 / 组织 / 类型
	for (const k of ['isp', 'org', 'type']) {
		const v = obj[k];
		if (typeof v === 'string' && v) parts.push(v);
	}
	return [...new Set(parts)].join(' ');
}

/** 多数据源逐行输出（跳过后端未加载的库） */
export function formatLocationText(data: any, ip: string): string {
	const lines: string[] = [`IP: ${ip}`];
	const sources: Array<[string, any]> = [
		['bilibili', data?.bilibili],
		['geocn', data?.geocn],
		['ip2region', data?.ip2region],
		['ip2location', data?.ip2location],
		['maxmind_city', data?.maxmind_city],
		['maxmind_asn', data?.maxmind_asn],
		['dbip_city', data?.dbip_city],
		['dbip_asn', data?.dbip_asn],
		['qqwry', data?.qqwry]
	];
	for (const [name, value] of sources) {
		if (value === undefined || value === null) continue;
		if (typeof value === 'string') {
			// 纯字符串结果（error 等）原样输出，"not loaded" 跳过
			if (value && value !== 'not loaded') {
				lines.push(`${name}: ${value}`);
			}
			continue;
		}
		if (typeof value === 'object') {
			const text = objectToText(value);
			if (text) lines.push(`${name}: ${text}`);
		}
	}
	return lines.join('\n');
}

/**
 * 查询指定 IP 的归属地：IPLocationAPI 里的节点逐个试，任一成功即返回文本。
 * 全部失败时返回 `{ text, status }`，调用方以该状态码 + 文本响应（旧站同样是透传最后一次错误）。
 */
export async function locateText(
	fetchFn: ServerFetch,
	ip: string
): Promise<{ text: string; status: number }> {
	const backends = config.IPLocationAPI;
	if (!backends || backends.length === 0) {
		return { text: 'Error 500: No location backend configured', status: 500 };
	}

	let lastError: any = null;
	for (const backend of backends) {
		const path = `/middleware/${backend.id}/location/${ip}`;
		try {
			const res = await fetchMiddlewareServer<any>(fetchFn, path);
			// 成功：location 响应必含 ip 字段；缺失视为失败（如误返回错误体），换下一个后端
			if (res && typeof res === 'object' && res.ip) {
				return { text: formatLocationText(res, ip), status: 200 };
			}
			lastError = new Error('invalid location response');
		} catch (e: any) {
			lastError = e;
			console.error(
				`[ip] backend "${backend.label}" failed for ${ip}:`,
				e?.message || e
			);
		}
	}

	// 全部后端失败：透传最后一次错误（text/plain）
	const raw = lastError?.status ?? lastError?.statusCode ?? 502;
	const status = typeof raw === 'number' ? raw : 502;
	return { text: `Error ${status}: ${lastError?.message || 'Backend unreachable'}`, status };
}
