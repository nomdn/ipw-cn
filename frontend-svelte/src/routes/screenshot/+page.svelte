<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import LoaderCircle from '@lucide/svelte/icons/loader-circle';
	import { config } from '#lib/config/index.ts';
	import { visitorIP, visitorIsIPv6 } from '#lib/visitor-ip.svelte.ts';
	import { INPUT_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import PageTitle from '#lib/components/PageTitle.svelte';

	let { data } = $props();

	let tmpDomain = $state('https://www.zakoflare.com');
	let url = $state('');
	let loading = $state(false);
	let error = $state('');
	let screenshotUrl = $state('');
	let imgLoaded = $state(false);

	let detectionImg: HTMLImageElement | null = null;
	let detectionTimeout: ReturnType<typeof setTimeout> | null = null;
	let currentRequestId = 0;

	// WordPress mshots 默认占位图（生成失败/重定向目标）的尺寸是 1200x900；
	// 真实网站截图可以是任意尺寸。
	const DEFAULT_MSHOTS_WIDTH = 1200;
	const DEFAULT_MSHOTS_HEIGHT = 900;
	const LOADING_TIMEOUT = 15000; // 15 seconds

	const titleSite = $derived(page.url.searchParams.get('site') || config.siteName);

	function cleanupDetection() {
		if (detectionImg) {
			detectionImg.onload = null;
			detectionImg.onerror = null;
			detectionImg = null;
		}
		if (detectionTimeout) {
			clearTimeout(detectionTimeout);
			detectionTimeout = null;
		}
	}

	function showError(message: string) {
		cleanupDetection();
		error = message;
		loading = false;
		screenshotUrl = '';
		imgLoaded = false;
	}

	function takeScreenshot() {
		url = tmpDomain.trim();
		if (!url) {
			error = '请输入网址';
			return;
		}

		if (!url.startsWith('http://') && !url.startsWith('https://')) {
			url = 'https://' + url;
		}

		cleanupDetection();
		error = '';
		imgLoaded = false;
		loading = true;
		screenshotUrl = '';

		const requestId = ++currentRequestId;

		screenshotUrl = `https://s0.wp.com/mshots/v1/${encodeURIComponent(url)}`;

		// 隐藏的探测图：mshots 在目标站点不可达/生成超时时会 302 到一张 1200x900 的默认图，
		// 靠尺寸把它认出来（真截图尺寸是任意的）。
		detectionImg = new Image();

		detectionImg.onload = () => {
			if (requestId !== currentRequestId) return;

			// 307 到 mshots 默认图 —— 目标还在生成中
			if (detectionImg!.currentSrc?.endsWith('/mshots/v1/default')) {
				cleanupDetection();
				loading = false;
				error = '正在生成截图中，请稍后...';
				return;
			}

			// WordPress mshots 的默认错误图正好是 1200x900
			if (
				detectionImg!.naturalWidth === DEFAULT_MSHOTS_WIDTH &&
				detectionImg!.naturalHeight === DEFAULT_MSHOTS_HEIGHT
			) {
				showError('截图加载失败，可能是目标站点无法访问或生成超时');
				return;
			}

			// 通过检测
			cleanupDetection();
			imgLoaded = true;
			loading = false;
		};

		detectionImg.onerror = () => {
			if (requestId !== currentRequestId) return;
			showError('截图加载失败，请检查网址是否正确');
		};

		detectionImg.src = screenshotUrl;

		// 超时兜底：探测在 LOADING_TIMEOUT 内没有结论就按失败处理
		detectionTimeout = setTimeout(() => {
			if (requestId !== currentRequestId) return;
			showError('截图加载失败，可能是目标站点无法访问或生成超时');
		}, LOADING_TIMEOUT);
	}

	// 展示用的 <img> 是否触发 load 取决于浏览器对 302 的处理，
	// 主判据是上面那张隐藏探测图；这里只在确认是默认图时补一条提示。
	function onImageLoad() {
		if (detectionImg && detectionImg.currentSrc?.endsWith('/mshots/v1/default')) {
			error = '正在生成截图中，请稍后...';
			loading = false;
		}
	}

	function onImageError() {
		showError('截图加载失败，请检查网址是否正确');
	}

	onMount(() => {
		const site = page.url.searchParams.get('site');
		if (site) {
			tmpDomain = site;
			takeScreenshot();
		}
	});
</script>

<svelte:head>
	<title>网站截图工具 - {titleSite}</title>
	<meta
		name="description"
		content="在线网站截图工具，输入网址即可获取网页快照，支持所有公开网站的实时截图，方便查看网站布局和设计效果"
	/>
	<meta name="keywords" content="网站截图,网页快照,在线截图,网页预览,网站布局查看,截图工具" />
	<meta property="og:title" content="网站截图工具 - 在线网页快照" />
	<meta property="og:description" content="输入网址即可获取网页快照，查看网站布局和设计效果" />
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
</svelte:head>

<PageTitle h1="网站截图" sub="输入网址，自动截图网站页面" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={tmpDomain}
			placeholder="请输入网址（如：https://example.com）"
			onkeydown={(e) => e.key === 'Enter' && takeScreenshot()}
		/>
		<Button class={BUTTON_CLASS} disabled={loading} onclick={takeScreenshot}>
			{loading ? '截图中…' : '网站截图'}
		</Button>
	</div>

	{#if error}
		<div class="error-message">{error}</div>
	{/if}

	{#if screenshotUrl}
		<div class="result-section">
			<div class="screenshot-container">
				{#if loading && !imgLoaded}
					<div class="loading-overlay">
						<LoaderCircle class="size-10 animate-spin" aria-hidden="true" />
						<p>正在截图中...</p>
					</div>
				{/if}
				<img
					src={screenshotUrl}
					alt="网站截图"
					class="screenshot-img"
					class:img-loaded={imgLoaded}
					onload={onImageLoad}
					onerror={onImageError}
				/>
			</div>
		</div>
	{/if}

	{#if data.doc}
		<div class="markdown">
			{@html data.doc}
		</div>
	{/if}

	<blockquote>
		访客IP: {visitorIP()}，您的网络 {visitorIsIPv6() ? 'IPv6' : 'IPv4'} 访问优先<br />
	</blockquote>
</div>

<style>
	.screenshot-container {
		position: relative;
		display: inline-block;
		width: 100%;
		max-width: 100%;
		overflow: hidden;
		border-radius: 8px;
		box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
		background: #f5f5f5;
		min-height: 200px;
	}

	.screenshot-img {
		display: block;
		max-width: 100%;
		height: auto;
		opacity: 0;
		transition: opacity 0.3s ease;
	}

	.screenshot-img.img-loaded {
		opacity: 1;
	}

	:global(html.dark) .screenshot-container {
		background: #1a1919;
	}

	.loading-overlay {
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		background: #f5f5f5;
		z-index: 1;
	}

	:global(html.dark) .loading-overlay {
		background: #1a1919;
		color: #adbac7;
	}

	.loading-overlay p {
		margin-top: 12px;
		color: #606266;
		font-size: 1.1em;
	}

	:global(html.dark) .loading-overlay p {
		color: #c0c4cc;
	}
</style>
