<script lang="ts">
	/**
	 * Element Plus 图标 `Close`（MIT）。
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
	<path d="M764.288 214.592 512 466.88 259.712 214.592a31.936 31.936 0 0 0-45.12 45.12L466.752 512 214.528 764.224a31.936 31.936 0 1 0 45.12 45.184L512 557.184l252.288 252.288a31.936 31.936 0 0 0 45.12-45.12L557.12 512.064l252.288-252.352a31.936 31.936 0 1 0-45.12-45.184z" />
</svg>
