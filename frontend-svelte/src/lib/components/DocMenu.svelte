<script lang="ts">
	/**
	 * 文档导航菜单（全站唯一实现，结构别在别处再抄一份）
	 *
	 * - 宽屏：由 /doc 与 /doc/** 页面直接渲染成左侧固定侧边栏（页面传 class 控制宽度/粘性）
	 * - 窄屏：页面那边把侧边栏整个隐藏，改由顶栏抽屉挂同一份本组件，
	 *   这样手机上不需要两个菜单，也不至于没入口进文档。
	 *
	 * 旧站用 element-plus 的 el-menu/el-sub-menu 实现，观感（56px 行高、左右 20px、
	 * 14px 字、#303133、选中转品牌绿、hover 底色 light-9、分组展开是高度过渡）全部落在
	 * site.css 的 `.ak-doc-*` 里，数值与 element-plus 源码逐条对应，别在这里用 Tailwind 拼。
	 *
	 * 展开态自己管（Svelte 5 的 $state），不再受 EP 的 default-openeds 影响
	 * （旧站还得靠 `:key` 强制重挂才能让抽屉里的分组重新初始化）。
	 */
	import { page } from '$app/state';
	import { untrack } from 'svelte';
	import ArrowRight from '#lib/components/icons/ArrowRight.svelte';
	import { cn } from '#lib/utils.js';

	interface DocGroup {
		/** 分组标题；同时也是展开态的 key */
		title: string;
		items?: { label: string; to: string }[];
		groups?: DocGroup[]; // 嵌套子分组（目前只有「最佳实践」一层）
	}

	const navTree: (DocGroup | { label: string; to: string })[] = [
		{ label: 'IPv6 工具箱使用文档', to: '/doc' },
		{
			title: 'IPv6 用户端',
			items: [
				{ label: '个人宽带如何开启IPv6网络访问', to: '/doc/user/enable_ipv6' },
				{ label: '命令行禁用/启用IPv6本地网络', to: '/doc/user/cmd_bash_disable_ipv6' },
				{ label: '命令行(curl)获取 IPv4 和 IPv6 地址', to: '/doc/user/cmd_getip' },
				{ label: '浏览器访问 IPv6 地址', to: '/doc/user/view_ipv6_adress_url' },
				{ label: 'Windows 10/11 设置 IPv4/IPv6 访问优先级', to: '/doc/user/ipv4_ipv6_prefix_precedence' },
				{ label: '国内 IPv6 资源导航', to: '/doc/user/ipv6_daohang' },
				{ label: '国内纯 IPv6 网站导航', to: '/doc/user/pure_ipv6_website' },
				{ label: '全国各省 DNS 服务器列表', to: '/doc/user/dns' },
				{ label: 'IPv6 DNS 地址列表', to: '/doc/user/ipv6_dns' },
				{ label: 'Python/Go 获取 IPv4 和 IPv6 地址', to: '/doc/user/code_getip' },
				{ label: 'DNS 解析流程', to: '/doc/user/dns_dig' }
			],
			groups: [
				{
					title: '最佳实践',
					items: [
						{ label: '阿里云自动化添加安全组', to: '/doc/user/AliyunAuthorizeSecurityGroup' },
						{ label: '腾讯云自动化添加安全组', to: '/doc/user/TencentCloudAddSecurityGroup' }
					]
				}
			]
		},
		{
			title: '云服务器配置 IPv6',
			items: [
				{ label: '网站开启 IPv6 的三种方式', to: '/doc/server/website_enable_ipv6' },
				{ label: '腾讯云 cvm 开启 IPv6', to: '/doc/server/tencent_cloud_cvm_ipv6' },
				{ label: 'Nginx 开启 IPv6', to: '/doc/server/nginx_ipv6' },
				{ label: '如何确认一个网站是否开启 IPv6', to: '/doc/server/ipv6webcheck' },
				{ label: '网站增加支持IPv6访问标识', to: '/doc/server/ipv6_sign' },
				{ label: '如何为域名添加 IPv6 解析记录', to: '/doc/server/ipv6_domain_record' },
				{ label: '网站如何开启 HTTP/2', to: '/doc/server/http2' },
				{ label: 'JS 检查网络是 IPv4 还是 IPv6', to: '/doc/server/check_userip_ipv4_ipv6' },
				{ label: '网站如何开启 TLS', to: '/doc/server/tls' }
			]
		},
		{
			title: 'IPv6 协议规范',
			items: [
				{ label: 'IPv6 RFC8200 解读', to: '/doc/rfc/rfc8200' },
				{ label: 'IPv6 地址标识方法', to: '/doc/rfc/ipv6_address_format' },
				{ label: 'tcpdump 分析IPv6包', to: '/doc/user/tcpdump_ipv6' },
				{ label: 'WireShark 分析IPv6包头', to: '/doc/user/wireshark_ipv6' },
				{ label: 'IPv6 Ping 原理', to: '/doc/user/ipv6_ping' }
			]
		},
		{
			title: 'API 接口',
			items: [
				{ label: '获取客户端公网 IP 及位置', to: '/doc/api/ip/locate' },
				{ label: '获取客户端公网 IP 地址', to: '/doc/api/ip/myip' },
				{ label: '查询指定 IP 地址的位置信息', to: '/doc/api/ip/query' },
				{ label: '查询域名的 WHOIS 信息', to: '/doc/api/whois/query' }
			]
		}
	];

	let { expandAll = false, class: className = '', onSelect }: {
		/** true = 所有分组初始展开（抽屉用）；false = 默认收起（页面侧栏用） */
		expandAll?: boolean;
		class?: string;
		onSelect?: () => void;
	} = $props();

	const ALL_GROUPS = ['IPv6 用户端', '最佳实践', '云服务器配置 IPv6', 'IPv6 协议规范', 'API 接口'];

	// 用 untrack 包住：展开态只在**初次挂载**时按 expandAll 定一次，之后完全由用户点击控制。
	// 直接写 `$state(expandAll ? ... : [])` 会被 Svelte 判为「只捕获了 prop 的初始值」
	// （state_referenced_locally）—— 这里正是有意为之，untrack 把这个意图写明。
	const initialOpened = untrack(() => (expandAll ? [...ALL_GROUPS] : []));
	let opened = $state<string[]>(initialOpened);

	function toggle(title: string) {
		opened = opened.includes(title) ? opened.filter((t) => t !== title) : [...opened, title];
	}

	const currentPath = $derived(page.url.pathname);

	function isActive(to: string) {
		return currentPath === to;
	}
