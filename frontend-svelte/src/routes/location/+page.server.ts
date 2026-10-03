import { config } from '#lib/config/index.ts';

/**
 * location 页的 curl 示例代码块：高亮在**服务端**算好，随 HTML 一起下发。
 * 与首页 `+page.server.ts` 同一套做法（原因见首页那份注释）。
 *
 * 唯一与首页不同的地方：这里的模板**没有 fallback 分支**，所以默认值给 `code`
 * ——高亮失败时正好退回原始代码文本，代码块不会空着。
 * （首页那边模板有 v-if/v-else 两条分支，默认值必须留空，两处不可照抄。）
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
`.trim();

export async function load() {
	try {
		const { highlightCode } = await import('#lib/shiki.ts');
		return { code, highlighted: await highlightCode(code, 'bash') };
	} catch (e) {
		console.error('[location] shiki 高亮失败，改用纯文本兜底:', e);
		return { code, highlighted: code };
	}
}
