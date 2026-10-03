<script lang="ts">
	/**
	 * 网站测速面板（/speedtest 与 /ipv6speedtest 共用）。
	 *
	 * 与 TcpingPanel 同理：旧站两份页面除了
	 *   ① 走 `/speed/v4/` 还是 `/speed/v6/`、② 节点池是否含 IPv6，
	 * 其余（9 列表格、状态码着色、加载/失败/空三态）完全一致。
	 */
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/state';
	import { config } from '#lib/config/index.ts';
	import { queryAllNodes, type NodeRow } from '#lib/node-pool.ts';
	import { extractHost, formatTime, formatSpeed, formatSize, getStatusCodeClass } from '#lib/tools.ts';
	import { INPUT_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';

	interface SpeedResult {
		host_record?: string;
		http_status_code?: number;
		https_status_code?: number;
		total_time?: number;
		dns_lookup_time?: number;
		first_byte_time?: number;
		page_size?: number;
		download_speed?: number;
	}

	let { family }: { family: 'ipv4' | 'ipv6' } = $props();

	// family 在组件生命周期里不会变（两个页面各传一个固定值），所以这里只取一次；
	// untrack 用于声明"确实只按初始值算"这一意图，避免 state_referenced_locally 警告。
	const nodes = untrack(() =>
		family === 'ipv4'
			? [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4]
			: [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv6]
	);

	type Row = NodeRow<SpeedResult>;

	let tmpDomain = $state('https://www.zakoflare.com');
	let loading = $state(false);
	let rows = $state<Row[]>(nodes.map((n) => ({ label: n.label ?? '', loading: false })));

	async function run() {
		const host = extractHost(tmpDomain);
		loading = true;
		rows = nodes.map((n) => ({ label: n.label ?? '', loading: true }));

		await queryAllNodes<SpeedResult>(
			nodes,
			(id) => `/middleware/${id}/speed/${family === 'ipv4' ? 'v4' : 'v6'}/${encodeURIComponent(host)}`,
			rows,
			(e) => (e instanceof Error ? e.message : '请求失败')
		);
		loading = false;
	}

	onMount(() => {
		const site = page.url.searchParams.get('site');
		if (site) {
			tmpDomain = site;
			run();
		}
	});
</script>

<div class="one-line">
	<Input
		class={INPUT_CLASS}
		bind:value={tmpDomain}
		placeholder="请输入域名（如：https://zakoflare.com）"
		onkeydown={(e) => e.key === 'Enter' && run()}
	/>
	<Button class={BUTTON_CLASS} disabled={loading} onclick={run}>
		{loading ? '测速中…' : '开始测试'}
	</Button>
</div>

<div class="result-section">
	<table class="result-table">
		<thead>
			<tr>
				<th class="table-header">测速服务器</th>
				<th class="table-header">解析 IP</th>
				<th class="table-header">HTTP状态码</th>
				<th class="table-header">HTTPS状态码</th>
				<th class="table-header">总时间</th>
				<th class="table-header">解析时间</th>
				<th class="table-header">HTTP连接</th>
				<th class="table-header">网页大小</th>
				<th class="table-header">下载速度</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.label)}
				{#if row.error}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value" colspan="8">
							<span class="status-code status-error">{row.error}</span>
						</td>
					</tr>
				{:else if row.loading}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value" colspan="8">测速中...</td>
					</tr>
				{:else if row.data}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value">{row.data?.host_record}</td>
						<td class="table-value">
							<span class="status-code {getStatusCodeClass(row.data?.http_status_code)}">
								{row.data?.http_status_code}
							</span>
						</td>
						<td class="table-value">
							<span class="status-code {getStatusCodeClass(row.data?.https_status_code)}">
								{row.data?.https_status_code}
							</span>
						</td>
						<td class="table-value">{formatTime(row.data?.total_time)}</td>
						<td class="table-value">{formatTime(row.data?.dns_lookup_time)}</td>
						<td class="table-value">{formatTime(row.data?.first_byte_time)}</td>
						<td class="table-value">{formatSize(row.data?.page_size)}</td>
						<td class="table-value">{formatSpeed(row.data?.download_speed)}</td>
					</tr>
				{:else}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value" colspan="8">-</td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</table>
</div>

<style>
	.result-table .table-header {
		font-size: 1.05em;
	}
</style>
