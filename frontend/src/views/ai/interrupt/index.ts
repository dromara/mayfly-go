import type { Component } from 'vue';
import ApprovalInterrupt from './ApprovalInterrupt.vue';
import ConfirmationInterrupt from './ConfirmationInterrupt.vue';
import GenericInterrupt from './GenericInterrupt.vue';
import ParamCompletionInterrupt from './param-completion/index.vue';

/**
 * 中断组件类型映射表
 * key: interrupt type (如 'interrupt_approval', 'interrupt_confirmation' 等)
 * value: 对应的 Vue 组件
 */
const interruptComponentMap = new Map<string, Component>([
    ['interrupt_approval', ApprovalInterrupt],
    ['interrupt_confirmation', ConfirmationInterrupt],
    ['interrupt_param_completion', ParamCompletionInterrupt],
]);

/**
 * 默认中断组件（当 type 未注册时使用）
 */
const DEFAULT_INTERRUPT_COMPONENT = GenericInterrupt;

/**
 * 注册新的中断组件类型
 * @param type 中断类型标识（对应 internal.extra.type）
 * @param component Vue 组件
 *
 * @example
 * registerInterruptComponent('CUSTOM_TYPE', CustomInterruptComponent)
 */
export function registerInterruptComponent(type: string, component: Component) {
    interruptComponentMap.set(type.toLowerCase(), component);
}

/**
 * 根据中断类型获取对应的组件
 * @param type 中断类型标识
 * @returns 对应的 Vue 组件，如果未找到则返回默认组件
 *
 * @example
 * const Component = getInterruptComponent('APPROVAL')
 */
export function getInterruptComponent(type?: string): Component {
    if (!type) {
        return DEFAULT_INTERRUPT_COMPONENT;
    }

    const component = interruptComponentMap.get(type.toLowerCase());
    return component || DEFAULT_INTERRUPT_COMPONENT;
}

/**
 * 获取所有已注册的中断类型
 * @returns 类型数组
 */
export function getRegisteredInterruptTypes(): string[] {
    return Array.from(interruptComponentMap.keys());
}

/**
 * 检查某个类型是否已注册
 * @param type 中断类型
 * @returns 是否已注册
 */
export function isInterruptTypeRegistered(type: string): boolean {
    return interruptComponentMap.has(type.toLowerCase());
}

// 导出组件以便直接使用
export { ApprovalInterrupt, ParamCompletionInterrupt, ConfirmationInterrupt, GenericInterrupt };

// 导出类型定义
export type { InterruptActionEvent } from './types';
