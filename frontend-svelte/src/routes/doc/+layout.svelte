<script lang="ts">
	/**
	 * 文档区布局（/doc 与 /doc/** 共用）。
	 *
	 * 旧站把这段布局在每个 doc 页面里各抄了一份（doc/index.vue 与 doc/[...slug].vue
	 * 的 .box / .sidebar-menu / .content 完全重复），这里提到 layout 里只写一次：
	 *   宽屏 → 左侧固定侧栏（DocMenu 唯一实现）+ 右侧正文；
	 *   窄屏 → 侧栏整个隐藏，文档导航改由顶栏抽屉承载（见 SiteHeader）。
	 *
	 * 尺寸上唯一与旧站不同的地方：旧站 `.box { height: 100vh }` 是排在静态顶栏**下面**的，
	 * 实际会超出视口一屏高；这里顶栏是 sticky、高度固定 4rem，所以用
	 * `calc(100vh - 4rem)` 让这块刚好铺满剩余视口，页面本身不产生纵向滚动条
	 * —— 旧站那段的意图（"防止出现页面级滚动条"）在这里才真正成立。
	 */
	import 'github-markdown-css/github-markdown-light.css';
	import '#lib/styles/github-markdown-dark.css';
	import DocMenu from '#lib/components/DocMenu.svelte';

	let { children } = $props();
</script>

<div class="doc-box">
	<!-- 文档导航只有一份实现（DocMenu）。
	     宽屏：就是这条左侧固定侧栏；窄屏：本侧栏隐藏，改由顶部抽屉菜单承载。 -->
	<aside class="doc-sidebar">
		<DocMenu />
	</aside>
	<div class="doc-content">
		<div class="markdown-body">
			{@render children()}
		</div>
	</div>
</div>

<style>
	.doc-box {
		display: flex;
		flex-direction: row;
		height: calc(100vh - 4rem); /* 4rem = 顶栏高度 */
		width: 100%;
		background-color: #fff;
	}
	:global(html.dark) .doc-box {
		background-color: #242424;
	}

	.doc-sidebar {
		width: 260px;
		height: 100%;
		flex-shrink: 0;
		overflow-y: auto;
		border-right: 1px solid #e4e7ed;
		padding: 12px;
	}
	:global(html.dark) .doc-sidebar {
		border-right-color: #2e2e2e;
	}

	.doc-content {
		flex: 1;
		padding: 20px;
		overflow-y: auto;
		background-color: #fff;
	}
	:global(html.dark) .doc-content {
		background-color: #242424;
	}

	@media (max-width: 768px) {
		/* 窄屏不再挤一条窄侧栏（原先 100px，长标题全被截断）：
		   文档导航已并入左侧抽屉菜单，这里直接撤掉，正文铺满整屏 */
		.doc-sidebar {
			display: none;
		}
		.doc-content {
			padding: 10px;
		}
	}
</style>
