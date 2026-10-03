<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { config } from '#lib/config/index.ts';
	import { queryAllNodes, type NodeRow } from '#lib/node-pool.ts';
	import { visitorIP, visitorIsIPv6 } from '#lib/visitor-ip.svelte.ts';
	import { INPUT_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import PageTitle from '#lib/components/PageTitle.svelte';

	interface DNSSECResult {
		domain: string;
		enabled: boolean;
		valid: boolean;
		has_rrsig: boolean;
		has_dnskey: boolean;
		has_ds: boolean;
		algorithm: number;
		key_tag: number;
		signer_name: string;
		validation: string;
		duration: number;
	}

	/** 三栈平铺（旧站顺序：双栈 → IPv4 → IPv6） */
	const nodes = [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4, ...config.APIBaseURL.IPv6];
	const canonicalUrl = `${config.siteUrl}dnssec`;

	let tmpDomain = $state('www.zakoflare.com');
	let loading = $state(false);
	let rows = $state<NodeRow<DNSSECResult>[]>([]);

	function getValidationClass(r: DNSSECResult): string {
		if (r.valid) return 'status-success';
		if (r.has_dnskey || r.has_rrsig) return 'status-warning';
		return 'status-error';
	}

	function formatDuration(ms?: number): string {
		return typeof ms === 'number' && isFinite(ms) ? ms.toFixed(2) + ' ms' : '-';
	}

	async function checkDNSSEC() {
		const domain = tmpDomain.trim();
		if (!domain) return;

		loading = true;
		rows = nodes.map((node) => ({ label: node.label ?? '', loading: true }));

		await queryAllNodes<DNSSECResult>(
			nodes,
			(id) => `/middleware/${id}/dnssec/${encodeURIComponent(domain)}`,
			rows,
			(e) => (e instanceof Error ? e.message : '请求失败')
		);
		loading = false;
	}

	onMount(() => {
		const domainParam = page.url.searchParams.get('domain');
		if (domainParam) {
			tmpDomain = domainParam;
			checkDNSSEC();
		}
	});
</script>

<svelte:head>
	<title>DNSSEC查询 | 域名DNSSEC验证检测 | {config.siteName}</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content="专业的DNSSEC查询工具,支持域名DNSSEC验证检测,检查域名是否启用DNSSEC签名,验证DNSKEY、RRSIG、DS记录链式信任关系,检测域名DNS数据完整性和真实性,防止DNS欺骗和中间人攻击,保障域名解析安全"
	/>
	<meta
		name="keywords"
		content="dnssec查询,dnssec验证,dnskey,rrsig,ds记录,域名安全,链式信任,dns欺骗防护,域名解析安全,dns签名验证"
	/>
	<meta property="og:title" content="DNSSEC查询工具 - 域名DNSSEC验证与安全检测" />
	<meta
		property="og:description"
		content="DNSSEC验证检测工具,检查域名DNSSEC签名状态,验证DNSKEY、RRSIG、DS记录链式信任关系"
	/>
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'DNSSEC验证查询工具',
		description:
			'专业的DNSSEC验证查询工具，检测域名DNSSEC签名状态，验证DNSKEY、RRSIG、DS记录链式信任关系，保障域名解析安全。',
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="DNSSEC 验证查询" sub="检测域名 DNSSEC 签名状态和链式信任验证" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={tmpDomain}
			placeholder="请输入域名（如：example.com）"
			onkeydown={(e) => e.key === 'Enter' && checkDNSSEC()}
		/>
		<Button class={BUTTON_CLASS} disabled={loading} onclick={checkDNSSEC}>
			{loading ? '验证中…' : '验证'}
		</Button>
	</div>

	{#if rows.some((r) => r.data)}
		<div class="result-section">
			<table class="result-table">
				<thead>
					<tr>
						<th class="table-header">DNS 服务器</th>
						<th class="table-header">DNSSEC</th>
						<th class="table-header">DNSKEY</th>
						<th class="table-header">RRSIG</th>
						<th class="table-header">DS</th>
						<th class="table-header">验证结果</th>
						<th class="table-header">耗时</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as row (row.label)}
						{#if row.data}
							<tr>
								<td class="table-label">{row.label}</td>
								<td class="table-value">
									<span class="status-code {row.data.enabled ? 'status-success' : 'status-error'}">
										{row.data.enabled ? '已启用' : '未启用'}
									</span>
								</td>
								<td class="table-value">
									<span class="status-code {row.data.has_dnskey ? 'status-success' : 'status-error'}">
										{row.data.has_dnskey ? '有' : '无'}
									</span>
								</td>
								<td class="table-value">
									<span class="status-code {row.data.has_rrsig ? 'status-success' : 'status-error'}">
										{row.data.has_rrsig ? '有' : '无'}
									</span>
								</td>
								<td class="table-value">
									<span class="status-code {row.data.has_ds ? 'status-success' : 'status-error'}">
										{row.data.has_ds ? '有' : '无'}
									</span>
								</td>
								<td class="table-value">
									<span class="status-code {getValidationClass(row.data)}">
										{row.data.valid ? '验证通过' : '验证失败'}
									</span>
								</td>
								<td class="table-value">{formatDuration(row.data.duration)}</td>
							</tr>
						{:else if row.loading}
							<tr>
								<td class="table-label">{row.label}</td>
								<td class="table-value" colspan="6">加载中...</td>
							</tr>
						{:else if row.error}
							<tr>
								<td class="table-label">{row.label}</td>
								<td class="table-value error-text" colspan="6">{row.error}</td>
							</tr>
						{:else}
							<tr>
								<td class="table-label">{row.label}</td>
								<td class="table-value" colspan="6">-</td>
							</tr>
						{/if}
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<blockquote>
		DNSSEC 原理介绍即将上线<br />
		DNSSEC 通过 DNSKEY、RRSIG、DS 记录为 DNS 数据提供完整性验证，防止 DNS 欺骗攻击。<br />
		如果显示 "验证失败"，可能原因：域名未配置 DNSSEC、DNSKEY 密钥不匹配、DS 记录缺失等。<br />
		<a href="/dns" target="_blank">DNS 解析查询</a> | <a href="/whois" target="_blank">Whois 查询</a><br />
		访客IP: {visitorIP() || '获取中...'}，您的网络{visitorIsIPv6() ? 'IPv6' : 'IPv4'}优先<br />
	</blockquote>
</div>

<style>
	.result-table .table-label {
		width: 180px;
	}
</style>
