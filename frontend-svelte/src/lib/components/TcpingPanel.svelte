<script lang="ts">
	/**
	 * TCPing 测试面板（/tcping 与 /ipv6tcping 共用）。
	 *
	 * 旧站 tcping.vue 与 ipv6tcping.vue 是两份 280 行的近似复制：差异只有
	 *   ① 取哪个字段（`server.ipv4` vs `server.ipv6`）、
	 *   ② 节点池是 DualStack+IPv4 还是 DualStack+IPv6。
	 * 其余（表格 8 列、丢包率着色、加载/失败/空三态）逐字相同，所以这里参数化成
	 * 一个组件，两个页面各自只留标题、meta 与 blockquote。
	 */
	import { onMount, untrack } from 'svelte';
	import { page } from '$app/state';
	import { config } from '#lib/config/index.ts';
	import { queryAllNodes, type NodeRow } from '#lib/node-pool.ts';
	import { extractHost } from '#lib/tools.ts';
	import { INPUT_CLASS, INPUT_NARROW_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';

	interface TCPingStats {
		ip: string;
		port: string;
		sent: number;
		success: number;
		loss_rate: number;
		max_rtt: number;
		min_rtt: number;
		avg_rtt: number;
	}
	interface TCPingResponse {
		ipv4?: TCPingStats;
		ipv6?: TCPingStats;
	}

	let { family }: { family: 'ipv4' | 'ipv6' } = $props();

	// family 在组件生命周期里不会变（两个页面各传一个固定值），所以这里只取一次；
	// untrack 用于声明"确实只按初始值算"这一意图，避免 state_referenced_locally 警告。
	const nodes = untrack(() =>
		family === 'ipv4'
			? [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv4]
			: [...config.APIBaseURL.DualStack, ...config.APIBaseURL.IPv6]
	);

	type Row = NodeRow<TCPingResponse>;

	let tmpDomain = $state('www.zakoflare.com');
	let port = $state('80');
	let loading = $state(false);
	// 一开始就把节点列出来（旧站挂载后才填，SSR 出来是空表；这里顺带把首屏补上）
	let rows = $state<Row[]>(nodes.map((n) => ({ label: n.label ?? '', loading: false })));

	function statsOf(row: Row): TCPingStats | undefined {
		return family === 'ipv4' ? row.data?.ipv4 : row.data?.ipv6;
	}

	async function run() {
		const host = extractHost(tmpDomain);
		if (!host) return;

		loading = true;
		rows = nodes.map((n) => ({ label: n.label ?? '', loading: true }));

		await queryAllNodes<TCPingResponse>(
			nodes,
			(id) => `/middleware/${id}/tcping/${encodeURIComponent(host)}?port=${encodeURIComponent(port)}`,
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
		placeholder="请输入域名（如：example.com）"
		onkeydown={(e) => e.key === 'Enter' && run()}
	/>
	<Input
		class={INPUT_NARROW_CLASS}
		bind:value={port}
		placeholder="端口号（默认 80）"
		onkeydown={(e) => e.key === 'Enter' && run()}
	/>
	<Button class={BUTTON_CLASS} disabled={loading} onclick={run}>
		{loading ? '测试中…' : '开始测试'}
	</Button>
</div>

<div class="result-section">
	<table class="result-table">
		<thead>
			<tr>
				<th class="table-header">服务器</th>
				<th class="table-header">解析 IP</th>
				<th class="table-header">发送包</th>
				<th class="table-header">接收包</th>
				<th class="table-header">丢包率(%)</th>
				<th class="table-header">最长时间(ms)</th>
				<th class="table-header">最短时间(ms)</th>
				<th class="table-header">平均时间(ms)</th>
			</tr>
		</thead>
		<tbody>
			{#each rows as row (row.label)}
				{#if statsOf(row)}
					{@const stats = statsOf(row)!}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value">{stats.ip || '-'}</td>
						<td class="table-value">{stats.sent}</td>
						<td class="table-value">{stats.success}</td>
						<td class="table-value">
							<span class={(stats.loss_rate ?? 0) > 0 ? 'loss-warning' : 'loss-ok'}>
								{stats.loss_rate?.toFixed(1) ?? '-'}
							</span>
						</td>
						<td class="table-value">{stats.max_rtt?.toFixed(2) ?? '-'}</td>
						<td class="table-value">{stats.min_rtt?.toFixed(2) ?? '-'}</td>
						<td class="table-value">{stats.avg_rtt?.toFixed(2) ?? '-'}</td>
					</tr>
				{:else if row.loading}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value" colspan="7">加载中...</td>
					</tr>
				{:else if row.error}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value error-text" colspan="7">{row.error}</td>
					</tr>
				{:else}
					<tr>
						<td class="table-label">{row.label}</td>
						<td class="table-value" colspan="7">-</td>
					</tr>
				{/if}
			{/each}
		</tbody>
	</table>
</div>

<style>
	.result-table .table-label {
		width: 200px;
	}
</style>
