/**
 * useChatMessages - 历史消息加载 + 分页
 * 对齐 tokhub 的 useChatMessages 模式
 */
import { ref, unref, type MaybeRef } from 'vue';
import { aiApi } from '../api';
import { useChatStore } from '../stores/chatStore';
import type { ContentSegment, MessageAttachment } from '../protocol/types';

export function useChatMessages(convId: MaybeRef<number>) {
    const store = useChatStore();
    const loading = ref(false);

    /**
     * 加载初始消息（最新一批 turn groups）
     */
    const loadMessages = async () => {
        const cid = unref(convId);
        const slice = store.getSlice(cid);
        if (!slice || slice.loaded) return;

        loading.value = true;
        try {
            const groups = await aiApi.getTurnItems.request({ id: cid, limit: 10 });
            store.loadTurnGroups(cid, groups);

            if (groups.length > 0) {
                store.setOldestTurnId(cid, groups[groups.length - 1].turnId);
                store.setHasMore(cid, groups.length >= 10);
            } else {
                store.setHasMore(cid, false);
            }
        } finally {
            loading.value = false;
        }
    };

    /**
     * 强制重新加载（忽略 loaded 缓存）：attach 目标 turn 已结束（turn_not_running）
     * 而本地流式中间态已被回放前置重置清除时，从服务端拉取落库终态回填
     */
    const reloadMessages = async () => {
        const cid = unref(convId);
        const slice = store.getSlice(cid);
        if (slice) {
            slice.messages = [];
            slice.loaded = false;
        }
        await loadMessages();
    };

    /**
     * 加载更多历史消息（向前翻页）
     */
    const loadMoreMessages = async () => {
        const cid = unref(convId);
        const slice = store.getSlice(cid);
        if (!slice || !slice.hasMoreMessages || slice.isLoadingMoreMessages) return;

        store.setLoadingMore(cid, true);
        try {
            const groups = await aiApi.getTurnItems.request({
                id: cid,
                beforeTurnId: slice.oldestTurnId || undefined,
                limit: 20,
            });
            store.loadTurnGroups(cid, groups);

            if (groups.length > 0) {
                store.setOldestTurnId(cid, groups[groups.length - 1].turnId);
            }
            // 与本次请求 limit(20) 一致：不足一批说明已到最早，无需再请求
            store.setHasMore(cid, groups.length >= 20);
        } finally {
            store.setLoadingMore(cid, false);
        }
    };

    /**
     * 添加用户消息到列表（乐观更新，附件作为元数据供前端渲染）
     */
    const addUserMessage = (content: string, attachments?: MessageAttachment[], segments?: ContentSegment[]) => {
        store.addMessage(unref(convId), {
            id: `user-${Date.now()}`,
            role: 'user',
            content,
            ...(segments?.length ? { segments } : {}),
            ...(attachments?.length ? { attachments } : {}),
        });
    };

    return {
        loading,
        loadMessages,
        reloadMessages,
        loadMoreMessages,
        addUserMessage,
    };
}
