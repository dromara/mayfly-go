/**
 * eventWriter - 流式事件单一写入口
 * 对齐 tokhub stream/eventWriter.ts
 *
 * 职责：
 * - 中断生命周期（到达 / 决策 / 回滚 / 清除）的收敛双写：
 *   pendingInterrupts（决策通道）+ message.parts（气泡片段）状态一致
 * - writeTurnNotice：用户停止/流式错误写入当前 turn 的 assistant 消息
 */

import type { InterruptEvent, NoticePart, ToolCallPart } from '../protocol/types';
import { InterruptResumeStatus, ToolCallStatus, isInterruptResolvedStatus } from '../registries/statuses';
import { getInterruptHandler } from '../registries/interruptRegistry';
import { interpretDecisionWithFallback } from '../interrupt/helpers';
import { useChatStore } from '../stores/chatStore';

/** 恢复决策（含 toolCallId，用于 parts / pendingInterrupts 精确匹配） */
export interface ResumeDecision {
    turnId: string;
    interruptId: string;
    interruptType: string;
    action: string;
    payload?: Record<string, unknown>;
    toolCallId?: string;
}

// ==================== 中断到达 ====================

/**
 * 中断到达：登记 pendingInterrupts + parts 双写 interrupted 状态
 * （对齐 tokhub eventWriter.interrupted）
 */
export function writeInterrupted(convId: number, interrupt: InterruptEvent, turnId: string) {
    const store = useChatStore();
    if (!interrupt.turnId) interrupt.turnId = turnId;
    store.addInterrupt(convId, interrupt);

    // parts：关联的 tool_call part 置 interrupted
    syncToolCallPartStatus(convId, interrupt.toolCallId, ToolCallStatus.Interrupted);
}

// ==================== 中断决策 ====================

/**
 * 中断决策：pendingInterrupts 状态翻转 + parts 双写
 * （对齐 tokhub applyInterruptDecisionToStore）
 *
 * 注意：决策后中断条目保留在 pendingInterrupts 中（status 已翻转），
 * 供恢复失败时回滚；批量恢复成功后由 clearDecidedInterrupts 统一移除。
 */
export function writeInterruptDecision(convId: number, decision: ResumeDecision) {
    const store = useChatStore();
    // 决策解释委托给类型 handler（对齐 tokhub interpretDecisionWithFallback），
    // 类型特定 action 语义由 handler.interpretDecision 解释，无 handler 时回退通用映射
    const resumeStatus = interpretDecisionWithFallback(
        decision.action,
        decision.payload,
        getInterruptHandler(decision.interruptType),
    ).status;

    // 1. pendingInterrupts：翻转决策状态
    const slice = store.getSlice(convId);
    const pending = slice?.pendingInterrupts.find((i) => i.actionId === decision.interruptId);
    if (pending) {
        pending.status = resumeStatus;
    }

    // 2. parts：载体 tool_call 双写状态流转 + 决策类型（对齐 tokhub createApplyDecisionToEvent）：
    //    - status：已决议且恢复执行（approved/params_completed 等）→ Pending（继续执行）；
    //      其余决议（rejected/cancelled 等）→ Cancelled（事件终止）
    //    - resumeType：决议徽章数据源（与历史路径 extra.interrupt.resume.type 同值域），
    //      恢复成功后 clearDecidedInterrupts 移除中断条目，徽章由 part 自身承接
    const resolved = isInterruptResolvedStatus(resumeStatus);
    syncToolCallPartDecision(
        convId,
        decision.toolCallId,
        resolved ? ToolCallStatus.Pending : ToolCallStatus.Cancelled,
        resumeStatus,
    );
}

// ==================== 恢复失败回滚 ====================

/**
 * 恢复失败回滚：pendingInterrupts / parts 双写还原，
 * 中断回到 Pending 等待重试（对齐 tokhub revertInterruptToPending）
 */
export function revertInterruptToPending(convId: number, decision: ResumeDecision) {
    const store = useChatStore();

    // 1. pendingInterrupts：还原决策状态
    const slice = store.getSlice(convId);
    const pending = slice?.pendingInterrupts.find((i) => i.actionId === decision.interruptId);
    if (pending) {
        pending.status = InterruptResumeStatus.Pending;
    }

    // 2. parts：tool_call part 还原 interrupted，并清除旧决议类型（决策已回滚，
    //    残留 resumeType 会导致待审批卡片与决议徽章同时展示）
    syncToolCallPartDecision(convId, decision.toolCallId, ToolCallStatus.Interrupted, '');
}

// ==================== 决策清除 ====================

/** 批量恢复成功后：移除已决策中断（按 actionId / toolCallId 精确匹配） */
export function clearDecidedInterrupts(convId: number, decisions: ResumeDecision[]) {
    const store = useChatStore();
    for (const decision of decisions) {
        store.removeInterrupt(convId, {
            actionId: decision.interruptId,
            type: decision.interruptType,
            description: '',
            toolCallId: decision.toolCallId,
        });
    }
}

// ==================== 轮次终止提示 ====================

/**
 * 轮次终止提示：用户主动停止 / 流式错误，写入当前 turn 的 assistant 消息。
 * 仅前端展示（notice part 不落库，历史加载时自然消失）
 */
export function writeTurnNotice(
    convId: number,
    turnId: string | null | undefined,
    kind: NoticePart['kind'],
    text: string,
) {
    const store = useChatStore();
    store.appendTurnNoticePart(convId, turnId, {
        type: 'notice',
        id: `notice-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        kind,
        text,
    });
}

// ==================== 内部工具 ====================

/** 将 toolCallId 匹配的 tool_call part 状态置为指定值（跨所有消息） */
function syncToolCallPartStatus(convId: number, toolCallId: string | undefined, status: string) {
    syncToolCallPartDecision(convId, toolCallId, status, undefined);
}

/**
 * 将 toolCallId 匹配的 tool_call part 双写决策结果（跨所有消息）：
 * status（执行状态机）+ resumeType（决议徽章数据源，可选）
 */
function syncToolCallPartDecision(
    convId: number,
    toolCallId: string | undefined,
    status: string,
    resumeType?: string,
) {
    if (!toolCallId) return;
    const store = useChatStore();
    const slice = store.getSlice(convId);
    if (!slice) return;
    for (const msg of slice.messages) {
        if (!msg.parts) continue;
        for (const part of msg.parts) {
            if (part.type === 'tool_call' && part.toolCallId === toolCallId) {
                part.status = status as ToolCallPart['status'];
                // undefined 不动；空串清除（决策回滚）
                if (resumeType !== undefined) {
                    part.resumeType = resumeType || undefined;
                }
            }
        }
    }
}
