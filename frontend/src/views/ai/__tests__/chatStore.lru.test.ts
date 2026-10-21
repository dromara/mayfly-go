/**
 * 会话切片缓存的 LRU 淘汰契约
 *
 * 为什么单独测：此前 `slice.active` 恒为 true（全仓没有任何把它置 false 的写入点），
 * evictIfNeeded 每次循环都零删除——文件头宣称的「LRU 淘汰：MAX_CACHED_CONVS = 8」完全失效，
 * 切过 N 个会话就常驻 N 份完整消息数组（含工具明细与流式中间态），长会话页面稳定增长到数十 MB。
 * 这类「缓存上限写在那里但永远不触发」的失效，只有把淘汰条件本身拿出来断言才看得见。
 */
import { beforeEach, describe, expect, it } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { MAX_CACHED_CONVS, useChatStore } from '../stores/chatStore';
import type { useChatStore as UseChatStore } from '../stores/chatStore';

type ChatStore = ReturnType<typeof UseChatStore>;

/** 隔开访问时间戳：同一毫秒内 lastAccessAt 相同，LRU 顺序无法判定 */
const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

const touch = (store: ChatStore, convId: number) => store.addMessage(convId, { id: `m-${convId}`, role: 'user', content: 'x' });

describe('chatStore 会话切片 LRU 淘汰', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    it('超过上限时淘汰最久未访问的会话，保留最近访问的', async () => {
        const store = useChatStore();
        const total = MAX_CACHED_CONVS + 3;
        for (let id = 1; id <= total; id++) {
            touch(store, id);
            await sleep(2);
        }

        const alive = Array.from({ length: total }, (_, i) => i + 1).filter((id) => store.getSlice(id));
        expect(alive.length).toBeLessThanOrEqual(MAX_CACHED_CONVS);
        expect(store.getSlice(1)).toBeUndefined();
        expect(store.getSlice(total)).toBeDefined();
    });

    it('回看旧会话会刷新它，之后被淘汰的是更久未访问的会话', async () => {
        const store = useChatStore();
        for (let id = 1; id <= MAX_CACHED_CONVS; id++) {
            touch(store, id);
            await sleep(2);
        }

        // 回看 1 号（LRU 的「最近使用」必须包含读，否则常被回看的会话反而先被丢）
        store.getOrCreateSlice(1);
        for (let id = MAX_CACHED_CONVS + 1; id <= MAX_CACHED_CONVS + 2; id++) {
            touch(store, id);
            await sleep(2);
        }

        expect(store.getSlice(1)).toBeDefined();
        expect(store.getSlice(2)).toBeUndefined();
    });

    it('正在流式的会话不可淘汰（否则生成中突然空白）', async () => {
        const store = useChatStore();
        const streamingConv = 1;
        touch(store, streamingConv);
        store.getOrCreateSlice(streamingConv).activeTurnId = 'turn-1';
        await sleep(2);

        for (let id = 2; id <= MAX_CACHED_CONVS + 3; id++) {
            touch(store, id);
            await sleep(2);
        }

        expect(store.getSlice(streamingConv)).toBeDefined();
    });

    it('淘汰后重新进入该会话可再次创建切片（数据按需重取，不留空壳）', async () => {
        const store = useChatStore();
        for (let id = 1; id <= MAX_CACHED_CONVS + 2; id++) {
            touch(store, id);
            await sleep(2);
        }
        expect(store.getSlice(1)).toBeUndefined();

        const recreated = store.getOrCreateSlice(1);
        expect(recreated.messages).toHaveLength(0);
        expect(recreated.loaded).toBe(false);
    });
});
