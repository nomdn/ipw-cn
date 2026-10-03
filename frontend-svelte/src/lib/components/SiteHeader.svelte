<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Expand from '#lib/components/icons/Expand.svelte';
	import Moon from '#lib/components/icons/Moon.svelte';
	import Sunny from '#lib/components/icons/Sunny.svelte';
	import ArrowDown from '#lib/components/icons/ArrowDown.svelte';
	import { config } from '#lib/config/index.ts';
	import { menu, drawerLinks, isDocPath, type MenuEntry, type NavGroup, type NavItem } from '#lib/nav.ts';
	import { resolveTheme, toggleTheme } from '#lib/theme.ts';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import DocMenu from '#lib/components/DocMenu.svelte';

	// SSR 阶段没有 window，默认按宽屏输出全部菜单项（与旧站一致）；onMount 后换成真实视口宽度。
	let viewportWidth = $state(1920);
	let isNarrow = $state(false);
	let drawerOpen = $state(false);
	let theme = $state<'light' | 'dark'>('light');

	const isDocRoute = $derived(isDocPath(page.url.pathname));

	// 宽屏可见项（含分隔线）：视口宽度达标即显示
	const visible = $derived(menu.filter((entry) => viewportWidth >= entry.minWidth));
	// 折叠项（不含分隔线）：视口宽度不足即收进「更多」
	const hidden = $derived(
		menu.filter((e): e is NavItem | NavGroup => e.kind !== 'divider' && viewportWidth < e.minWidth)
	);

	// 断点必须与旧站完全一致：旧站用的是 matchMedia('(max-width: 768px)')。
	// 这里刻意**不用 Tailwind 的 md: 变体** —— md 是 min-width:768px，在正好 768px 时
	// 会和 max-width:768px 同时成立，窄屏布局和宽屏布局会一起冒出来（就是那个
	// 「窄屏下出现两个汉堡按钮」的现象之一）。
	function onResize() {
		viewportWidth = window.innerWidth;
		isNarrow = window.innerWidth <= 768;
	}

	onMount(() => {
		theme = resolveTheme();
		onResize();
		window.addEventListener('resize', onResize);
		return () => window.removeEventListener('resize', onResize);
	});
</script>

<!--
  顶栏。结构对齐旧站的 el-menu（mode="horizontal"）：

    窄屏：[汉堡] [logo]  …  [明暗切换]
    宽屏：[logo 站名] [菜单项…] [分隔线] [分组 ▾] … [更多 ▾] [明暗切换]

  · logo 那一项带 margin-right:auto —— 宽屏余量顶在左侧、菜单整体靠右（旧站同款）；
  · 全部分隔线、尺寸、hover 规则都在 site.css 的 .ak-menu-* 里，
    数值取自 element-plus 源码（见那一节的注释），别在这里用 Tailwind 类拼尺寸，
    否则折叠断点会整体失准；
  · 汉堡图标只有**一个** —— 就在 logo 左侧，就是旧站 index 0 那一项里的 Expand 图标。
    （之前 logo 链接里还塞了一个纯装饰的汉堡，加上真正的触发按钮，窄屏看起来是两个。）
