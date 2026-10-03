/**
 * 深浅色主题（替代旧站的 @vueuse/core useDark）。
 *
 * 与旧站保持一致的三点：
 *   1. 存储键沿用 `vueuse-color-scheme`（值 `light` / `dark`；不存在 = 跟随系统），
 *      老用户的偏好在迁移后仍然生效；
 *   2. 切换方式是在 <html> 上加/去 `dark` 类 —— shadcn 的 `.dark` 令牌与
 *      site.css 的 `html.dark` 规则都认这个类；
 *   3. 首屏不闪：`src/app.html` 里有一段同步内联脚本，在 CSS 之前就把类加上，
 *      这里只负责水合之后的读写。
 */

export type ThemeChoice = 'light' | 'dark';

const STORAGE_KEY = 'vueuse-color-scheme';

function mediaQuery(): MediaQueryList | null {
	return typeof window === 'undefined' ? null : window.matchMedia('(prefers-color-scheme: dark)');
}

/** 当前生效的主题（已解析过"跟随系统"） */
export function resolveTheme(): ThemeChoice {
	if (typeof document === 'undefined') return 'light';
	return document.documentElement.classList.contains('dark') ? 'dark' : 'light';
}

function apply(theme: ThemeChoice) {
	document.documentElement.classList.toggle('dark', theme === 'dark');
}

export function initTheme() {
	const mq = mediaQuery();
	if (!mq) return;

	// 系统主题变化时，仅在"用户没手动选过"的情况下跟随
	const onChange = (e: MediaQueryListEvent) => {
		try {
			if (localStorage.getItem(STORAGE_KEY)) return;
		} catch {
			/* 隐私模式下 localStorage 可能抛错，按"没选过"处理 */
		}
		apply(e.matches ? 'dark' : 'light');
	};
	mq.addEventListener('change', onChange);
	return () => mq.removeEventListener('change', onChange);
}

export function toggleTheme() {
	const next: ThemeChoice = resolveTheme() === 'dark' ? 'light' : 'dark';
	apply(next);
	try {
		localStorage.setItem(STORAGE_KEY, next);
	} catch {
		/* 存不了就算了，本次会话仍然生效 */
	}
}
