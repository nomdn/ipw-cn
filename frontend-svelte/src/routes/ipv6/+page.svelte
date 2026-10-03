<script lang="ts">
	// 永远不会渲染：+page.server.ts 的 load 一律 302 到 /location。
	// 这个文件存在的意义只是让 SvelteKit 认下 /ipv6 这条路由。
</script>

<div class="redirecting">正在跳转到 IP 归属地查询…</div>

<style>
	.redirecting {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 50vh;
		color: #999;
	}
</style>
