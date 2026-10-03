<script lang="ts">
	import { config } from '#lib/config/index.ts';
	import { visitorIP, visitorIsIPv6 } from '#lib/visitor-ip.svelte.ts';
	import TcpingPanel from '#lib/components/TcpingPanel.svelte';
	import PageTitle from '#lib/components/PageTitle.svelte';
</script>

<svelte:head>
	<title>IPv6 TCPing测试工具 | IPv6服务器连通性检测 | {config.siteName}</title>
	<link rel="canonical" href={`${config.siteUrl}ipv6tcping`} />
	<meta
		name="description"
		content="专业的IPv6 TCPing测试工具,提供多节点IPv6 TCP连通性检测服务,支持自定义端口测试,实时检测IPv6服务器丢包率、平均延迟、最大最小响应时间,助力IPv6网络质量诊断与优化,推进IPv6规模部署和应用"
	/>
	<meta
		name="keywords"
		content="ipv6 tcping测试,ipv6连通性检测,ipv6服务器延迟,ipv6丢包率,ipv6端口测试,ipv6网络质量,ipv6服务器测试,ipv6网络诊断"
	/>
	<meta property="og:title" content="IPv6 TCPing测试 - IPv6服务器连通性与延迟检测" />
	<meta property="og:description" content="多节点IPv6 TCPing测试,检测服务器连通性、丢包率与响应延迟" />
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'IPv6 TCPing连通性测试工具',
		description:
			'专业的IPv6 TCPing测试工具，多节点检测IPv6服务器连通性和延迟，支持自定义端口测试，提供丢包率、平均延迟、最大最小响应时间等数据。',
		url: `${config.siteUrl}ipv6tcping`,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="IPv6 TCPing 测试" sub="多节点 TCPing 测试，检测服务器连通性和延迟" />

<div class="content">
	<TcpingPanel family="ipv6" />

	<blockquote>
		<a href="/doc/user/ipv6_ping" target="_blank">IPv6 Ping 原理介绍</a><br />
		<strong>注意本页是TCPing，不是ICMPv6 Ping，下列文本仅供参考</strong><br />
		<strong>1. 本地 IPv6 方式</strong><br />
		Windows: ping -6 ipw.wsmdn.top<br />

		macOS 或 Linux: ping6 ipw.wsmdn.top<br />
		<strong>2. 服务器 IPv6 Ping 失败可能原因：</strong><br />
		服务器已开启 IPv6，但防火墙（又名安全组）未对源地址是 IPv6 地址(::/0)的 ICMPv6协议 开放访问，<br />
		服务器未开启 IPv6，请参考 <a href="/doc/server/website_enable_ipv6" target="_blank">服务器开启 IPv6</a
		><br />
		<a href="/tcping" target="_blank">IPv4 TCPing 测试</a> | <a href="/ipv6speedtest" target="_blank"
			>IPv6 网站测速</a
		> | <a href="/ipv6webcheck">网站开启IPv6检测</a> | <a href="/dns">DNS解析查询</a><br />

		访客IP: {visitorIP()}，您的网络{visitorIsIPv6() ? 'IPv6' : 'IPv4'}访问优先
	</blockquote>
</div>
