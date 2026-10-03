<script lang="ts">
	import { config } from '#lib/config/index.ts';
	import TcpingPanel from '#lib/components/TcpingPanel.svelte';
	import PageTitle from '#lib/components/PageTitle.svelte';
</script>

<svelte:head>
	<title>TCPing测试工具 | IPv4服务器连通性检测 | {config.siteName}</title>
	<link rel="canonical" href={`${config.siteUrl}tcping`} />
	<meta
		name="description"
		content="专业的IPv4 TCPing测试工具,提供多节点TCP连通性检测服务,支持自定义端口测试,实时检测服务器丢包率、平均延迟、最大最小响应时间,帮助运维人员快速诊断服务器网络质量,确保服务稳定运行"
	/>
	<meta
		name="keywords"
		content="tcping测试,tcp连通性检测,ipv4 tcping,服务器延迟测试,丢包率检测,端口连通性,服务器网络测试,网络质量检测"
	/>
	<meta property="og:title" content="TCPing测试工具 - IPv4服务器连通性与延迟检测" />
	<meta property="og:description" content="多节点TCPing测试,检测服务器连通性、丢包率与响应延迟" />
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'IPv4 TCPing连通性测试工具',
		description:
			'专业的IPv4 TCPing测试工具，多节点检测服务器连通性和延迟，支持自定义端口测试，提供丢包率、平均延迟、最大最小响应时间等数据。',
		url: `${config.siteUrl}tcping`,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="IPv4 TCPing 测试" sub="多节点 TCPing 测试，检测服务器连通性和延迟" />

<div class="content">
	<TcpingPanel family="ipv4" />
</div>
