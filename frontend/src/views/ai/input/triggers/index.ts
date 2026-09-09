/**
 * 触发器子系统统一出口
 *
 * 新增触发类型步骤：
 * 1. 新建 <name>Trigger.ts，实现 TriggerDef 并 registerTrigger 注册
 *    （列表模式给 buildItems/onSelect；面板模式给 panelComponent）；
 * 2. 在 registry.ts 顶部追加副作用导入；
 * 3. 如为面板模式，在 ChatInput 模板补对应浮层组件分支。
 * 编辑器检测/键盘/定位/芯片插入均由注册表驱动，无需其它改动。
 */

// ── 类型导出 ───────────────────────────────────────────────────
export type {
    TriggerDef,
    TriggerContext,
    TriggerMenuItem,
    TriggerSelection,
    TriggerPanelSelectPayload,
} from './types';

// ── 注册表导出 ─────────────────────────────────────────────────
export { registerTrigger, getTriggerDefs, findTriggerDef, getTriggerChars, buildTriggerTokenRegex } from './registry';

// ── Hooks 导出 ─────────────────────────────────────────────────
export { useTriggerState, type TriggerState, type UseTriggerStateOptions } from './useTriggerState';
export { useMenuPosition } from './useMenuPosition';

// ── 触发器实现导出 ─────────────────────────────────────────────
export { buildSkillItems, selectSkillItem } from './skillTrigger';
export { selectResourceFromPanel } from './resourceTrigger';

// ── 浮层组件导出（面板 payload 类型未在此 re-export：*.vue shim 只认 default 导出，
//    消费方需要时从 './ResourceTreePanel.vue' 直接 import，与历史用法一致）
export { default as TriggerMenu } from './TriggerMenu.vue';
export { default as ResourceTreePanel } from './ResourceTreePanel.vue';
