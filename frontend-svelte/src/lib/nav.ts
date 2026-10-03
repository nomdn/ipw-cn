/**
 * 顶部导航菜单的映射表（原 frontend-ssr/app/app.vue 里的 `menu` 数组，原样搬过来）。
 *
 * 规则：菜单按显示顺序排在数组里，每项自带 minWidth（px）折叠阈值；
 * 视口宽度 >= minWidth 显示，低于则折叠进「更多」子菜单。
 * 断点从右往左递减（最右侧 index 9 先折叠），数值可按需调整。
 * divider 也带 minWidth（取其后第一项的值）：所在段全部折叠时，分隔线一并隐藏。
 *
 * 断点设计基准：全显示约 1480px，1366 屏可用约 1350px，
 * 故 1366 屏折叠 index 7/8/9（显示 1-6 +「更多」，约 1220px 宽裕）；1440 折叠 8/9；1500+ 全显示。
 */
export interface NavItem {
	kind: 'item';
	index: string;
	label: string;
	to: string;
	minWidth: number;
}
export interface NavGroup {
	kind: 'group';
	index: string;
	label: string;
	minWidth: number;
	children: { index: string; label: string; to: string }[];
}
export interface NavDivider {
	kind: 'divider';
	minWidth: number;
}
export type MenuEntry = NavItem | NavGroup | NavDivider;

export const menu: MenuEntry[] = [
	{ kind: 'item', index: '1', label: 'IPv6 网站检测', to: '/ipv6webcheck', minWidth: 980 },
	{ kind: 'item', index: '2', label: 'IPv6/IPv4 地址查询', to: '/location', minWidth: 1040 },
	{ kind: 'item', index: '3', label: 'IPv6 TCPing测试', to: '/ipv6tcping', minWidth: 1100 },
	{ kind: 'divider', minWidth: 1160 },
	{ kind: 'item', index: '4', label: 'IPv6 DNS解析', to: '/dns', minWidth: 1160 },
	{ kind: 'item', index: '5', label: 'IPv6 SSL检查', to: '/ssl', minWidth: 1220 },
	{ kind: 'item', index: '6', label: 'IPv6 网站测速', to: '/ipv6speedtest', minWidth: 1280 },
	{ kind: 'divider', minWidth: 1380 },
	{
		kind: 'group',
		index: '7',
		label: 'IPv4工具箱',
		minWidth: 1380,
		children: [
			{ index: '7-0', label: 'IPv4 网站测速', to: '/speedtest' },
			{ index: '7-1', label: 'IPv4 TCPing测试', to: '/tcping' }
		]
	},
	{
		kind: 'group',
		index: '8',
		label: '其他工具',
		minWidth: 1420,
		children: [
			{ index: '8-0', label: '网站截图', to: '/screenshot' },
			{ index: '8-1', label: 'Whois查询', to: '/whois' },
			{ index: '8-3', label: 'ASN查询', to: '/asn' },
			{ index: '8-4', label: 'DNSSEC验证', to: '/dnssec' }
		]
	},
	{ kind: 'item', index: '9', label: '文档', to: '/doc', minWidth: 1500 }
];

/** 窄屏抽屉里的工具链接（原 app.vue 的 el-drawer 内容，顺序与文案保持一致） */
export const drawerLinks: { label: string; to: string }[] = [
	{ label: 'IPv6 网站检测', to: '/ipv6webcheck' },
	{ label: 'IPv6/IPv4 地址查询', to: '/location' },
	{ label: 'IPv6 TCPing', to: '/ipv6tcping' },
	{ label: 'IPv6 DNS解析', to: '/dns' },
	{ label: 'IPv6 SSL检查', to: '/ssl' },
	{ label: 'IPv6 网站测速', to: '/ipv6speedtest' },
	{ label: 'IPv4 网站测速', to: '/speedtest' },
	{ label: 'IPv4 TCPing', to: '/tcping' },
	{ label: '网站截图', to: '/screenshot' },
	{ label: 'Whois查询', to: '/whois' },
	{ label: 'ASN查询', to: '/asn' },
	{ label: 'DNSSEC验证', to: '/dnssec' }
];

/** 是否文档路由（文档页不渲染页脚、抽屉里改挂文档导航） */
export function isDocPath(pathname: string): boolean {
	return pathname === '/doc' || pathname.startsWith('/doc/');
}
