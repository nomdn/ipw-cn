import { config } from '#lib/config/index.ts';

/**
 * 首页的 curl 示例代码块：高亮在**服务端**算好，随 HTML 一起下发。
 *
 * 为什么放 SSR（旧站同一决定）：
 *   ① 客户端不必再下载/执行 shiki（引擎 + 语言 + 主题 ≈ 164KB）；
 *   ② 不会再有「先出纯文本兜底、若干秒后才换成高亮」的那次块高变化（可见重排）。
 * 这段代码是构建期常量（只依赖 config，不随请求变化），所以结果可以安全地随 HTML 一起缓存。
 *
 * 注意 `import` 写在 handler 内部：动态 import 会把它切成独立 chunk，
 * 只在真正要高亮时加载，不会像顶层静态 import 那样被塞进首屏的模块预加载列表。
 */
const code = `
# 请勿用于商业用途，仅供个人测试学习之用，请遵守中国法律法规
# 查询本机外网 IPv4 地址
curl ${config.v4OnlyAPI}

# 查询本机外网 IPv6 地址
curl ${config.v6OnlyAPI}

# 测试网络是 IPv4 还是 IPv6 访问优先
# (访问 IPv4/IPv6 双栈站点，如果返回 IPv6 地址，则 IPv6 访问优先)
curl ${config.DualStackAPI}
`.trim(); // 关键：去掉首尾多余的空行

export async function load() {
	try {
		const { highlightCode } = await import('#lib/shiki.ts');
		return { code, highlighted: await highlightCode(code, 'bash') };
	} catch (e) {
		// 高亮失败不是致命错误：模板会自动落到纯文本兜底分支（fallback）
		console.error('[home] shiki 高亮失败，改用纯文本兜底:', e);
		return { code, highlighted: '' };
	}
}
