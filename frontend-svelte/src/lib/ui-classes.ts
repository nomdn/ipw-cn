/**
 * 工具页里输入框 / 按钮 / 下拉框的统一尺寸。
 *
 * 旧站是靠 `assets/css/tool-common.css` 里的 `.el-input / .el-button / .el-select`
 * 硬覆盖 element-plus 的组件尺寸（50px 高、420px 宽、1.3em 字），窄屏再整体改成
 * 100% 宽 / 40px 高 / 0.9em 字。EP 换成 shadcn-svelte 之后，同样的效果落在
 * Tailwind 类上 —— 但它散在 13 个页面里各写一遍，漏一个就会和别的页面对不齐，
 * 所以集中在这里，页面只 import 常量。
 *
 * 为什么不用 `@apply` 写进 site.css：`max-md:` 这类带断点的变体只能用 Tailwind 语法，
 * 而 shadcn 组件的 `class` 是通过 `cn()`（= clsx + tailwind-merge）合并的，
 * 只有以字符串形式传进去的类才会参与冲突消解（后写的覆盖组件自带的 h-9 / text-sm）。
 */
const FIELD_BASE = 'h-[50px] text-[1.3em] max-md:h-10 max-md:text-[0.9em]';

/** 主输入框：420px 宽，与旧站 .el-input 一致 */
export const INPUT_CLASS = `${FIELD_BASE} w-[420px] max-w-full mr-2.5 max-md:mr-0 max-md:mb-2.5 max-md:w-full`;

/** 次要输入框（如端口号）：200px 宽 */
export const INPUT_NARROW_CLASS = `${FIELD_BASE} w-[200px] max-w-full mr-2.5 max-md:mr-0 max-md:mb-2.5 max-md:w-full`;

/** 主按钮：165px 宽 */
export const BUTTON_CLASS = `${FIELD_BASE} w-[165px] shrink-0 max-md:w-full`;

/** 下拉框（DNS 记录类型）：150px 宽 */
export const SELECT_CLASS = `h-[50px] w-[150px] max-md:h-10 max-md:w-full max-md:mb-2.5 mr-2.5 max-md:mr-0`;
