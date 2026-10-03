<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import '../app.css';
	import '#lib/styles/site.css';
	import { config } from '#lib/config/index.ts';
	import { ipProbeOrigins, middlewareOrigin } from '#lib/preconnect.ts';
	import { isDocPath } from '#lib/nav.ts';
	import { initTheme } from '#lib/theme.ts';
	import SiteHeader from '#lib/components/SiteHeader.svelte';
	import SiteFooter from '#lib/components/SiteFooter.svelte';

	let { children } = $props();

	// 文档页是阅读页：正文底部再加备案块会打断阅读，窄屏还会和侧栏抽屉抢滚动区（旧站同此约定）。
	const isDocRoute = $derived(isDocPath(page.url.pathname));

	onMount(() => {
		const disposeTheme = initTheme();

		// Umami 统计：水合后再注入，不占关键路径。
		// （旧站也是 onMounted 里 appendChild；这样一来统计脚本不会阻塞首帧，
		//  代价是首屏窗口内的 PV 依赖脚本就绪速度——可接受。）
		let umamiScript: HTMLScriptElement | null = null;
		if (!umamiScript && config.umamiScriptUrl) {
			umamiScript = document.createElement('script');
			umamiScript.src = config.umamiScriptUrl;
			umamiScript.async = true;
			umamiScript.setAttribute('data-website-id', config.umamiWebsiteId);
			document.head.appendChild(umamiScript);
		}

		return () => disposeTheme?.();
	});
</script>

<!-- 站点元信息：与旧站 app.vue 的 useHead 一致（noindex 开关 + 百度站长验证） -->
<svelte:head>
	<link rel="icon" type="image/x-icon" href="/favicon.svg" />
	<!-- 三个 IP 探测源站 preconnect。crossorigin 不能省：页面的 fetch 是跨域（CORS）模式，
	     不带它预热出来的是 no-cors 连接，真正的 fetch 用不上、还得重连。
	     中间件上游只在部分页面用到，dns-prefetch 即可，不占连接。 -->
	{#each ipProbeOrigins as origin (origin)}
		<link rel="preconnect" href={origin} crossorigin="anonymous" />
	{/each}
	{#if middlewareOrigin}
		<link rel="dns-prefetch" href={middlewareOrigin} />
	{/if}
	{#if config.noindex}
		<meta name="robots" content="noindex, nofollow" />
		<meta name="googlebot" content="noindex, nofollow" />
		<meta name="bingbot" content="noindex, nofollow" />
	{/if}
	<meta name="baidu-site-verification" content="codeva-xzdbvF8gQu" />
</svelte:head>

<SiteHeader />

<!-- 不加 role="main"：<main> 元素本身已经是 main landmark，
     再写一遍属于冗余（axe 的 redundant-role / Lighthouse 会点名）。 -->
<main id="main-content">
	{@render children()}
</main>

{#if !isDocRoute}
	<SiteFooter />
{/if}
