<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { config } from '#lib/config/index.ts';
	import { queryNodePool } from '#lib/node-pool.ts';
	import { visitorIP, visitorIsIPv6 } from '#lib/visitor-ip.svelte.ts';
	import { INPUT_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import PageTitle from '#lib/components/PageTitle.svelte';

	interface ASNResult {
		ip: string;
		geolite2_asn?: { asn: string; org: string };
		dbip_asn?: { asn: string; org: string };
		ip2location_asn?: { asn: string; as: string };
		whois?: any;
	}

	const apiList = config.IPLocationAPI;
	const canonicalUrl = `${config.siteUrl}asn`;

	let tmpIP = $state('1.1.1.1');
	let loading = $state(false);
	let result = $state<ASNResult | null>(null);
	let error = $state('');

	async function queryASN() {
		const ip = tmpIP.trim();
		tmpIP = ip;
		error = '';
		result = null;
		if (!ip) {
			error = '请输入IP地址';
			return;
		}

		loading = true;
		const outcome = await queryNodePool<ASNResult>(
			apiList,
			(id) => `/middleware/${id}/asn/${encodeURIComponent(ip)}`,
			{ onRetry: (msg) => (error = msg) }
		);
		loading = false;

		if (outcome.ok) {
			result = outcome.data;
			error = '';
		} else {
			error = '请求失败，请检查IP或网络';
		}
	}

	onMount(() => {
		const ip = page.url.searchParams.get('ip');
		if (ip) {
			tmpIP = ip;
			queryASN();
		}
	});
</script>

<svelte:head>
	<title>ASN查询 | IP自治系统号查询 | {config.siteName}</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content="专业的ASN自治系统查询工具,支持IP地址ASN号查询,提供Maxmind GEOLite2、DB-IP、IP2Location 多数据源对比,同时集成WHOIS解析获取ASN详细信息,包括组织名称、国家、注册日期等,助力网络运维和路由分析"
	/>
	<meta
		name="keywords"
		content="asn查询,自治系统号,asn lookup,ip asn,asn whois,网络自治系统,运营商asn,ip归属asn"
	/>
	<meta property="og:title" content="ASN自治系统查询工具 - IP ASN号与组织信息查询" />
	<meta
		property="og:description"
		content="多数据源ASN查询,支持Maxmind GEOLite2、DB-IP、IP2Location，集成WHOIS解析获取ASN详细信息"
	/>
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'ASN自治系统查询工具',
		description:
			'专业的ASN自治系统查询工具，支持IP地址ASN号查询，提供Maxmind GEOLite2、DB-IP、IP2Location 多数据源对比，集成WHOIS解析。',
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="ASN 自治系统查询" sub="查询 IP 地址所属的自治系统号（ASN）和组织信息" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={tmpIP}
			placeholder="请输入IP地址（如：1.1.1.1）"
			onkeydown={(e) => e.key === 'Enter' && queryASN()}
		/>
		<Button class={BUTTON_CLASS} disabled={loading} onclick={queryASN}>
			{loading ? '查询中…' : '查询'}
		</Button>
	</div>

	{#if error}
		<div class="error-message">{error}</div>
	{/if}

	{#if result}
		<div class="result-section">
			<table class="result-table">
				<thead>
					<tr>
						<th class="table-header">项目</th>
						<th class="table-header">信息</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td class="table-label">IP 地址</td>
						<td class="table-value">{result.ip}</td>
					</tr>
					{#if result.geolite2_asn?.asn}
						<tr>
							<td class="table-label">Maxmind ASN</td>
							<td class="table-value">
								AS{result.geolite2_asn.asn}
								{#if result.geolite2_asn.org}
									<span style="color: #999; font-size: 0.9em;">{result.geolite2_asn.org}</span>
								{/if}
							</td>
						</tr>
					{/if}
					{#if result.dbip_asn?.asn}
						<tr>
							<td class="table-label">DB-IP ASN</td>
							<td class="table-value">
								{result.dbip_asn.asn}
								{#if result.dbip_asn.org}
									<span style="color: #999; font-size: 0.9em;">{result.dbip_asn.org}</span>
								{/if}
							</td>
						</tr>
					{/if}
					{#if result.ip2location_asn?.asn}
						<tr>
							<td class="table-label">IP2Location ASN</td>
							<td class="table-value">
								{result.ip2location_asn.asn}
								{#if result.ip2location_asn.as}
									<span style="color: #999; font-size: 0.9em;">{result.ip2location_asn.as}</span>
								{/if}
							</td>
						</tr>
					{/if}
					{#if result.whois?.asNumber}
						<tr>
							<td class="table-label">ASN</td>
							<td class="table-value">AS{result.whois.asNumber}</td>
						</tr>
					{/if}
					{#if result.whois?.asName}
						<tr>
							<td class="table-label">AS 名称</td>
							<td class="table-value">{result.whois.asName}</td>
						</tr>
					{/if}
					{#if result.whois?.orgName}
						<tr>
							<td class="table-label">组织名称</td>
							<td class="table-value">{result.whois.orgName}</td>
						</tr>
					{/if}
					{#if result.whois?.orgId}
						<tr>
							<td class="table-label">组织 ID</td>
							<td class="table-value">{result.whois.orgId}</td>
						</tr>
					{/if}
					{#if result.whois?.country}
						<tr>
							<td class="table-label">国家</td>
							<td class="table-value">{result.whois.country}</td>
						</tr>
					{/if}
					{#if result.whois?.regDate}
						<tr>
							<td class="table-label">注册日期</td>
							<td class="table-value">{result.whois.regDate}</td>
						</tr>
					{/if}
					{#if result.whois?.updated}
						<tr>
							<td class="table-label">更新日期</td>
							<td class="table-value">{result.whois.updated}</td>
						</tr>
					{/if}
					{#if result.whois?.abuseEmail}
						<tr>
							<td class="table-label">Abuse 邮箱</td>
							<td class="table-value">{result.whois.abuseEmail}</td>
						</tr>
					{/if}
					{#if result.whois?.abusePhone}
						<tr>
							<td class="table-label">Abuse 电话</td>
							<td class="table-value">{result.whois.abusePhone}</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>
	{/if}

	<blockquote>
		ASN（Autonomous System Number）是互联网中每个自治系统的唯一标识符。<br />
		数据来源：Maxmind GEOLite2、DB-IP，部分节点集成 WHOIS 进一步解析 ASN 详情。<br />
		<a href="/location" target="_blank">IP 归属地查询</a> | <a href="/whois" target="_blank">Whois 查询</a><br />
		访客IP: {visitorIP() || '获取中...'}，您的网络{visitorIsIPv6() ? 'IPv6' : 'IPv4'}优先<br />
	</blockquote>
</div>

<style>
	.result-table .table-label {
		width: 200px;
	}
</style>
