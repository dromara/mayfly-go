import { beforeEach, describe, expect, it } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useChatStore } from '../stores/chatStore';

/**
 * 追问草稿（跨页面一次性投递）的契约：
 * 投递方只 stage，消费方只 consume，且取走即清空——重复消费会把同一段上下文填两次。
 */
describe('chatStore 追问草稿', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    it('consume 取回草稿并清空，再次 consume 为空', () => {
        const store = useChatStore();
        expect(store.consumeDraft()).toBe('');

        store.stageDraft('【运行上下文】\n请基于以上运行上下文继续分析：');
        expect(store.draft).not.toBe('');
        expect(store.consumeDraft()).toContain('运行上下文');
        expect(store.draft).toBe('');
        expect(store.consumeDraft()).toBe('');
    });

    it('后投递覆盖先投递（同一时刻只有一份待填草稿）', () => {
        const store = useChatStore();
        store.stageDraft('第一次');
        store.stageDraft('第二次');
        expect(store.consumeDraft()).toBe('第二次');
    });
});