-->
<header class="ak-header">
	<ul class="ak-menu">
		<li class="ak-menu-logo">
			{#if isNarrow}
				<Sheet.Root bind:open={drawerOpen}>
					<Sheet.Trigger>
						{#snippet child({ props })}
							<button {...props} class="ak-menu-icon ak-menu-burger" aria-label="打开导航菜单">
								<Expand />
							</button>
						{/snippet}
					</Sheet.Trigger>
					<!-- 抽屉宽度按旧站：文档路由 80%（要放整份文档目录），其它 60%。
					     内容样式见 site.css 的 .ak-drawer*（照搬旧站对 el-drawer 的非 scoped 覆盖）。 -->
					<Sheet.Content
						side="left"
						class={isDocRoute ? 'ak-drawer-sheet ak-drawer-sheet-doc' : 'ak-drawer-sheet'}
					>
						<nav class="ak-drawer" aria-label="移动端导航">
							{#each drawerLinks as link (link.to)}
								<a href={link.to} onclick={() => (drawerOpen = false)}>{link.label}</a>
							{/each}
							{#if !isDocRoute}
								<!-- 非文档页抽屉里只留一个文档入口（旧站同款）：整份目录太长 -->
								<a href="/doc" onclick={() => (drawerOpen = false)}>文档</a>
							{:else}
								<div class="ak-drawer-doc">
									<DocMenu expandAll onSelect={() => (drawerOpen = false)} />
								</div>
							{/if}
						</nav>
					</Sheet.Content>
				</Sheet.Root>
			{/if}

			<a href="/" aria-label={isNarrow ? '返回首页' : undefined}>
				<img src="/favicon.svg" alt="" width="50" height="50" />
				{#if !isNarrow}
					<h2>{config.siteName}</h2>
				{/if}
			</a>
		</li>

		{#if !isNarrow}
			{#each visible as entry (entry.kind === 'divider' ? `div-${entry.minWidth}` : entry.index)}
				{#if entry.kind === 'item'}
					<li><a href={entry.to} class="ak-menu-item">{entry.label}</a></li>
				{:else if entry.kind === 'group'}
					<li>
						<DropdownMenu.Root>
							<DropdownMenu.Trigger>
								{#snippet child({ props })}
									<button {...props} class="ak-menu-group">
										{entry.label}
										<!-- 箭头与「展开时旋转 180°」都用 CSS 管：
										     bits-ui 会把 data-state 写在触发按钮上（见 site.css）。 -->
										<ArrowDown class="ak-menu-arrow" />
									</button>
								{/snippet}
							</DropdownMenu.Trigger>
							<DropdownMenu.Content align="start" class="ak-popover">
								{#each entry.children as sub (sub.index)}
									<DropdownMenu.Item>
										{#snippet child({ props })}
											<a {...props} href={sub.to}>{sub.label}</a>
										{/snippet}
									</DropdownMenu.Item>
								{/each}
							</DropdownMenu.Content>
						</DropdownMenu.Root>
					</li>
				{:else}
					<!-- 分组之间的竖分隔条：纯装饰，aria-hidden -->
					<li aria-hidden="true" class="ak-menu-divider"></li>
				{/if}
			{/each}

			<!-- 折叠项统一收进「更多」 -->
			{#if hidden.length}
				<li>
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<button {...props} class="ak-menu-group">
									更多
									<ArrowDown class="ak-menu-arrow" />
								</button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="start" class="ak-popover">
							{#each hidden as entry (entry.index)}
								{#if entry.kind === 'item'}
									<DropdownMenu.Item>
										{#snippet child({ props })}
											<a {...props} href={entry.to}>{entry.label}</a>
										{/snippet}
									</DropdownMenu.Item>
								{:else}
									<DropdownMenu.Group>
										<DropdownMenu.GroupHeading>{entry.label}</DropdownMenu.GroupHeading>
										{#each entry.children as sub (sub.index)}
											<DropdownMenu.Item>
												{#snippet child({ props })}
													<a {...props} href={sub.to}>{sub.label}</a>
												{/snippet}
											</DropdownMenu.Item>
										{/each}
									</DropdownMenu.Group>
								{/if}
							{/each}
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				</li>
			{/if}
		{/if}

		<!-- 明暗切换：窄屏也保留（旧站 index 10 那一项没被窄屏规则隐藏，靠 logo 的
		     margin-right:auto 顶到最右侧） -->
		<li>
			<button
				class="ak-menu-icon"
				aria-label="切换深色/浅色模式"
				onclick={() => {
					toggleTheme();
					theme = resolveTheme();
				}}
			>
				{#if theme === 'dark'}
					<Moon />
				{:else}
					<Sunny />
				{/if}
			</button>
		</li>
	</ul>
</header>
