import { defineEnvVars } from '@sveltejs/kit/env';

/**
 * 环境变量声明（SvelteKit 3 的 `$app/env/private` 只暴露在这里声明过的键）。
 *
 * 为什么要声明而不是直接 `env.XXX`：Kit 3 在构建期只会把**声明过**的变量名生成为
 * `$app/env/private` 的具名导出（以及运行时的赋值 setter），没声明的键在产物里
 * 会被静态分析判定为 `undefined` 并整段摇树掉 —— 也就是运行时永远读不到值。
 * 这一点与 Nuxt 的 `runtimeConfig` 不同（Nuxt 是声明式配置，SvelteKit 3 是这里）。
 *
 * 消费方写法：`import { API_KEYS, APIKEYS } from '$app/env/private'`。
 * **不要**再走 `$env/dynamic/private` —— Kit 3 已把它标为废弃
 * （dev 下会打 env_module_deprecated 警告），且它的任意键访问会被摇树掉。
 *
 * 部署侧（Cloudflare Worker）对应的变量名：
 *   API_KEYS —— 各拨测节点的 token 表，JSON 字符串：{"<backendID>": "<token>"}
 *   APIKEYS  —— 旧站沿用的写法，等价；两者都存在时以 API_KEYS 为准
 * 两个都给成可选（未配置时为空串），未配置时中间件不注入 Authorization 头，
 * 由上游自行决定是否放行 —— 与旧站行为一致。
 */
export const variables = defineEnvVars({
	API_KEYS: {
		public: false,
		static: false,
		schema: (value) => value ?? '',
		description: '拨测节点 token 表（JSON：{"<backendID>": "<token>"}）'
	},
	APIKEYS: {
		public: false,
		static: false,
		schema: (value) => value ?? '',
		description: 'API_KEYS 的历史别名（旧站环境变量名）'
	}
});
