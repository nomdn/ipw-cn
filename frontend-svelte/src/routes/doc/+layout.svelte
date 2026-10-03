<script lang="ts">
	/**
	 * 文档区布局（/doc 与 /doc/** 共用）。
	 *
	 * 旧站把这段布局在每个 doc 页面里各抄了一份（doc/index.vue 与 doc/[...slug].vue
	 * 的 .box / .sidebar-menu / .content 完全重复），这里提到 layout 里只写一次：
	 *   宽屏 → 左侧固定侧栏（DocMenu 唯一实现）+ 右侧正文；
	 *   窄屏 → 侧栏整个隐藏，文档导航改由顶栏抽屉承载（见 SiteHeader）。
	 *
	 * 几何对齐旧站 /doc：body 外边距归零（.doc-route）、顶栏 60px、
	 * 正文容器带 40px 上外边距（旧站 `.box .content{margin-top:40px}`），
	 * 于是正文首行落在 y=120（60 顶栏 + 40 外边距 + 20 内边距）——与旧站一致。
	 * 唯一保留的改进：.box 高度取 `100vh - 60px`（旧站写死 100vh，会多出 60px
	 * 让整页产生纵向滚动条），这里刚好铺满视口、不出现页面级滚动条。
	 */
	import 'github-markdown-css/github-markdown-light.css';
	import '#lib/styles/github-markdown-dark.css';
	import DocMenu from '#lib/components/DocMenu.svelte';

	let { children } = $props();

	// 旧站 /doc 的全局样式把 body 设成了 `display:flex; margin:0 auto`（实测该页 body
	// 外边距为 0，而 /、/ssl 仍是 UA 默认的 8px），所以文档页是通栏的。
	// 用 .doc-route 门控：挂载时加上、卸载时摘掉 —— 组件样式一旦注入就常驻，
	// 直接写全局 body{margin:0} 会在 SPA 里泄漏到其它路由。
	$effect(() => {
		document.body.classList.add('doc-route');
		return () => document.body.classList.remove('doc-route');
	});
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
		height: calc(100vh - 60px); /* 顶栏 60px；body 外边距已归零（.doc-route） */
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
		padding: 0; /* 旧站 .sidebar-menu 无内边距：首项紧贴顶栏下沿、hover 底色通栏 */
	}
	:global(html.dark) .doc-sidebar {
		border-right-color: #2e2e2e;
	}

	.doc-content {
		flex: 1;
		margin-top: 40px; /* 旧站 .box .content{margin-top:40px} */
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
