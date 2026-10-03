<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleX from '@lucide/svelte/icons/circle-x';
	import { config } from '#lib/config/index.ts';
	import { queryNodePool } from '#lib/node-pool.ts';
	import { visitorIP, visitorIsIPv6 } from '#lib/visitor-ip.svelte.ts';
	import { INPUT_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import PageTitle from '#lib/components/PageTitle.svelte';

	let { data } = $props();

	const apiList = config.IPLocationAPI;
	const canonicalUrl = `${config.siteUrl}ipv6`;

	let ipAddress = $state('');
	let ipLocation = $state<any>({});
	let error = $state('');
	let loading = $state(false);

	/**
	 * 查询指定 IP 的归属地：IPLocationAPI 里的节点逐个试（内层还会切中间件通道）。
	 *
	 * 说明：访客自己的 IP 有两处用到 —— 底部的「访客IP」与网络优先级提示。
	 * 旧站这两处读的是同一个 ref，且只在**没有** `?ip=` 参数时才去探测；
	 * 这里改用全局共享的 `visitorIP()`（见 #lib/visitor-ip），无论从哪个入口进来都能显示，
	 * 属于顺带修好的一个小空白（旧站带 `?ip=` 打开时那两处是空的）。
	 */
	async function locateIP(ip: string) {
		const target = ip.trim();
		ipAddress = target;
		error = '';
		if (!target) {
			error = '请输入IP地址';
			return;
		}

		loading = true;
		const outcome = await queryNodePool<any>(
			apiList,
			(id) => `/middleware/${id}/location/${encodeURIComponent(target)}`,
			{ onRetry: (msg) => (error = msg) }
		);
		loading = false;

		if (outcome.ok) {
			ipLocation = outcome.data ?? {};
			error = '';
		} else {
			ipLocation = {};
			error = '请求失败，请检查网络或稍后重试';
		}
	}

	// 带 `?ip=` 进来就直接查（旧站同样在 onMounted 里这么干）
	$effect(() => {
		const ip = page.url.searchParams.get('ip');
		if (ip) locateIP(ip);
	});
</script>

<svelte:head>
	<title>IPv6/IPv4地址查询工具 | IP归属地定位 | {config.siteName}</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content="专业的IPv6/IPv4地址查询工具,支持IPv4和IPv6地址归属地查询,提供BiliBili Live、GeoCN、IP2Region、Maxmind等多种数据源对比,精确定位IP地理位置、运营商信息,支持中国大陆及境外地址查询,助力IPv6普及与应用"
	/>
	<meta
		name="keywords"
		content="ipv6地址查询,ipv4地址查询,ip归属地,ip地理位置,ip定位,运营商查询,ipv6归属地,ipv4归属地,maxmind,ip2region,geocn"
	/>
	<meta property="og:title" content={`IPv6/IPv4地址归属地查询工具 - ${config.siteName}`} />
	<meta property="og:description" content="多数据源IP地址归属地查询,支持IPv4和IPv6,提供地理位置、运营商等详细信息" />
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'IPv6/IPv4地址归属地查询工具',
		description:
			'专业的IPv6/IPv4地址归属地查询工具，支持BiliBili Live、GeoCN、IP2Region、Maxmind、纯真社区库等多数据源对比，精确定位IP地理位置、运营商信息。',
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="IPv6/IPv4地址归属地查询" sub="极简的IPv6地址查询工具，致力于普及 IPv6" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={ipAddress}
			placeholder="请输入IP地址"
			onkeydown={(e) => e.key === 'Enter' && locateIP(ipAddress)}
		/>
		<Button class={BUTTON_CLASS} disabled={loading} onclick={() => locateIP(ipAddress)}>
			{loading ? '查询中…' : '查询'}
		</Button>
	</div>

	{#if error}
		<div class="error-message">{error}</div>
	{/if}

	<div class="location">
		<div class="ip-info" style="height: 40px;">
			<b>IP</b>&nbsp;<span>{ipAddress}</span>
		</div>

		{#if ipLocation}
			<div class="result-section">
				<table class="result-table">
					<tbody>
						{#if ipLocation.bilibili && (ipLocation.bilibili.administrative_area || ipLocation.bilibili.city)}
							<tr>
								<td class="table-label">bilibili Live接口</td>
								<td class="table-value">
									<span>
										{ipLocation.bilibili?.country}&nbsp;{ipLocation.bilibili?.administrative_area}&nbsp;{ipLocation
											.bilibili?.city}
									</span>
								</td>
								<td class="table-value"><span>{ipLocation.bilibili?.isp}</span></td>
							</tr>
						{/if}
						{#if ipLocation.ip2region && typeof ipLocation.ip2region === 'object' && (ipLocation.ip2region.administrative_area || ipLocation.ip2region.city)}
							<tr>
								<td class="table-label">IP2Region</td>
								<td class="table-value">
									{ipLocation.ip2region?.country}&nbsp;{ipLocation.ip2region
										?.administrative_area}&nbsp;{ipLocation.ip2region?.city}
								</td>
								<td class="table-value">{ipLocation.ip2region?.isp}</td>
							</tr>
						{/if}
						{#if ipLocation.ip2location && typeof ipLocation.ip2location === 'object' && (ipLocation.ip2location.administrative_area || ipLocation.ip2location.city)}
							<tr>
								<td class="table-label">IP2Location</td>
								<td class="table-value">
									{ipLocation.ip2location?.country}&nbsp;{ipLocation.ip2location
										?.administrative_area}&nbsp;{ipLocation.ip2location?.city}
								</td>
								<td class="table-value">{ipLocation.ip2location_asn?.as}</td>
							</tr>
						{/if}
						{#if ipLocation.geocn && (ipLocation.geocn.administrative_area || ipLocation.geocn.city || ipLocation.geocn.district)}
							<tr>
								<td class="table-label">GeoCN(仅中国大陆)</td>
								<td class="table-value">
									{ipLocation.geocn?.administrative_area}&nbsp;{ipLocation.geocn?.city}&nbsp;{ipLocation
										.geocn?.district}
								</td>
								<td class="table-value">{ipLocation.geocn?.isp}&nbsp;{ipLocation.geocn?.type}</td>
							</tr>
						{/if}
						{#if ipLocation.maxmind_city && ipLocation.maxmind_asn && (ipLocation.maxmind_city.country || ipLocation.maxmind_city.city)}
							<tr>
								<td class="table-label">Maxmind GEOLite2 City</td>
								<td class="table-value">
									{ipLocation.maxmind_city?.country}&nbsp;{ipLocation.maxmind_city
										?.administrative_area}&nbsp;{ipLocation.maxmind_city?.city}
								</td>
								<td class="table-value">{ipLocation.maxmind_asn?.org}</td>
							</tr>
						{/if}
						{#if ipLocation.qqwry && (ipLocation.qqwry.country || ipLocation.qqwry.administrative_area || ipLocation.qqwry.city)}
							<tr>
								<td class="table-label">纯真社区库</td>
								<td class="table-value">
									{ipLocation.qqwry?.country}&nbsp;{ipLocation.qqwry?.administrative_area}&nbsp;{ipLocation
										.qqwry?.city}
								</td>
								<td class="table-value">{ipLocation.qqwry?.isp}</td>
							</tr>
						{/if}
						{#if ipLocation.dbip_city && (ipLocation.dbip_city.administrative_area || ipLocation.dbip_city.city)}
							<tr>
								<td class="table-label">DB-IP City</td>
								<td class="table-value">
									{ipLocation.dbip_city?.country}&nbsp;{ipLocation.dbip_city
										?.administrative_area}&nbsp;{ipLocation.dbip_city?.city}
								</td>
								<td class="table-value">{ipLocation.dbip_asn?.org}</td>
							</tr>
						{/if}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	<div style="font-size: 1.5em;">
		{#if visitorIsIPv6()}
			<h3>
				<CircleCheck class="inline-block size-[1em] align-[-0.12em] fill-current text-[lightgreen]" />
				您的网络IPv6优先
			</h3>
		{:else if visitorIP()}
			<h3>
				<CircleX class="inline-block size-[1em] align-[-0.12em] text-red-600" />
				您的网络IPv4优先
			</h3>
		{/if}
	</div>

	<blockquote>
		精度参照表：<br />
		中国大陆:BiliBili Live &gt; GeoCN &gt; IP2Region &gt; 纯真社区库 &gt; Maxmind GEOLite2 City ≈ DB-IP<br />
		中国大陆 IPv6 地址: BiliBili Live &gt; GeoCN &gt; IP2Region &gt; Maxmind GEOLite2 City ≈ DB-IP &gt; 纯真社区库<br
		/>
		境外及港澳台地址: Maxmind GEOLite2 City ≈ DB-IP &gt; BiliBili Live &gt; IP2Region &gt; 纯真社区库<br />
		手机默认开启 IPv6，宽带开启 IPv6 请自行搜索<br />
		访客IP: {visitorIP()}，<span
			>{visitorIsIPv6() ? '您的网络IPv6优先' : visitorIP() ? '您的网络IPv4优先' : ''}</span
		>
	</blockquote>

	<div>{@html data.highlighted}</div>
</div>

<style>
	/* 表格列宽与旧站一致（结果表本身的样子在 site.css 里） */
	.result-table .table-label {
		width: 200px;
	}
</style>
