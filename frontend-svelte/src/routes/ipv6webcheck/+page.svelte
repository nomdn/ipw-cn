<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import CircleCheckFilled from '#lib/components/icons/CircleCheckFilled.svelte';
	import CircleCloseFilled from '#lib/components/icons/CircleCloseFilled.svelte';
	import InfoFilled from '#lib/components/icons/InfoFilled.svelte';
	import Position from '#lib/components/icons/Position.svelte';
	import { config } from '#lib/config/index.ts';
	import { queryNodePool } from '#lib/node-pool.ts';
	import { extractHost, getStatusCodeClass, formatTime, formatSize, formatSpeed } from '#lib/tools.ts';
	import { INPUT_CLASS, BUTTON_CLASS } from '#lib/ui-classes.ts';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import PageTitle from '#lib/components/PageTitle.svelte';

	interface PerformanceCheckItem {
		host_record: string;
		http_status_code: number;
		https_status_code: number;
		dns_lookup_time: number;
		tcp_connect_time: number;
		http_connect_time: number;
		first_byte_time: number;
		total_time: number;
		page_size: number;
		download_speed: number;
		is_reachable: boolean;
	}
	interface PerformanceCheckResponse {
		ipv4?: PerformanceCheckItem;
		ipv6?: PerformanceCheckItem;
	}

	const apiList = config.APIBaseURL.DualStack;
	const siteUrlWithSlash = config.siteUrl.replace(/\/$/, '') + '/';
	const canonicalUrl = siteUrlWithSlash + 'ipv6webcheck';

	let tmpDomain = $state('https://www.zakoflare.com');
	let testDomain = $state('');
	let loading = $state(false);
	let error = $state('');
	let result = $state<PerformanceCheckResponse | null>(null);

	/** 徽标代码样例（旧站模板里内联了 13 遍同样的 HTML，这里两份函数消化掉） */
	function badgeCode(file: string): string {
		const host = extractHost(testDomain);
		return `<a href="${siteUrlWithSlash}ipv6webcheck/?site=${host}" title="本站支持 IPv6访问" target='_blank'><img style='display:inline-block;vertical-align:middle' alt="本站支持 IPv6访问" src="${siteUrlWithSlash}${file}"></a>`;
	}
	/** 证书徽标（lite 与完整版）的代码样例：文件名不同、其余一致 */
	const LITE_BADGES = [
		'ipv6-certified-lite-s1.svg',
		'ipv6-certified-lite-s2.svg',
		'ipv6-certified-lite-s3.svg',
		'ipv6-certified-lite-s4.svg',
		'ipv6-certified-lite-s5.svg',
		'ipv6-certified-lite-s6.svg'
	];
	const CERT_BADGES = [
		'ipv6-certified-s1.svg',
		'ipv6-certified-s2.svg',
		'ipv6-certified-s3.svg',
		'ipv6-certified-s4.svg',
		'ipv6-certified-s5.svg',
		'ipv6-certified-s6.svg'
	];
	const SITE_BADGES = ['ipv6-s1.svg', 'ipv6-s2.svg', 'ipv6-s3.svg', 'ipv6-s4.svg', 'ipv6-s5.svg', 'ipv6-s6.svg'];

	async function checkWeb() {
		// testDomain 是完整 URL（含 https:// 与路径斜杠），服务端 [...slug] 路由依赖
		// 分段重组协议部分，所以这里刻意不做 encodeURIComponent
		testDomain = tmpDomain;
		error = '';
		result = null;
		loading = true;

		const outcome = await queryNodePool<PerformanceCheckResponse>(
			apiList,
			(id) => `/middleware/${id}/detail/${testDomain}`,
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

	onMount(() => {
		const site = page.url.searchParams.get('site');
		if (site) {
			tmpDomain = site;
			checkWeb();
		}
	});
</script>

<svelte:head>
	<title>IPv6网站检测工具 | IPv6访问支持检查 | {config.siteName}</title>
	<link rel="canonical" href={canonicalUrl} />
	<meta
		name="description"
		content="专业的IPv6网站检测工具,全面检查网站是否支持IPv6访问,提供IPv4和IPv6双栈HTTP/HTTPS状态码、DNS解析时间、TCP连接时间、下载速度等详细对比数据,帮助网站管理员确认IPv6部署状态,提供IPv6徽标认证,推进IPv6规模部署和应用"
	/>
	<meta
		name="keywords"
		content="ipv6网站检测,ipv6访问检测,ipv6支持检查,ipv6双栈检测,ipv6 http检测,ipv6 https检测,ipv6网站认证,ipv6徽标,ipv6部署"
	/>
	<meta property="og:title" content="IPv6网站检测 - IPv6访问支持状态检查与认证" />
	<meta property="og:description" content="全面检测网站IPv6访问支持情况,提供详细性能对比与IPv6徽标认证" />
	<meta property="og:image" content={`${siteUrlWithSlash}favicon.svg`} />
	<meta property="og:type" content="website" />
	{@html `<script type="application/ld+json">${JSON.stringify({
		'@context': 'https://schema.org',
		'@type': 'WebApplication',
		name: 'IPv6网站访问支持检测',
		description:
			'专业的IPv6网站检测工具，检查网站是否支持IPv6访问，提供IPv4和IPv6双栈HTTP/HTTPS状态码、DNS解析时间、TCP连接时间等详细对比数据，提供IPv6徽标认证。',
		url: canonicalUrl,
		applicationCategory: 'DeveloperApplication',
		operatingSystem: 'Web',
		offers: { '@type': 'Offer', price: '0', priceCurrency: 'CNY' },
		provider: { '@type': 'Organization', name: config.siteName }
	})}</script>`}
</svelte:head>

<!-- 两种徽标排版（旧站就是这么分的，别合并）：
     · badgeList  —— 每个徽标各占一行、各自配一段代码（网站徽标 ipv6-s1..s6）；
     · badgeCertRow —— 6 个徽标横排一行、只给第一个配一段代码（认证徽标两组）。 -->
{#snippet badgeList(files: string[])}
	{#each files as file (file)}
		<img src={`/${file}`} alt="" />
		<pre><code>{badgeCode(file)}</code></pre>
	{/each}
{/snippet}

{#snippet badgeCertRow(files: string[])}
	<div class="one-line">
		{#each files as file (file)}
			<img src={`/${file}`} alt="" />
		{/each}
	</div>
	<pre><code>{badgeCode(files[0] ?? '')}</code></pre>
{/snippet}

<PageTitle h1="IPv6网站检测" sub="检查网站是否开启 IPv6 访问，致力于普及IPv6" />

<div class="content">
	<div class="one-line">
		<Input
			class={INPUT_CLASS}
			bind:value={tmpDomain}
			placeholder="请输入域名（如：https://zakoflare.com）"
			onkeydown={(e) => e.key === 'Enter' && checkWeb()}
		/>
		<Button class={BUTTON_CLASS} disabled={loading} onclick={checkWeb}>
			{loading ? '检测中…' : '开始测试'}
		</Button>
	</div>

	{#if error}
		<div class="error-message">{error}</div>
	{/if}

	{#if result && result.ipv4}
		<div class="result-section">
			<table class="result-table">
				<thead>
					<tr>
						<th class="table-header">检测项目</th>
						<th class="table-header">IPv4</th>
						<th class="table-header">IPv6</th>
					</tr>
				</thead>
				<tbody>
					<tr>
						<td class="table-label">主机记录</td>
						<td class="table-value">
							<div class="one-line">
								<a href={`/ipv6?ip=${result.ipv4.host_record}`}>{result.ipv4.host_record}</a>
								<Position class="ak-inline-icon" />
							</div>
						</td>
						<td class="table-value">
							<div class="one-line">
								{#if result.ipv6?.host_record}
									<a href={`/ipv6?ip=${result.ipv6.host_record}`}>{result.ipv6.host_record}</a>
								{:else}
									<span>-</span>
								{/if}
								<Position class="ak-inline-icon" />
							</div>
						</td>
					</tr>
					<tr>
						<td class="table-label">HTTP 状态码</td>
						<td class="table-value">
							<span class="status-code {getStatusCodeClass(result.ipv4.http_status_code)}">
								{result.ipv4.http_status_code}
							</span>
						</td>
						<td class="table-value">
							{#if result.ipv6}
								<span class="status-code {getStatusCodeClass(result.ipv6.http_status_code)}">
									{result.ipv6.http_status_code}
								</span>
							{:else}
								<span>-</span>
							{/if}
						</td>
					</tr>
					<tr>
						<td class="table-label">HTTPS 状态码</td>
						<td class="table-value">
							<span class="status-code {getStatusCodeClass(result.ipv4.https_status_code)}">
								{result.ipv4.https_status_code}
							</span>
						</td>
						<td class="table-value">
							{#if result.ipv6}
								<span class="status-code {getStatusCodeClass(result.ipv6.https_status_code)}">
									{result.ipv6.https_status_code}
								</span>
							{:else}
								<span>-</span>
							{/if}
						</td>
					</tr>
					<tr>
						<td class="table-label">DNS 查询耗时</td>
						<td class="table-value">{formatTime(result.ipv4.dns_lookup_time)}</td>
						<td class="table-value"
							>{result.ipv6?.dns_lookup_time ? formatTime(result.ipv6.dns_lookup_time) : '-'}</td
						>
					</tr>
					<tr>
						<td class="table-label">TCP 连接耗时</td>
						<td class="table-value">{formatTime(result.ipv4.tcp_connect_time)}</td>
						<td class="table-value"
							>{result.ipv6?.tcp_connect_time ? formatTime(result.ipv6.tcp_connect_time) : '-'}</td
						>
					</tr>
					<tr>
						<td class="table-label">HTTP 连接耗时</td>
						<td class="table-value">{formatTime(result.ipv4.http_connect_time)}</td>
						<td class="table-value"
							>{result.ipv6?.http_connect_time ? formatTime(result.ipv6.http_connect_time) : '-'}</td
						>
					</tr>
					<tr>
						<td class="table-label">首字节耗时</td>
						<td class="table-value">{formatTime(result.ipv4.first_byte_time)}</td>
						<td class="table-value"
							>{result.ipv6?.first_byte_time ? formatTime(result.ipv6.first_byte_time) : '-'}</td
						>
					</tr>
					<tr>
						<td class="table-label">总耗时</td>
						<td class="table-value">{formatTime(result.ipv4.total_time)}</td>
						<td class="table-value">{result.ipv6?.total_time ? formatTime(result.ipv6.total_time) : '-'}</td>
					</tr>
					<tr>
						<td class="table-label">页面大小</td>
						<td class="table-value">{formatSize(result.ipv4.page_size)}</td>
						<td class="table-value">{result.ipv6?.page_size ? formatSize(result.ipv6.page_size) : '-'}</td>
					</tr>
					<tr>
						<td class="table-label">下载速度</td>
						<td class="table-value">{formatSpeed(result.ipv4.download_speed)}</td>
						<td class="table-value"
							>{result.ipv6?.download_speed ? formatSpeed(result.ipv6.download_speed) : '-'}</td
						>
					</tr>
				</tbody>
			</table>
		</div>
	{/if}

	{#if result && result.ipv4 && result.ipv6 && result.ipv4.is_reachable && result.ipv6.is_reachable}
		<div>
			<h3>
				结论：<CircleCheckFilled class="ak-inline-icon text-[lightgreen]" />
				网站{extractHost(testDomain)} 支持IPv6访问
			</h3>
			<p>
				<InfoFilled class="ak-inline-icon text-[lightgreen]" />
				请把下方代码贴到网站底部，把这个好消息告诉你的用户，以便用户核验。
			</p>
			{@render badgeList(SITE_BADGES)}
			{@render badgeCertRow(LITE_BADGES)}
			{@render badgeCertRow(CERT_BADGES)}
			<p>提示：修改IPv6徽标文件名，可修改对应样式</p>
		</div>
	{:else if result && result.ipv4 && result.ipv6 && result.ipv4.is_reachable && !result.ipv6.is_reachable}
		<div>
			<h3>
				结论：<CircleCloseFilled class="ak-inline-icon text-[red]" />
				网站{extractHost(testDomain)} 不支持IPv6访问
			</h3>
			<h2>国家正在支持IPv6发展，我建议你赶紧想办法给IPv6适配</h2>
			<img src="/jingya.jpg" alt="不支持 IPv6 访问示意" width="480" height="270" />
		</div>
	{:else if result && !result.ipv6?.is_reachable && !result.ipv4?.is_reachable}
		<div>
			<h3>
				结论：<CircleCloseFilled class="ak-inline-icon text-[red]" />
				网站{extractHost(testDomain)} 不可达
			</h3>
			<h2>...</h2>
			<img src="/jingya.jpg" alt="网站不可达示意" width="480" height="270" />
		</div>
	{:else if result && result.ipv4 && result.ipv6 && !result.ipv4.is_reachable && result.ipv6.is_reachable}
		<div>
			<h3>
				结论：<CircleCloseFilled class="ak-inline-icon text-[red]" />
				网站{extractHost(testDomain)} 不支持IPv4访问
			</h3>
			<h2>
				全球来看，越南、捷克等部分国家已提出了单栈发展目标。尽管我国网络IPv6流量占比大幅提升，但网络中仍存在大量的NAT（网络地址转换）设备和IPv4依赖应用。
			</h2>
			<h2 style="text-align: right;">----中国《IPv6发展报告》2026</h2>
			<img src="/jingya.jpg" alt="不支持 IPv4 访问示意" width="480" height="270" />
		</div>
	{/if}

	<blockquote>
		网站不支持 IPv6可能原因：<br />
		<br />
		1. 网站所在服务器未开启 IPv6，请参考 <a href="/doc/server/website_enable_ipv6" target="_blank"
			>网站开启 IPv6 的三种方式</a><br />
		2. 网站所在服务器已开启 IPv6，但防火墙未对源地址是 IPv6 地址(::/0)的 443（HTTPS）端口开放访问<br />
		3. 网站所在服务器已开启 IPv6，但未开启SSL证书，请参考 <a href="/doc/server/nginx_ipv6" target="_blank"
			>Nginx 开启 IPv6 SSL</a
		><br />
	</blockquote>
</div>
