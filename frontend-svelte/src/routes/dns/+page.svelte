<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { config } from '#lib/config/index.ts';
	import { queryAllNodes, type NodeRow } from '#lib/node-pool.ts';
	import { visitorIP, visitorIsIPv6 } from '#lib/visitor-ip.svelte.ts';
	import { formatTime } from '#lib/tools.ts';
	import { INPUT_CLASS, BUTTON_CLASS, SELECT_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import PageTitle from '#lib/components/PageTitle.svelte';

	let { data } = $props();

	interface DNSResult {
		record?: string[];
		duration?: number;
		ttl?: number;
	}

	const recordTypes = [
		{ value: 'a', label: 'A 记录' },
		{ value: 'aaaa', label: 'AAAA 记录' },
		{ value: 'cname', label: 'CNAME 记录' },
		{ value: 'mx', label: 'MX 记录' },
		{ value: 'ns', label: 'NS 记录' },
		{ value: 'txt', label: 'TXT 记录' },
		{ value: 'srv', label: 'SRV 记录' },
		{ value: 'caa', label: 'CAA 记录' },
		{ value: 'ptr', label: 'PTR 记录' }
	];

	/** 三栈平铺（旧站顺序：双栈 → IPv4 → IPv6） */
	const nodes = [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4, ...config.APIBaseURL.IPv6];

	const canonicalUrl = `${config.siteUrl}dns`;

	let tmpDomain = $state('www.zakoflare.com');
	let recordType = $state('a');
	let isloading = $state(false);
	/** 结果表里「类型」列显示的是**发起查询时**选的类型，不是下拉框当前值（旧站同此） */
	let nowRecordType = $state('');
	let rows = $state<NodeRow<DNSResult>[]>([]);

	async function queryDNS() {
		const domain = tmpDomain;
		isloading = true;

		// 换一个全新的 rows 数组：上一次未落地的结果写的是旧数组，不会再影响界面
		rows = nodes.map((node) => ({ label: node.label ?? '', loading: true }));

		const current = { rows, type: recordType };
		nowRecordType = recordType;

		await queryAllNodes<DNSResult>(
			nodes,
			(id) => `/middleware/${id}/dns/${recordType}/${encodeURIComponent(domain)}`,
			current.rows,
			(e) => (e instanceof Error ? e.message : '请求失败')
		);
		isloading = false;
	}

	// 行内「数据格是否该渲染」判定。
	//
	// 结果表共 6 列（服务器 / 类型 / 记录 / 记录数 / 耗时 / TTL）；后三列只在真正拿到结果时
	// 才有意义。加载中与失败两种状态由「记录」那格用 colspan=4 横跨到表尾，若此时再渲染后三格，
	// 整行就会变成 2+4+3=9 格 —— 表格按最大列数排版，凭空多出 3 列，表头只剩 6 列、
	// 右侧留出空档，错误行也跟着错位。
	function hasData(row: NodeRow<DNSResult>): boolean {
		return !row.loading && !row.error;
	}

	onMount(() => {
		const domainParam = page.url.searchParams.get('domain');
		const typeParam = page.url.searchParams.get('type');
		if (domainParam) tmpDomain = domainParam;
		if (typeParam && recordTypes.some((t) => t.value === typeParam)) recordType = typeParam;
		if (domainParam) queryDNS();
	});
</script>

<svelte:head>
	<title>DNS查询工具 | 多节点域名解析检测 | {config.siteName}</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content="专业的多节点DNS查询工具,支持A记录、AAAA记录、CNAME记录、MX记录、NS记录、TXT记录、SRV记录、CAA记录等多种DNS解析记录查询,提供全国多节点并发检测,快速返回DNS解析结果,助力域名解析问题排查与优化"
	/>
	<meta
		name="keywords"
		content="dns查询,dns解析,域名解析,a记录查询,aaaa记录,cname记录,mx记录,ns记录,txt记录,srv记录,dns服务器,域名dns检测"
	/>
	<meta property="og:title" content="DNS查询工具 - 多节点域名解析记录检测" />
	<meta property="og:description" content="多节点DNS查询工具,支持多种DNS记录类型查询,快速检测域名解析状态" />
	<meta property="og:image" content={`${config.siteUrl}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'DNS查询与解析检测工具',
		description:
			'专业的多节点DNS查询工具，支持A、AAAA、CNAME、MX、NS、TXT、SRV、CAA等多种记录类型，全国多节点并发检测。',
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<PageTitle h1="DNS查询" sub="多节点 DNS 查询，检测域名解析记录" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={tmpDomain}
			placeholder="请输入域名（如：example.com）"
			onkeydown={(e) => e.key === 'Enter' && queryDNS()}
		/>
		<Select.Root type="single" bind:value={recordType}>
			<Select.Trigger class={SELECT_CLASS}>
				<Select.Value placeholder="记录类型" />
			</Select.Trigger>
			<Select.Content>
				{#each recordTypes as item (item.value)}
					<Select.Item value={item.value} label={item.label}>{item.label}</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
		<Button class={BUTTON_CLASS} disabled={isloading} onclick={queryDNS}>
			{isloading ? '查询中…' : '查询'}
		</Button>
	</div>

	<div class="result-section">
		{#if rows.length > 0}
			<table class="result-table">
				<thead>
					<tr>
						<th class="table-header">服务器</th>
						<th class="table-header">类型</th>
						<th class="table-header">记录</th>
						<th class="table-header">记录数</th>
						<th class="table-header">耗时</th>
						<th class="table-header">TTL (S)</th>
					</tr>
				</thead>
				<tbody>
					{#each rows as row (row.label)}
						<tr>
							<td class="table-label">{row.label}</td>
							<td class="table-value">{nowRecordType.toUpperCase() || '--'}</td>
							{#if row.loading}
								<td class="table-value" colspan="4" style="text-align: left;">加载中...</td>
							{:else if row.error}
								<td class="table-value" colspan="4" style="color: #F56C6C;">{row.error}</td>
							{:else}
								<td class="table-value">
									{#if row.data?.record}
										{#each row.data.record.slice(0, 5) as ip, index (index)}
											<div class="ip-address"><span>{ip}</span></div>
										{/each}
									{:else if row.data?.record?.length === 0}
										<span class="status-code" style="color: #F56C6C; background: #fef0f0;">解析失败</span>
									{/if}
								</td>
							{/if}

							<!-- 后三格必须与「记录」格的 colspan 互斥（见 hasData 注释） -->
							{#if hasData(row)}
								<td class="table-value">{row.data?.record?.length || 0}</td>
								<td class="table-value">{formatTime(row.data?.duration)}</td>
								<td class="table-value">{row.data?.ttl}</td>
							{/if}
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</div>

	<blockquote>
		<span class="quate">A</span> 将域名指向一个 IPv4 地址，如 <span class="quate">106.55.75.123</span
		>。 A记录归属地是柠檬超绝大杂烩免费接口，随便用。<br />

		<span class="quate">AAAA</span> 将域名指向一个 IPv6 地址，如 <span class="quate"
			>2402:4e00:1013:e500:0:9671:f018:4947</span
		>。同一个主机名可以同时解析到 IPv4(A记录)地址 和 IPv6(AAAA 记录)地址上，当只有IPv4 地址的用户会解析到 IPv4
		地址，一般情况下有 IPv6 地址的用户会优先解析到 IPv6 地址。<br />

		<span class="quate">CNAME</span> 将域名指向另一个域名地址，与其保持相同解析，如 <span class="quate"
			>ipw.wsmdn.top</span
		> 别名到 <span class="quate">ipw.wsmdn.top.eo.dnse1.com</span>.<br />

		<span class="quate">MX</span> 用于邮件服务器，一般由邮件注册商提供，如 <span class="quate">mxbiz1.qq.com</span
		>。如果邮箱格式为 test@<span class="quate">wsmdn.top</span> 则输入 <span class="quate">wsmdn.top</span
		> 查询。如果邮箱格式为 test@<span class="quate">mail.wsmdn.top</span>则输入<span class="quate"
			>mail.wsmdn.top</span
		>查询。推荐2个免费的企业邮箱：腾讯企业邮、网易免费企业邮。<br />

		<span class="quate">TXT</span> 附加文本信息，常用于域名所有权验证，如在申请 HTTPS 证书时需要增加记录、<br />

		<span class="quate">PTR</span> IP 的反向解析记录，例如 <span class="quate">159.75.190.197</span> 反解析到
		<span class="quate">wsmdn.top</span>，一般用于提升自建域名邮件服务器的可信度，可提单找云服务商添加。<br />

		<span class="quate">NS</span> 域名的 DNS 服务器地址，例如 <span class="quate">ns3.dnsv2.com</span
		>，推荐 华为云DNS.<br />

		<a href="/ipv6webcheck" target="_blank">网站开启IPv6检测</a> | <a href="/ssl" target="_blank"
			>SSL证书在线检查</a
		><br />

		访客IP: {visitorIP()}，您的网络 {visitorIsIPv6() ? 'IPv6' : 'IPv4'} 访问优先<br />
	</blockquote>
</div>
