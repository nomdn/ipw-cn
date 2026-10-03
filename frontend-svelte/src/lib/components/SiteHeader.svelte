<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	// 图标走 `@lucide/svelte/icons/<name>` 子路径导入，而不是从包根 import：
	// 包根的 dist/lucide-svelte.js 里有一句 `export * from './icons/index.js'`，
	// 但发布出来的 tarball 里并没有 dist/icons/index.js（barrel 文件缺失），
	// 于是从根导入会直接 "Module not found"。子路径导入同时也让 tree-shaking 更干净
	// —— 这也是 shadcn-svelte 自己生成的组件采用的方式。
	import Moon from '@lucide/svelte/icons/moon';
	import Sun from '@lucide/svelte/icons/sun';
	import MenuIcon from '@lucide/svelte/icons/menu';
	import { config } from '#lib/config/index.ts';
	import { menu, drawerLinks, isDocPath, type MenuEntry, type NavGroup, type NavItem } from '#lib/nav.ts';
	import { resolveTheme, toggleTheme } from '#lib/theme.ts';
	import { Button } from '#lib/components/ui/button/index.js';
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
  顶栏。与旧站（element-plus 水平菜单）相比，这里换成语义化的 <header>/<nav>/<ul>/<li>/<a>：
  旧站之所以要写一大堆 CSS 去压制 EP 的内部结构（把内层 a 撑满 li 以满足 target-size、
  给装饰分隔线改回 tabindex、给 logo 链接补条件 aria-label），根因就是 EP 的 el-menu
  会往 DOM 上写 role="menuitem" 与 tabindex。原生结构没有这些问题，代码量少一大截。
-->
<header class="sticky top-0 z-40 w-full border-b border-border bg-background/95 backdrop-blur">
	<div class="mx-auto flex h-16 max-w-[1800px] items-center gap-2 px-4">
		<!-- logo：宽屏显示站名，窄屏只留图标 + 汉堡按钮 -->
		<a
			href="/"
			class="flex shrink-0 items-center gap-2 text-foreground no-underline"
			aria-label={isNarrow ? '返回首页' : undefined}
		>
			{#if isNarrow}
				<MenuIcon class="size-6 md:hidden" aria-hidden="true" />
			{/if}
			<img
				src="/favicon.svg"
				alt=""
				width="50"
				height="50"
				class="size-[38px] shrink-0"
			/>
			<h2 class="hidden text-lg font-semibold whitespace-nowrap md:inline-block">
				{config.siteName}
			</h2>
		</a>

		<!-- 宽屏导航：SSR 先按宽屏渲染，水合后按真实视口折叠 -->
		{#if !isNarrow}
			<nav aria-label="主导航" class="hidden min-w-0 flex-1 md:block">
				<ul class="flex items-center gap-1 pl-2">
					{#each visible as entry (entry.kind === 'divider' ? `div-${entry.minWidth}` : entry.index)}
						{#if entry.kind === 'item'}
							<li>
								<a
									href={entry.to}
									class="inline-flex h-9 items-center rounded-md px-3 text-sm whitespace-nowrap text-foreground no-underline hover:bg-muted"
								>
									{entry.label}
								</a>
							</li>
						{:else if entry.kind === 'group'}
							<li>
								<DropdownMenu.Root>
									<DropdownMenu.Trigger>
										{#snippet child({ props })}
											<button
												{...props}
												class="inline-flex h-9 items-center rounded-md px-3 text-sm whitespace-nowrap hover:bg-muted"
											>
												{entry.label}
											</button>
										{/snippet}
									</DropdownMenu.Trigger>
									<DropdownMenu.Content align="start">
										{#each entry.children as sub (sub.index)}
											<DropdownMenu.Item>
												{#snippet child({ props })}
													<a {...props} href={sub.to} class="no-underline">{sub.label}</a>
												{/snippet}
											</DropdownMenu.Item>
										{/each}
									</DropdownMenu.Content>
								</DropdownMenu.Root>
							</li>
						{:else}
							<!-- 分组之间的竖分隔条：纯装饰，aria-hidden（旧站为了躲开 axe 的
							     aria-hidden-focus / aria-required-attr 折腾过一轮，原生 span 没这问题）。 -->
							<li aria-hidden="true" class="mx-1 h-5 w-px shrink-0 self-center bg-border"></li>
						{/if}
					{/each}

					<!-- 折叠项统一收进「更多」 -->
					{#if hidden.length}
						<li>
							<DropdownMenu.Root>
								<DropdownMenu.Trigger>
									{#snippet child({ props })}
										<button
											{...props}
											class="inline-flex h-9 items-center rounded-md px-3 text-sm whitespace-nowrap hover:bg-muted"
										>
											更多
										</button>
									{/snippet}
								</DropdownMenu.Trigger>
								<DropdownMenu.Content align="start">
									{#each hidden as entry (entry.index)}
										{#if entry.kind === 'item'}
											<DropdownMenu.Item>
												{#snippet child({ props })}
													<a {...props} href={entry.to} class="no-underline">{entry.label}</a>
												{/snippet}
											</DropdownMenu.Item>
										{:else}
											<DropdownMenu.Group>
												<DropdownMenu.GroupHeading>{entry.label}</DropdownMenu.GroupHeading>
												{#each entry.children as sub (sub.index)}
													<DropdownMenu.Item>
														{#snippet child({ props })}
															<a {...props} href={sub.to} class="no-underline">{sub.label}</a>
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
				</ul>
			</nav>
		{:else}
			<!-- 窄屏：汉堡按钮打开左侧抽屉（内挂工具链接；文档路由下追加文档导航） -->
			<Sheet.Root bind:open={drawerOpen}>
				<Sheet.Trigger>
					{#snippet child({ props })}
						<Button {...props} variant="ghost" size="icon" aria-label="打开导航菜单">
							<MenuIcon class="size-5" aria-hidden="true" />
						</Button>
					{/snippet}
				</Sheet.Trigger>
				<Sheet.Content side="left" class="w-[70%] overflow-y-auto">
					<Sheet.Header>
						<Sheet.Title>{config.siteName}</Sheet.Title>
					</Sheet.Header>
					<nav aria-label="移动端导航" class="flex flex-col px-2 pb-6">
						{#each drawerLinks as link (link.to)}
							<a
								href={link.to}
								class="rounded-md px-3 py-3 text-[15px] text-foreground no-underline hover:bg-muted"
								onclick={() => (drawerOpen = false)}
							>
								{link.label}
							</a>
						{/each}
						{#if !isDocRoute}
							<a
								href="/doc"
								class="rounded-md px-3 py-3 text-[15px] text-foreground no-underline hover:bg-muted"
								onclick={() => (drawerOpen = false)}
							>
								文档
							</a>
						{:else}
							<div class="mt-2 border-t border-border pt-2">
								<DocMenu expandAll onSelect={() => (drawerOpen = false)} />
							</div>
						{/if}
					</nav>
				</Sheet.Content>
			</Sheet.Root>
		{/if}

		<div class="ml-auto flex shrink-0 items-center">
			<Button
				variant="ghost"
				size="icon"
				aria-label="切换深色/浅色模式"
				onclick={() => {
					toggleTheme();
					theme = resolveTheme();
				}}
			>
				{#if theme === 'dark'}
					<Moon class="size-5" aria-hidden="true" />
				{:else}
					<Sun class="size-5" aria-hidden="true" />
				{/if}
			</Button>
		</div>
	</div>
</header>
