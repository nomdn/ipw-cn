<script lang="ts">
	/** /doc/<slug> —— 正文由 +page.server.ts 在服务端渲染好（markdown → HTML），这里只落位。 */
	let { data } = $props();
</script>

<svelte:head>
	<title>{data.meta.title}</title>
	<meta name="description" content={data.meta.description} />
</svelte:head>

{@html data.html}
