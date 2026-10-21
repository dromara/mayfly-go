/**
 * useChatMessages - 历史消息加载 + 分页
 * useChatMessages 模式
 */
import { ref, unref, type MaybeRef } from 'vue';
import { aiApi } from '../api';
import { useChatStore } from '../stores/chatStore';
import type { ContentSegment, MessageAttachment } from '../protocol/types';

/**
 * 历史拉取批大小：请求 limit 与「是否还有更多」的判定必须同值，因此各自收成一个常量，
 * 不再在 request 与 setHasMore 两处各写一个裸数字（改一处忘改另一处会永久少拉一页）
 */
const HISTORY_PAGE_SIZE = 10;
const HISTORY_LOAD_MORE_SIZE = 20;

export function useChatMessages(convId: MaybeRef<number>) {
    const store = useChatStore();
    const loading = ref(false);

    /**
     * 加载初始消息（最新一批 turn groups）
     *
     * 切片不存在时要建而不是直接返回：只有聊天页会先 selectConv 创建切片，
     * 只读轨迹（运维运行回放）是直接按 conversationId 拉取，用 getSlice 取不到就会
     * 表现为「轨迹永远是空」且无任何报错
     */
    const loadMessages = async () => {
        const cid = unref(convId);
        const slice = store.getOrCreateSlice(cid);
        if (slice.loaded) return;

        loading.value = true;
        try {
            const groups = await aiApi.getTurnItems.request({ id: cid, limit: HISTORY_PAGE_SIZE });
            store.loadTurnGroups(cid, groups);

            if (groups.length > 0) {
                store.setOldestTurnId(cid, groups[groups.length - 1].turnId);
                store.setHasMore(cid, groups.length >= HISTORY_PAGE_SIZE);
            } else {
                store.setHasMore(cid, false);
            }
        } catch (e) {
            // 失败不能只靠请求层 toast：本地会停在「加载完成但空列表」，用户看不出是错了还是真没数据
            console.error('[ai] load history failed:', e);
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
        const slice = store.getOrCreateSlice(cid);
        if (!slice.hasMoreMessages || slice.isLoadingMoreMessages) return;

        store.setLoadingMore(cid, true);
        try {
            const groups = await aiApi.getTurnItems.request({
                id: cid,
                beforeTurnId: slice.oldestTurnId || undefined,
                limit: HISTORY_LOAD_MORE_SIZE,
            });
            store.loadTurnGroups(cid, groups);

            if (groups.length > 0) {
                store.setOldestTurnId(cid, groups[groups.length - 1].turnId);
            }
            // 与本次请求 limit 一致：不足一批说明已到最早，无需再请求
            store.setHasMore(cid, groups.length >= HISTORY_LOAD_MORE_SIZE);
        } catch (e) {
            console.error('[ai] load more history failed:', e);
        } finally {
            store.setLoadingMore(cid, false);
        }
    };

    /**
     * 添加用户消息到列表（乐观更新，附件作为元数据供前端渲染）。
     * 返回消息 id：投递失败时调用方据此回滚，不能只加不撤
     */
    const addUserMessage = (content: string, attachments?: MessageAttachment[], segments?: ContentSegment[]): string => {
        return store.addMessage(unref(convId), {
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
