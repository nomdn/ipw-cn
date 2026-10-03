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

	let { data } = $props();

	const apiList = config.APIBaseURL.DualStack;

	let tmpdomain = $state('zakoflare.com');
	let domain = $state('');
	let loading = $state(false);
	let result = $state<any>(null);
	let error = $state('');

	/** `?site=` 优先作为标题后缀（旧站如此），否则用站名 */
	const titleSite = $derived(page.url.searchParams.get('site') || config.siteName);
	const canonicalUrl = `${config.siteUrl}whois`;

	async function queryWhois() {
		const target = tmpdomain.trim();
		domain = target;
		error = '';
		result = null;
		if (!target) {
			error = '请输入域名';
			return;
		}

		loading = true;
		const outcome = await queryNodePool<any>(
			apiList,
			(id) => `/middleware/${id}/whois/${encodeURIComponent(target)}`,
			{ onRetry: (msg) => (error = msg) }
		);
		loading = false;

		if (outcome.ok) {
			result = outcome.data;
			error = '';
		} else {
			error = '请求失败，请检查域名或网络';
		}
	}

	function formatDate(dateStr: string): string {
		if (!dateStr) return '-';
		const d = new Date(dateStr);
		return d.toLocaleString('zh-CN', {
			year: 'numeric',
			month: '2-digit',
			day: '2-digit',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	function getStatusClass(status: string): string {
		if (status.includes('prohibited') || status.includes('lock')) return 'status-warning';
		return 'status-success';
	}

	onMount(() => {
		const site = page.url.searchParams.get('site');
		if (site) {
			tmpdomain = site;
			queryWhois();
		}
	});
</script>

<svelte:head>
	<title>Whois查询 | {titleSite}</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content="专业的Whois域名查询工具,支持.com、.net等国内外域名WHOIS信息查询,提供域名注册商、注册时间、到期时间、DNS服务器等详细信息,助力域名管理和交易决策"
	/>
	<meta
		name="keywords"
		content="whois查询,域名whois,whois信息查询,域名注册信息,域名到期时间,域名注册商,dns服务器查询,域名whois工具"
	/>
	<meta property="og:title" content="Whois域名查询工具 - 域名注册信息在线查询" />
	<meta
		property="og:description"
		content="专业的Whois域名查询工具,支持国内外域名WHOIS信息查询,提供域名注册商、注册时间、到期时间等详细信息"
	/>
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	<meta property="og:url" content={canonicalUrl} />
	<meta name="twitter:card" content="summary_large_image" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'Whois域名查询工具',
		description:
			'专业的Whois域名查询工具,支持国内外域名WHOIS信息查询,提供域名注册商、注册时间、到期时间、DNS服务器等详细信息',
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Any',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		featureList: 'Whois查询,域名注册信息,域名到期时间,域名注册商查询,DNS服务器查询',
		about: [
			{
				'@type': 'Thing',
				name: 'Whois查询',
				description:
					'通过Whois协议查询域名的注册信息,包括注册商、注册时间、到期时间、DNS服务器等技术服务。'
			},
			{
				'@type': 'Thing',
				name: '域名管理',
				description:
					'通过Whois查询结果,域名持有者可以了解域名到期时间,及时续费避免域名被释放或高价赎回。'
			}
		]
	})}</script>`}
</svelte:head>

<PageTitle h1="Whois 域名查询" sub="查询域名注册信息、注册商、到期时间等 Whois 数据" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={tmpdomain}
			placeholder="请输入域名（如：example.com）"
			onkeydown={(e) => e.key === 'Enter' && queryWhois()}
		/>
		<Button class={BUTTON_CLASS} disabled={loading} onclick={queryWhois}>
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
						<td class="table-label">域名</td>
						<td class="table-value">{result.domain}</td>
					</tr>
					{#if result.status?.length}
						<tr>
							<td class="table-label">域名状态</td>
							<td class="table-value">
								{#each result.status as s, i (i)}
									<span class="status-code {getStatusClass(s)}" style="margin-right: 4px;">{s}</span>
								{/each}
							</td>
						</tr>
					{/if}
					{#if result.registrar?.name}
						<tr>
							<td class="table-label">注册商</td>
							<td class="table-value">
								{result.registrar.name}
								{#if result.registrar.ianaId}
									<span style="color: #999; font-size: 0.9em;">(IANA ID: {result.registrar.ianaId})</span>
								{/if}
							</td>
						</tr>
					{/if}
					{#if result.registrant && Object.keys(result.registrant).length}
						<tr>
							<td class="table-label">注册人</td>
							<td class="table-value">
								{#if result.registrant.name}<div>{result.registrant.name}</div>{/if}
								{#if result.registrant.org}<div>{result.registrant.org}</div>{/if}
								{#if result.registrant.phone}<div>{result.registrant.phone}</div>{/if}
								{#if result.registrant.email}<div>{result.registrant.email}</div>{/if}
								{#if result.registrant.province}<div>{result.registrant.province}</div>{/if}
								{#if result.registrant.contactUri}
									<a href={result.registrant.contactUri} target="_blank" rel="noreferrer">查看注册信息</a>
								{/if}
							</td>
						</tr>
					{/if}
					{#if result.technical && Object.keys(result.technical).length}
						<tr>
							<td class="table-label">技术联系人</td>
							<td class="table-value">
								{#if result.technical.name}<div>{result.technical.name}</div>{/if}
								{#if result.technical.org}<div>{result.technical.org}</div>{/if}
								{#if result.technical.phone}<div>{result.technical.phone}</div>{/if}
								{#if result.technical.email}<div>{result.technical.email}</div>{/if}
							</td>
						</tr>
					{/if}
					{#if result.abuseContact && Object.keys(result.abuseContact).length}
						<tr>
							<td class="table-label">abuse 联系人</td>
							<td class="table-value">
								{#if result.abuseContact.name}<div>{result.abuseContact.name}</div>{/if}
								{#if result.abuseContact.phone}<div>{result.abuseContact.phone}</div>{/if}
								{#if result.abuseContact.email}<div>{result.abuseContact.email}</div>{/if}
							</td>
						</tr>
					{/if}
					{#if result.dates?.registration}
						<tr>
							<td class="table-label">注册时间</td>
							<td class="table-value">{formatDate(result.dates.registration)}</td>
						</tr>
					{/if}
					{#if result.dates?.expiration}
						<tr>
							<td class="table-label">到期时间</td>
							<td class="table-value">{formatDate(result.dates.expiration)}</td>
						</tr>
					{/if}
					{#if result.dates?.lastChanged}
						<tr>
							<td class="table-label">最后修改</td>
							<td class="table-value">{formatDate(result.dates.lastChanged)}</td>
						</tr>
					{/if}
					{#if result.nameservers?.length}
						<tr>
							<td class="table-label">DNS 服务器</td>
							<td class="table-value">
								{#each result.nameservers as ns (ns)}
									<div>{ns}</div>
								{/each}
							</td>
						</tr>
					{/if}
					{#if result.whoisServer}
						<tr>
							<td class="table-label">Whois 服务器</td>
							<td class="table-value">{result.whoisServer}</td>
						</tr>
					{/if}
				</tbody>
			</table>
		</div>
	{/if}

	<blockquote>
		数据来源：后端 WHOIS 服务，通过 IANA 注册局 bootstrap 文件自动匹配对应 WHOIS 服务器。<br />
		部分注册局对注册人信息做了隐私保护，将显示为 "Redacted for Privacy"。<br />
		如需查看完整注册信息，可点击 "查看注册信息" 链接跳转到注册局官网查询。<br />
		访客IP: {visitorIP() || '获取中...'} 您的网络{visitorIsIPv6() ? 'IPv6' : 'IPv4'}优先<br />
	</blockquote>

	{#if data.doc}
		<div class="markdown">
			{@html data.doc}
		</div>
	{/if}
</div>

<style>
	.status-code {
		display: inline-block;
		margin-bottom: 2px;
	}
</style>
