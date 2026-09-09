/**
 * useChatCallbacks - 流式事件回调分发到 Store
 * buildCallbacks 模式
 */
import { unref, type MaybeRef } from 'vue';
import { useI18n } from 'vue-i18n';
import type { InterruptEvent, ItemDelta, TurnItem } from '../protocol/types';
import { useChatStore } from '../stores/chatStore';
import { writeInterrupted, writeTurnNotice } from './eventWriter';
import { TurnStatus } from '../registries/statuses';

export function useChatCallbacks(convId: MaybeRef<number>) {
    const store = useChatStore();
    const { t } = useI18n();

    /** i18n key（如 ai.chat.connectionDisconnected）翻译，原文错误截断后透传 */
    const resolveErrorText = (error: string): string => {
        if (/^[a-z][a-zA-Z0-9]*(\.[a-zA-Z0-9]+)+$/.test(error)) return t(error);
        return error.length > 300 ? `${error.slice(0, 300)}…` : error;
    };

    return {
        onTurnStart(turnId: string, conversationId?: number) {
            const slice = store.getOrCreateSlice(unref(convId));
            slice.activeTurnId = turnId;
            // 运行中指示器：turn 开始，会话进入执行中
            store.markConvRunning(conversationId ?? unref(convId));
        },

        onItemStarted(item: TurnItem, turnId: string) {
            store.onItemStarted(unref(convId), item, turnId);
        },

        onItemUpdated(itemId: string, delta: ItemDelta, turnId: string) {
            store.onItemUpdated(unref(convId), itemId, delta, turnId);
        },

        onItemCompleted(item: TurnItem, turnId: string) {
            store.onItemCompleted(unref(convId), item, turnId);
        },

        onInterrupted(interrupt: InterruptEvent, turnId: string) {
            // 委托 eventWriter：登记 pendingInterrupts + parts 双写 interrupted
            writeInterrupted(unref(convId), interrupt, turnId);
        },

        // usage（token 用量）由 StreamCallbacks 透传，当前 UI 无展示需求，
        // 未来需要时在此声明参数即可，协议层无需变动
        onTurnCompleted(turnId: string, status?: string) {
            store.finalizeTurn(unref(convId), turnId);
            // 运行中指示器：turn 终态，会话退出执行中
            store.markConvIdle(unref(convId));
            // 显式停止：写入“已停止”提示（由服务端 stop 协议驱动的终态，唯一真正的停止路径）
            if (status === TurnStatus.Stopped) {
                writeTurnNotice(unref(convId), turnId, 'stopped', t('ai.chat.turnStopped'));
            }
        },

        /** attach 成功：回放完毕，续上流式输出（恢复 activeTurnId 与 assistant 消息流式态） */
        onTurnAttached(turnId: string, _conversationId?: number) {
            const cid = unref(convId);
            store.getOrCreateSlice(cid).activeTurnId = turnId;
            // 确保 turn 存在 assistant message（复用历史消息或新建，承接续流内容）
            store.ensureTurnAssistantMessage(cid, turnId).streaming = true;
            store.markConvRunning(cid);
        },

        /** attach 目标无运行中 turn：校正指示器（历史已由 loadMessages 加载） */
        onTurnNotRunning() {
            store.markConvIdle(unref(convId));
        },

        onError(error: string, source?: string) {
            console.error(`[ChatStream Error] (${source || 'unknown'}):`, error);
            // 错误可见化：写入当前 turn 的 assistant 消息（此前仅 console，用户看到的是"静默结束"）
            const cid = unref(convId);
            const slice = store.getSlice(cid);
            writeTurnNotice(cid, slice?.activeTurnId, 'error', resolveErrorText(error));
        },

        onEnd() {
            const slice = store.getSlice(unref(convId));
            if (slice?.activeTurnId) {
                store.finalizeTurn(unref(convId), slice.activeTurnId);
            }
        },

        onConversationCreated(conversationId: number) {
            store.selectedConvId = conversationId;
        },
    };
}
