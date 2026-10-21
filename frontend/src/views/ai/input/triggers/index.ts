/**
 * 触发器子系统统一出口
 *
 * 新增触发类型步骤（注册即用，零模板/零编辑器改动）：
 * 1. 新建 <name>Trigger.ts，实现 TriggerDef 并 registerTrigger 注册：
 *    列表模式给 buildItems/onSelect；面板模式给 panelComponent/onPanelSelect；
 * 2. 在 registry.ts 底部追加注册调用（不能在被注册文件顶层自注册，会与 registry 形环）。
 *
 * 检测/键盘导航/浮层定位/面板渲染/芯片插入全部由注册表驱动：TriggerOverlays 动态渲染
 * activeDef.panelComponent，选中转换取 activeDef.onPanelSelect，新增类型不需要改任何模板。
 */

// ── 类型导出 ───────────────────────────────────────────────────
export type { TriggerDef, TriggerContext, TriggerMenuItem, TriggerSelection, TriggerPanelSelectPayload } from './types';

// ── 注册表导出 ─────────────────────────────────────────────────
export { registerTrigger, getTriggerDefs, findTriggerDef, getTriggerChars, buildTriggerTokenRegex } from './registry';

// ── Hooks 导出 ─────────────────────────────────────────────────
export { useTriggerState, type TriggerState, type UseTriggerStateOptions } from './useTriggerState';
export { useMenuPosition } from './useMenuPosition';

// ── 触发器实现导出 ─────────────────────────────────────────────
// 触发器实现不从桶文件对外导出：注册由 registry 底部完成，测试直取模块（对外导出会诱导消费方绕过注册表）

// ── 浮层组件导出（面板 payload 类型未在此 re-export：*.vue shim 只认 default 导出，
//    消费方需要时从 './ResourceTreePanel.vue' 直接 import，与历史用法一致）
export { default as TriggerMenu } from './TriggerMenu.vue';
export { default as ResourceTreePanel } from './ResourceTreePanel.vue';
