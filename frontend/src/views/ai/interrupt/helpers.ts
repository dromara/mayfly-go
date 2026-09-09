/**
 * 中断恢复公共辅助函数
 *
 * 提供与中断类型无关的通用操作：
 * - stateFromInterruptEvent：从 tool_call item extra.interrupt 构造历史恢复状态
 *   （buildBaseInterruptInfo 公共构造器）
 * - createInterruptHandler：标准 handler 工厂（approval 家族工厂
 *   createApprovalFamilyHandler：buildFromHistoryItem / isPending / interpretDecision
 *   默认实现由工厂统一提供，类型差异点经 overrides 注入）
 * - interpretDecisionWithFallback：决策解释委托类型 handler，无 handler 时兜底
 */

import type { Component } from 'vue';
import { markRaw } from 'vue';
import type { DecisionInterpretation, InterruptHandler, InterruptState } from '../registries/interruptRegistry';
import type { TurnItem } from '../protocol/types';
import { interpretActionStatus, normalizeResumeStatus, InterruptResumeStatus } from '../registries/statuses';

/**
 * 从 extra.interrupt（后端持久化于 tool_call item extra 列的 InterruptInfo）
 * 构造公共历史恢复状态
 */
export function stateFromInterruptEvent(
    type: string,
    interruptInfo: Record<string, unknown>,
    item: TurnItem,
): InterruptState {
    const resume = interruptInfo.resume as Record<string, unknown> | undefined;
    return {
        kind: type,
        interruptId: String(interruptInfo.request_id || item.id),
        turnId: '',
        description: String(interruptInfo.message || ''),
        toolName: item.tool_name,
        toolCallId: item.tool_call_id,
        // 恢复决策内嵌 interrupt 对象：归一化具体决策类型，
        // 不再一律 resolved（历史与实时同值域，决议徽章可区分已批准/已完善等）
        status: resume ? normalizeResumeStatus(String(resume.type || '')) : InterruptResumeStatus.Pending,
        metadata: (interruptInfo.metadata as Record<string, unknown>) || {},
    };
}

/**
 * 创建标准中断处理器（approval 家族工厂）
 *
 * 统一提供：
 * - buildFromHistoryItem 主路径：守卫 tool_call item + extra.interrupt 匹配本类型
 * - isPending：resume 未内嵌决策即 pending
 * - interpretDecision 默认实现：委托 statuses 的通用 action → 状态映射兜底；
 *   类型特定 action 语义（如审批 reject 携带 reason）经 overrides 覆盖
 */
export function createInterruptHandler(
    type: string,
    component: Component,
    overrides?: Partial<Omit<InterruptHandler, 'type'>>,
): InterruptHandler {
    return {
        type,
        component: markRaw(component),

        buildFromHistoryItem(item: TurnItem, itemExtra?: Record<string, unknown>): InterruptState | null {
            if (item.type !== 'tool_call' || !item.tool_call_id) return null;
            // 中断信息存于 tool_call item 的 extra 列（{"interrupt": InterruptInfo}）
            const interruptInfo = itemExtra?.interrupt as Record<string, unknown> | undefined;
            // kind 为中断类型短名（approval / param_completion），handler 类型为 interrupt_{kind}
            if (!interruptInfo || `interrupt_${interruptInfo.kind}` !== type) return null;
            return stateFromInterruptEvent(type, interruptInfo, item);
        },

        isPending(state: InterruptState): boolean {
            return state.status === InterruptResumeStatus.Pending;
        },

        interpretDecision(action: string, payload?: Record<string, unknown>): DecisionInterpretation {
            return {
                status: interpretActionStatus(action),
                reason: payload?.reason as string | undefined,
                answer: payload?.answer as string | undefined,
            };
        },

        ...overrides,
    };
}

/**
 * 决策解释委托 + 兜底（interpretDecisionWithFallback）
 *
 * 优先委托类型 handler 的 interpretDecision（类型特定 action 语义），
 * 无 handler 或未返回有效状态时回退到通用 action → 状态映射
 */
export function interpretDecisionWithFallback(
    action: string,
    payload: Record<string, unknown> | undefined,
    handler?: InterruptHandler,
): DecisionInterpretation {
    const interpreted = handler?.interpretDecision(action, payload);
    if (interpreted?.status) return interpreted;
    return {
        status: interpretActionStatus(action),
        reason: payload?.reason as string | undefined,
        answer: payload?.answer as string | undefined,
    };
}
