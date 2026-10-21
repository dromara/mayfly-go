/**
 * 乐观消息回滚的契约
 *
 * 为什么单独测：`sendMessage` 返回 true 只代表字节写进了 socket，不代表后端接了这轮
 * （上一轮还在收尾时后端会直接拒收 `conversation has a running turn`）。那时乐观气泡已渲染、
 * 却永远等不到回复，界面就留着一条骗人的消息。回滚动作全靠 removeMessage，
 * 它若删错、删漏或在不该删时删掉真实记录，用户看到的历史就被改坏了。
 */
import { beforeEach, describe, expect, it } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useChatStore } from '../stores/chatStore';
import type { ChatMessage } from '../protocol/types';

const msg = (id: string, text: string, role: ChatMessage['role'] = 'user'): ChatMessage => ({
    id,
    role,
    content: text,
});

describe('chatStore 乐观消息回滚', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    it('按 id 精确删除，其余消息与顺序不受影响', () => {
        const store = useChatStore();
        store.addMessage(7, msg('u-1', '第一条'));
        const optimisticId = store.addMessage(7, msg('u-2', '未被接受的一条'));
        store.addMessage(7, msg('a-1', '回复', 'assistant'));

        store.removeMessage(7, optimisticId);

        const slice = store.getSlice(7)!;
        expect(slice.messages.map((m) => m.id)).toEqual(['u-1', 'a-1']);
    });

    it('同文本的另一条不被误删（只认 id）', () => {
        const store = useChatStore();
        const first = store.addMessage(7, msg('u-a', '重发同一条'));
        store.addMessage(7, msg('u-b', '重发同一条'));

        store.removeMessage(7, first);

        const left = store.getSlice(7)!.messages;
        expect(left).toHaveLength(1);
        expect(left[0].id).toBe('u-b');
    });

    it('id 不存在时是安全 no-op（重复回滚不会抹掉真实消息）', () => {
        const store = useChatStore();
        store.addMessage(7, msg('u-1', '真实消息'));

        store.removeMessage(7, '不存在的-id');
        store.removeMessage(7, '不存在的-id');

        expect(store.getSlice(7)!.messages.map((m) => m.id)).toEqual(['u-1']);
    });

    it('会话切片不存在时不抛错（回滚可能发生在切片已被 LRU 淘汰之后）', () => {
        const store = useChatStore();
        expect(() => store.removeMessage(999, 'u-1')).not.toThrow();
    });

    it('addMessage 返回的 id 可直接用于回滚（两者必须同源）', () => {
        const store = useChatStore();
        const id = store.addMessage(7, msg('u-x', '待回滚'));
        expect(store.getSlice(7)!.messages.some((m) => m.id === id)).toBe(true);

        store.removeMessage(7, id);
        expect(store.getSlice(7)!.messages).toHaveLength(0);
    });
});
