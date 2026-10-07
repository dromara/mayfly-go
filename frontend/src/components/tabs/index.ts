import Tabs from './Tabs.vue';

/**
 * 统一标签页组件出口（通用，非 ops 专用；系统任意页面均可使用）。
 *
 * - Tabs   ：高层组件 = 标签条 + 内容面板（内置 v-show 常驻挂载保活、内置右键关闭菜单）。
 *            业务页面首选：只需给 :tabs / v-model 和一个默认作用域插槽渲染内容，
 *            无需再手写 v-for + v-show 样板。
 * - TabBar ：低层「只画标签条」的可换皮实现（切换点见 tab-bar.ts）。
 *            仅当页面需要完全自管内容区时才直接使用。
 *
 * 不对外暴露具体皮肤实现：换皮统一走 tab-bar.ts 单一切换点，
 * 避免使用方绕过契约直接 import 某个 .vue，破坏「替换实现零改动」的保证。
 */
export { Tabs };
export { TabBar } from './tab-bar';

export type { TabItem, TabBarProps, TabBarEmits, TabsEmits } from './types';
