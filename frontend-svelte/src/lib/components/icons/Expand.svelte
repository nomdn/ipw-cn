<script lang="ts">
	/**
	 * Element Plus 图标 `Expand`（MIT）。
	 *
	 * 几何数据抽自 frontend-ssr 依赖的 @element-plus/icons-vue@2.3.2 编译产物，
	 * 与旧站用的是同一份路径 —— 迁移后图标观感不发生任何变化。
	 * 需要新图标时把名字加进 WANT，再跑 scripts/extract-element-plus-icons.mjs，别手抄路径。
	 */
	import type { SVGAttributes } from 'svelte/elements';

	let { class: className = '', ...rest }: SVGAttributes<SVGSVGElement> = $props();
</script>

<svg
	xmlns="http://www.w3.org/2000/svg"
	viewBox="0 0 1024 1024"
	fill="currentColor"
	class={className}
	aria-hidden="true"
	{...rest}
>
	<path d="M128 192h768v128H128zm0 256h512v128H128zm0 256h768v128H128zm576-352 192 160-192 128z" />
</svg>
