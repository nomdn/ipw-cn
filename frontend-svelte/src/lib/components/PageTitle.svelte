<script lang="ts">
	/**
	 * 工具页顶部标题块（`.title > header > h1 + p`）。
	 * 旧站 13 个页面各抄一遍这 5 行，样式来自 site.css；这里收成一个组件。
	 */
	let { h1, sub }: { h1: string; sub?: string } = $props();
</script>

<div class="title">
	<header>
		<h1>{h1}</h1>
		{#if sub}<p>{sub}</p>{/if}
	</header>
</div>
