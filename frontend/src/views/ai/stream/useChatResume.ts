/**
 * useChatResume - 中断恢复编排
 *
 * 职责：
 * - 决策池：recordDecision 收集用户操作，经 eventWriter 双写状态
 * - 批量恢复：同 turn 全部决策后 executeResumeBatch 一次性发送（后端 resumeParams 数组协议）
 * - 并发守卫：同一时刻仅允许一条恢复流
 * - 失败回滚：revertBatchToPending 三处还原 Pending，决策保留可重试
 */

import { reactive, unref, type MaybeRef } from 'vue';
import { Msg } from '@/hooks/useI18n';
import type { ResumeEntry } from '../protocol/types';
import { isInterruptDecided } from '../registries/statuses';
import { useChatStore } from '../stores/chatStore';
import * as eventWriter from './eventWriter';
import type { ResumeDecision } from './eventWriter';
import type { InterruptActionEvent } from '../interrupt/types';

/** 恢复发送器：由宿主（ChatContainer）注入，负责建连与 WebSocket 发送，返回是否投递成功 */
export type ResumeSender = (entries: ResumeEntry[]) => Promise<boolean>;

export interface UseChatResumeOptions {
    send: ResumeSender;
}

export function useChatResume(convId: MaybeRef<number>, options: UseChatResumeOptions) {
    const store = useChatStore();

    /** 决策池：interruptId → 决策（恢复成功前保留，失败可重试） */
    const resumeDecisions = reactive(new Map<string, ResumeDecision>());

    /** 并发守卫：恢复流进行中标记 */
    let resuming = false;

    /**
     * 记录一条用户决策：
     * 入池 → eventWriter 双写（pendingInterrupts + parts）
     * → 该 turn 全部中断已决策时触发批量恢复
     */
    function recordDecision(event: InterruptActionEvent) {
        const cid = unref(convId);
        const slice = store.getSlice(cid);

        // 补全 turnId（event 缺失时从 pendingInterrupts 反查）
        let turnId = event.turnId;
        if (!turnId && slice) {
            const interrupt = slice.pendingInterrupts.find(
                (i) => i.actionId === event.interruptId || i.toolCallId === event.toolCallId,
            );
            turnId = interrupt?.turnId || '';
        }
        if (!turnId) {
            console.warn('[useChatResume] interrupt missing turnId, skip:', event.interruptId);
            return;
        }

        const decision: ResumeDecision = {
            turnId,
            interruptId: event.interruptId,
            interruptType: event.interruptType,
            action: event.action,
            payload: event.payload,
            toolCallId: event.toolCallId,
        };
        resumeDecisions.set(decision.interruptId, decision);

        // 双写：决策状态同步到 pendingInterrupts / parts
        eventWriter.writeInterruptDecision(cid, decision);

        // 该 turn 的中断全部已决策 → 批量恢复（按 turn 分组完整性检查）
        const turnInterrupts = (slice?.pendingInterrupts ?? []).filter((i) => i.turnId === turnId);
        const allDecided = turnInterrupts.length > 0 && turnInterrupts.every((i) => isInterruptDecided(i.status));
        if (allDecided) {
            executeResumeBatch(turnId);
        }
    }

    /**
     * 批量恢复：同 turn 全部决策一次性发送
     * 并发守卫 → 复用历史 assistant message → 发送 → 成功清除 / 失败回滚
     */
    async function executeResumeBatch(turnId: string) {
        const cid = unref(convId);

        // 并发守卫：同一时刻仅一条恢复流（后到的 turn 延后处理）
        if (resuming) {
            console.warn('[useChatResume] resume in flight, defer turn:', turnId);
            return;
        }

        const batch = [...resumeDecisions.values()].filter((d) => d.turnId === turnId);
        if (batch.length === 0) return;

        resuming = true;
        try {
            // 复用 / 创建该 turn 的 assistant message（恢复流内容挂载点），
            // 重新进入流式状态：resume 后模型首 token 有延迟，
            // 否则这段时间界面无任何响应表现
            store.ensureTurnAssistantMessage(cid, turnId).streaming = true;

            const entries: ResumeEntry[] = batch.map((d) => ({
                turnId: d.turnId,
                interruptId: d.interruptId,
                interruptType: d.interruptType,
                action: d.action,
                payload: d.payload,
            }));
            const sent = await options.send(entries);

            if (!sent) {
                // 恢复失败：中断/事件/parts 三处还原 Pending，决策保留供重试
                for (const decision of batch) {
                    eventWriter.revertInterruptToPending(cid, decision);
                }
                Msg.error('ai.chat.resumeFailed');
                return;
            }

            // 成功：清除决策池 + 移除 pending 中断
            eventWriter.clearDecidedInterrupts(cid, batch);
            for (const decision of batch) {
                resumeDecisions.delete(decision.interruptId);
            }
        } finally {
            resuming = false;
        }
    }

    return {
        resumeDecisions,
        recordDecision,
        executeResumeBatch,
    };
}
