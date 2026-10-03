/**
 * 工具页里输入框 / 按钮 / 下拉框的统一类名。
 *
 * 旧站靠 `assets/css/tool-common.css` 直接覆盖 element-plus 的 `.el-input /
 * .el-button / .el-select`（50px 高、420px 宽、1.3em 字，窄屏整体 100% 宽 / 40px 高）。
 * EP 换成 shadcn-svelte 之后，尺寸与配色落在 `src/lib/styles/site.css` 的
 * `.ak-input / .ak-button / .ak-select` 上（那里是**无层**样式，优先级高于组件库
 * 自带的 h-8 / rounded-lg / text-sm），这里只做常量导出，避免在 13 个页面里各写一遍。
 *
 * 为什么不把尺寸写成 Tailwind 类：
 *   ① 无层样式优先级高于 @layer utilities，写了 `h-[50px]` 也会被 `.ak-input` 盖掉；
 *   ② 窄屏那套「100% 宽 / 40px 高」写进 CSS 的 @media 更省事，也不用担心
 *      shadcn 组件的 class 被 `cn()`（tailwind-merge）消解掉。
 */
export const INPUT_CLASS = 'ak-input';
export const INPUT_NARROW_CLASS = 'ak-input ak-input--narrow';
export const BUTTON_CLASS = 'ak-button';
export const SELECT_CLASS = 'ak-select';
