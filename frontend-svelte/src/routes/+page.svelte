<script lang="ts">
	import { onMount } from 'svelte';
	import { isIPv6 } from 'is-ip';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import { config } from '#lib/config/index.ts';
	import { isIPv4 } from '#lib/tools.ts';

	let { data } = $props();

	const canonicalUrl = `${config.siteUrl}/`;

	// 三个地址各自独立请求、各自渲染：谁先回来谁先上屏。
	// 不用 Promise.allSettled 包起来等齐——那样只要有一个慢（例如纯 IPv4 网络下 v6 接口
	// 要等超时），已经拿到的 IPv4 与双栈结果也得一起干等。
	//
	// 同一条通道只打一次：浏览器自身的 fetch 不会重复发起（旧站要显式传 ofetch 的
	// retry:false 才躲开它默认的 GET 重试；原生 fetch 没这个问题）。
	let ipAddress = $state('');
	let yourIPv4 = $state('');
	let yourIPv6 = $state('');

	onMount(() => {
		const controller = new AbortController();
		fetch(config.DualStackAPI, { signal: controller.signal })
			.then((r) => r.text())
			.then((ip) => (ipAddress = ip.trim()))
			// 单个接口失败不影响其它两行展示，保持「查询中」占位
			.catch(() => {});
		fetch(config.v4OnlyAPI, { signal: controller.signal })
			.then((r) => r.text())
			.then((ip) => (yourIPv4 = ip.trim()))
			.catch(() => {});
		fetch(config.v6OnlyAPI, { signal: controller.signal })
			.then((r) => r.text())
			.then((ip) => (yourIPv6 = ip.trim()))
			.catch(() => {});
		return () => controller.abort();
	});
</script>

