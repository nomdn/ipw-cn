<script lang="ts">
	/**
	 * Element Plus 图标 `CircleCloseFilled`（MIT）。
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
	<path d="M512 64a448 448 0 1 1 0 896 448 448 0 0 1 0-896m0 393.664L407.936 353.6a38.4 38.4 0 1 0-54.336 54.336L457.664 512 353.6 616.064a38.4 38.4 0 1 0 54.336 54.336L512 566.336 616.064 670.4a38.4 38.4 0 1 0 54.336-54.336L566.336 512 670.4 407.936a38.4 38.4 0 1 0-54.336-54.336z" />
</svg>
