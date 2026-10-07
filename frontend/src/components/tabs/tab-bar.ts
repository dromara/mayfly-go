import ElTabsBar from './ElTabsBar.vue';
import UnifiedTabBar from './UnifiedTabBar.vue';

/**
 * 标签条实现的单一切换点（Switch Point）。
 *
 * - 'vscode'  : 自研 VS Code 编辑器标签风格（默认）
 * - 'element' : 回退到 Element Plus 原生 el-tabs
 *
 * 高层 Tabs 与需要自管内容区的页面都只认这里导出的 TabBar，不直接 import 具体 .vue；
 * 因此「换回 Element Plus」或「接入第三种实现」只需改这里（或新增一个满足契约的组件再注册），
 * 所有使用方零改动 —— 对扩展开放、对修改关闭。
 */
export type TabBarVariant = 'vscode' | 'element';

/** 切换实现只改这里的返回值；返回类型标注为联合，保证下方分支比较合法（const 字面量会被 TS 收窄导致比较报错） */
function currentVariant(): TabBarVariant {
    return 'vscode';
}

export const TabBar = currentVariant() === 'element' ? ElTabsBar : UnifiedTabBar;
