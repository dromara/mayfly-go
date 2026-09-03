/**
 * 中断模块统一导出
 * 对齐 tokhub interrupts/index.ts：单一注册表，按中断类型查找处理器
 * （业务逻辑 + UI 操作组件同源，经 handler.component 渲染）。
 *
 * 目录约定：
 * - handlers/：各中断类型处理器（xxxHandler，工厂/公共逻辑见 helpers.ts）
 * - 兜底组件 GenericInterrupt：未注册类型的降级渲染
 *
 * 新增内置中断类型时只需：
 *   1. 创建 XxxInterrupt.vue 实现 InterruptComponentProps
 *   2. 在 handlers/ 下经 createInterruptHandler 创建处理器（必要时 overrides 覆盖差异点）
 *   3. 在下方注册处理器
 *
 * 宿主 / 插件无需修改框架源码：调用 registerInterruptHandler 即可挂载自定义类型。
 *
 * 公共代码（eventWriter / ToolCallPart / chatStore）只通过注册表访问类型特定逻辑，
 * 不引用具体类型名称。
 */

import type { Component } from 'vue';
import { markRaw } from 'vue';
import GenericInterrupt from './GenericInterrupt.vue';
import { registerInterruptHandler, getInterruptHandler } from '../registries/interruptRegistry';
import { approvalHandler } from './handlers/approvalHandler';
import { confirmationHandler } from './handlers/confirmationHandler';
import { paramCompletionHandler } from './handlers/paramCompletionHandler';

// ==================== 内置中断处理器注册 ====================

registerInterruptHandler(approvalHandler);

registerInterruptHandler(confirmationHandler);

registerInterruptHandler(paramCompletionHandler);

// ==================== 组件查找（降级方案） ====================

const DEFAULT_COMPONENT: Component = markRaw(GenericInterrupt);

/**
 * 根据中断类型获取对应的组件
 * 优先从注册表查找（handler.component），未注册则返回默认组件
 */
export function getInterruptComponent(type?: string): Component {
    if (!type) return DEFAULT_COMPONENT;
    const handler = getInterruptHandler(type);
    return handler?.component || DEFAULT_COMPONENT;
}

// ==================== 导出 ====================

export { default as ApprovalInterrupt } from './ApprovalInterrupt.vue';
export { default as ConfirmationInterrupt } from './ConfirmationInterrupt.vue';
export { default as GenericInterrupt } from './GenericInterrupt.vue';
export { default as ParamCompletionInterrupt } from './param-completion/index.vue';
export {
    stateFromInterruptEvent,
    createInterruptHandler,
    interpretDecisionWithFallback,
} from './helpers';
export type { InterruptActionEvent, InterruptActionHandler, InterruptComponentProps } from './types';