</script>

<!--
  收起的分组**必须留在 DOM 里**：/doc/** 走 SSG，预渲染的爬虫靠 HTML 里的 <a> 才能把
  32 篇文档全部枚举出来（旧站 el-menu 是 display 级折叠，链接本来就在 HTML 中）。
  所以这里不用 {#if}，而是切 .ak-collapse 的 grid-template-rows（0fr → 1fr）做高度过渡，
  效果等价于 EP 的折叠动画；收起时额外把它置为 visibility:hidden，
  免得高度为 0 的链接还能被 Tab 焦点进去。
-->
<nav aria-label="文档导航" class={cn('ak-doc-menu', className)}>
	<ul class="ak-doc-list">
		{#each navTree as node (('to' in node ? node.to : node.title))}
			{#if 'to' in node}
				<li>
					<a
						href={node.to}
						class="ak-doc-item"
						class:is-active={isActive(node.to)}
						onclick={onSelect}>{node.label}</a
					>
				</li>
			{:else}
				{@const group = node as DocGroup}
				{@const isOpen = opened.includes(group.title)}
				<li>
					<button
						type="button"
						class="ak-doc-item"
						aria-expanded={isOpen}
						onclick={() => toggle(group.title)}
					>
						<span>{group.title}</span>
						<ArrowRight class={cn('ak-doc-arrow', isOpen && 'is-open')} />
					</button>
					<div class="ak-collapse" class:is-open={isOpen}>
						<ul class="ak-doc-list">
							{#each group.items ?? [] as item (item.to)}
								<li>
									<a
										href={item.to}
										class="ak-doc-item"
										class:is-active={isActive(item.to)}
										onclick={onSelect}>{item.label}</a
									>
								</li>
							{/each}
							{#each group.groups ?? [] as sub (sub.title)}
								{@const subOpen = opened.includes(sub.title)}
								<li>
									<button
										type="button"
										class="ak-doc-item"
										aria-expanded={subOpen}
										onclick={() => toggle(sub.title)}
									>
										<span>{sub.title}</span>
										<ArrowRight class={cn('ak-doc-arrow', subOpen && 'is-open')} />
									</button>
									<div class="ak-collapse" class:is-open={subOpen}>
										<ul class="ak-doc-list">
											{#each sub.items ?? [] as item (item.to)}
												<li>
													<a
														href={item.to}
														class="ak-doc-item"
														class:is-active={isActive(item.to)}
														onclick={onSelect}>{item.label}</a
													>
												</li>
											{/each}
										</ul>
									</div>
								</li>
							{/each}
						</ul>
					</div>
				</li>
			{/if}
		{/each}
	</ul>
</nav>
