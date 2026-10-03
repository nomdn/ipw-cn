<script lang="ts">
	import { config } from '#lib/config/index.ts';
	import SpeedtestPanel from '#lib/components/SpeedtestPanel.svelte';
	import PageTitle from '#lib/components/PageTitle.svelte';
</script>

<svelte:head>
	<title>IPv4网站测速工具 | 全国多节点并发测速 | {config.siteName}</title>
	<link rel="canonical" href={`${config.siteUrl}speedtest`} />
	<meta
		name="description"
		content="专业的IPv4网站测速工具,提供全国多节点并发测速服务,快速返回网站响应时间、下载速度、页面大小、DNS解析时间、HTTP连接时间等详细性能指标,支持IPv4网站性能检测与优化,助力网站性能监控与用户体验改善"
	/>
	<meta
		name="keywords"
		content="ipv4网站测速,ipv4测速,网站速度测试,ipv4性能检测,网站响应时间,ipv4下载速度,ipv4性能优化,网站性能监控"
	/>
	<meta property="og:title" content="IPv4网站测速 - 全国多节点并发性能检测工具" />
	<meta property="og:description" content="全国多节点IPv4网站测速,快速获取响应时间、下载速度等性能数据" />
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'IPv4网站性能测速工具',
		description:
			'专业的IPv4网站测速工具，全国多节点并发测速，提供响应时间、下载速度、页面大小、DNS解析时间等详细性能指标。',
		url: `${config.siteUrl}speedtest`,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="IPv4 网站测速" sub="全国并发测速，1s 内快速返回测速结果" />

<div class="content">
	<SpeedtestPanel family="ipv4" />
</div>
