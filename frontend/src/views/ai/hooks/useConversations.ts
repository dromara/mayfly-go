/**
 * useConversations - 会话列表管理
 */
import { ref } from 'vue';
import { aiApi } from '../api';
import type { Conversation } from '../protocol/types';
import { useChatStore } from '../stores/chatStore';

export function useConversations() {
    const store = useChatStore();
    const loading = ref(false);

    const loadConversations = async () => {
        loading.value = true;
        try {
            const convs = await aiApi.listConversations.request();
            store.conversations = convs;
        } finally {
            loading.value = false;
        }
    };

    const createConversation = async (title?: string): Promise<Conversation> => {
        const conv = await aiApi.createConversation.request({ title });
        store.conversations = [conv, ...store.conversations];
        return conv;
    };

    const deleteConversation = async (id: number) => {
        await aiApi.deleteConversation.request({ id });
        store.conversations = store.conversations.filter((c) => c.id !== id);
        store.resetSlice(id);
    };

    const renameConversation = async (id: number, title: string) => {
        await aiApi.renameConversation.request({ id, title });
        const conv = store.conversations.find((c) => c.id === id);
        if (conv) conv.title = title;
    };

    const selectConversation = (conv: Conversation) => {
        store.selectConv(conv);
    };

    return {
        loading,
        loadConversations,
        createConversation,
        deleteConversation,
        renameConversation,
        selectConversation,
    };
}
