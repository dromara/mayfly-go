/**
 * 右键菜单关闭动画时长，需与 ui/context-menu/ContextMenuContent.vue 的 duration-100 保持一致。
 *
 * 独立成 leaf 模块（不 import 任何东西）的原因：该时长同时被派发方（ContextmenuItemNode）
 * 与回归测试引用，若放在 index.ts 会形成「index.ts → index.vue → ContextmenuItemNode → index.ts」
 * 的模块循环，放在 <script setup> 内则无法导出，测试只能写死魔法数字而与之漂移。
 */
export const MENU_CLOSE_ANIMATION_MS = 100;