<svelte:head>
	<title>{config.siteName} | IP查询工具 | IPv4/IPv6地址查询与网络测试平台</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content={`${config.siteName}提供专业的IP查询服务,支持IPv4和IPv6地址在线查询、归属地定位、网络测速、DNS解析、SSL证书检测、TCPing测试等多种网络工具,致力于推进IPv6规模部署和应用,打造去中心化的IP查询平台`}
	/>
	<meta
		name="keywords"
		content="ipv6,ipv4,ip查询,ipv6查询,ipv4查询,ipv6地址查询,ipv4地址查询,网络测速,DNS查询,SSL检测,TCPing,IP归属地,IPv6优先"
	/>
	<meta property="og:title" content={`${config.siteName} - 专业IP查询与网络测试工具平台`} />
	<meta
		property="og:description"
		content="提供IPv4/IPv6地址查询、网络测速、DNS解析、SSL检测等全方位网络诊断工具,助力IPv6普及与部署"
	/>
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	<meta property="og:url" content={canonicalUrl} />
	<meta name="twitter:card" content="summary_large_image" />

	<!-- 结构化数据（JSON-LD）：与旧站一致，利于搜索引擎理解站点用途。
	     用 {@html} 输出是 Kit 里写 JSON-LD 的常规做法 —— Kit 会把这段内联脚本的
	     hash 一并算进 CSP，所以不会被自己的 CSP 拦掉。 -->
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: config.siteName,
		description: `${config.siteName}提供专业的IP查询服务,支持IPv4和IPv6地址在线查询、归属地定位、网络测速、DNS解析、SSL证书检测、TCPing测试等多种网络工具,致力于推进IPv6规模部署和应用,打造去中心化的IP查询平台`,
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Any',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		featureList:
			'IPv4地址查询,IPv6地址查询,IP归属地定位,网络测速,DNS解析,SSL证书检测,TCPing测试,IPv6优先检测',
		about: [
			{
				'@type': 'Thing',
				name: 'IP地址查询',
				description:
					'通过特定的IP地址获取相关的地理位置、运营商、网络类型等信息的技术服务。IPv4地址是32位地址格式,IPv6地址是128位地址格式,IPv6能够提供更大的地址空间,解决IPv4地址枯竭问题。'
			},
			{
				'@type': 'Thing',
				name: 'IPv6优先检测',
				description:
					'当访问一个同时支持IPv4和IPv6的双栈网站时,如果网络IPv6优先,系统会优先使用IPv6地址进行连接。通过访问双栈测试域名来判断网络优先级。'
			},
			{
				'@type': 'Thing',
				name: 'IPv6部署',
				description:
					'IPv6是全球下一代互联网协议标准,相比IPv4具有更大的地址空间、更好的安全性、更高的网络效率。国家正在大力推进IPv6规模部署和应用,以适应未来互联网发展需求。'
			}
		]
	})}</script>`}
</svelte:head>

<div class="title">
	<header>
		<h1>IP查询</h1>
		<p>致力于IP查询去中心化,推进 IPv6 规模部署和应用</p>
	</header>
</div>

<div class="content">
	<div class="one-line">
		<b>IPv4</b>&nbsp;<p>{yourIPv4}</p>&nbsp;<a href={`/location?ip=${yourIPv4}`} target="_blank"
			>查询归属地</a
		>
	</div>
	<div class="one-line">
		<b>IPv6</b>&nbsp;
		{#if yourIPv6}
			<p>{yourIPv6}</p>
			<a href={`/location?ip=${yourIPv6}`} target="_blank">&nbsp;查询归属地</a>
		{:else}
			<a href="/doc/user/enable_ipv6" target="_blank">没有IPv6地址,查看如何开启IPv6</a>
		{/if}
	</div>
	<div class="ip-priority">
		{#if ipAddress && isIPv6(ipAddress)}
			<h2>
				<CircleCheck
					class="inline-block size-[1em] align-[-0.12em] text-[lightgreen]"
					aria-hidden="true"
				/>
				您的网络IPv6优先
			</h2>
		{:else if ipAddress && isIPv4(ipAddress)}
			<h2>
				<CircleX class="inline-block size-[1em] align-[-0.12em] text-red-600" aria-hidden="true" />
				您的网络IPv4优先
			</h2>
		{:else}
			<h2>
				<CircleX class="inline-block size-[1em] align-[-0.12em]" aria-hidden="true" />
				查询中，请稍后
			</h2>
		{/if}
	</div>
	<blockquote>
		手机默认开启 IPv6，宽带开启 IPv6 请参阅<a href="/doc/user/enable_ipv6" target="_blank">文档</a>
	</blockquote>

	<!-- SSR 期已高亮好；失败时落到纯文本兜底（两条分支互斥，不会把纯文本当 HTML 渲染） -->
	{#if data.highlighted}
		<div class="code-block">{@html data.highlighted}</div>
	{:else}
		<div class="code-block code-block--fallback">{data.code}</div>
	{/if}
</div>


<style>
	/* 网络优先级提示：原来是「div 1.5em × h3」，但 h1 之后直跳 h3 违反标题层级（heading-order），
	   语义上应为 h2。site.css 给 h3 显式定了 1.3em / 窄屏 1.1em，h2 只有 1.5em 一档，
	   所以这里把字号按原样补回，避免语义修正顺带改了视觉（含 margin：UA 的 h3 是 1em，h2 是 0.83em）。 */
	.ip-priority {
		font-size: 1.5em;
	}
	.ip-priority h2 {
		font-size: 1.3em;
		margin: 1em 0;
	}

	.code-block {
		margin-top: 1rem;
		padding: 1rem;
		border-radius: 0.75rem;
		overflow-x: auto;
		max-width: 100%;
		white-space: pre-wrap;
		overflow-wrap: break-word;
	}

	.code-block--fallback {
		background: rgb(48, 46, 46);
		border: 1px solid rgba(62, 175, 124, 0.18);
		font-family:
			'JetBrains Mono', 'Fira Code', 'Cascadia Code', 'Consolas', 'Monaco', 'Courier New',
			monospace !important;
		color: rgb(255, 255, 255);
		white-space: pre-wrap;
	}

	@media (max-width: 768px) {
		.ip-priority h2 {
			font-size: 1.1em;
		}
		.code-block {
			padding: 0.75rem;
			font-size: 0.8em;
		}
	}
</style>
