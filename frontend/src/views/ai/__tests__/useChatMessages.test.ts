import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { ref } from 'vue';
import type { TurnGroupVO } from '../protocol/types';
import { aiApi } from '../api';
import { useChatStore } from '../stores/chatStore';
import { useChatMessages } from '../hooks/useChatMessages';

// 只关心「加载动作是否真的发了请求」，接口层整体打桩
vi.mock('../api', () => ({
    aiApi: { getTurnItems: { request: vi.fn() } },
}));

const request = aiApi.getTurnItems.request as unknown as ReturnType<typeof vi.fn>;

/** 一轮最小历史（用户消息 + assistant 正文） */
function group(turnId: string): TurnGroupVO {
    return {
        turnId,
        items: [
            {
                id: 1,
                turnId,
                itemType: 'message',
                itemId: `u-${turnId}`,
                status: 'success',
                createTime: '2026-10-14T10:00:00',
                item: { type: 'message', id: `u-${turnId}`, role: 'user', content: [{ type: 'input_text', text: '巡检' }] },
            },
            {
                id: 2,
                turnId,
                itemType: 'message',
                itemId: `a-${turnId}`,
                status: 'success',
                createTime: '2026-10-14T10:00:10',
                item: { type: 'message', id: `a-${turnId}`, role: 'assistant', content: [{ type: 'output_text', text: '已完成' }] },
            },
        ],
    };
}

/**
 * 历史加载对「切片是否存在」不该敏感。
 *
 * 回归点：聊天页会先 selectConv 创建切片，所以早期实现里 loadMessages 用 getSlice 取不到就直接返回；
 * 只读轨迹（运维运行回放）是按 conversationId 直接拉取的，走到这条早退分支就表现为
 * 「轨迹永远空白且无任何报错」——因此这里钉住：没切片也要发请求。
 */
describe('useChatMessages 加载', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        request.mockReset();
    });

    it('切片不存在时也会请求接口并写入消息', async () => {
        const store = useChatStore();
        const cid = 900;
        request.mockResolvedValue([group('t1')]);

        await useChatMessages(ref(cid)).loadMessages();

        expect(request).toHaveBeenCalledTimes(1);
        expect(store.getSlice(cid)?.messages.length).toBeGreaterThan(0);
    });

    it('已加载过的会话重复调用不再请求（缓存语义保留）', async () => {
        const cid = 901;
        request.mockResolvedValue([group('t1')]);
        const { loadMessages } = useChatMessages(ref(cid));

        await loadMessages();
        await loadMessages();

        expect(request).toHaveBeenCalledTimes(1);
    });

    it('resetSlice 后重新加载（轨迹关闭再打开拿到的是新数据）', async () => {
        const store = useChatStore();
        const cid = 902;
        request.mockResolvedValue([group('t1')]);
        const { loadMessages } = useChatMessages(ref(cid));

        await loadMessages();
        store.resetSlice(cid);
        await loadMessages();

        expect(request).toHaveBeenCalledTimes(2);
    });
});
