<template>
    <div class="ai-assistant-body">
        <!-- 左侧会话列表（抽屉场景可通过 showSidebar 折叠，v-show 保留列表滚动位置） -->
        <ConversationSidebar
            v-show="showSidebar"
            :items="sidebarItems"
            :active="String(currentConvId)"
            @create="createNewSession"
            @change="onChangeSession"
            @rename="onRenameSession"
            @delete="(item) => onDeleteSession(Number(item.key))"
        />

        <!-- 右侧聊天区域 -->
        <main class="ai-assistant-body__chat">
            <ChatContainer
                v-if="currentConversation"
                :conversation="currentConversation"
                @conversation-created="onConversationCreated"
            />
            <div v-else class="ai-assistant-body__empty">
                <Empty>
                    <EmptyMedia variant="icon">
                        <MessageCircleMoreIcon />
                    </EmptyMedia>
                    <EmptyHeader>
                        <EmptyTitle>{{ t('ai.assistant.startNewConversation') }}</EmptyTitle>
                    </EmptyHeader>
                    <EmptyContent>
                        <EmptyDescription>{{ t('ai.assistant.selectOrCreateSession') }}</EmptyDescription>
                    </EmptyContent>
                </Empty>
            </div>
        </main>
    </div>
</template>

<script setup lang="ts" name="AiAssistantBody">
/**
 * AiAssistantBody - AI 助手主体（会话侧栏 + 聊天区）
 *
 * AiAssistant 页面与全局 AiAssistantFab 抽屉共用的单一实现：
 * 会话加载/切换/重命名/删除、运行中指示器校正等逻辑收敛于此，
 * 消费方只决定布局容器（页面全屏 / 抽屉半屏）与侧栏显隐。
 */
import { formatDate } from '@/common/utils/format';
import { Msg } from '@/hooks/useI18n';
import { MessageCircleMoreIcon } from '@lucide/vue';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { Conversation } from '../protocol/types';
import { useConversations } from '../hooks/useConversations';
import { useChatStore } from '../stores/chatStore';
import { aiApi } from '../api';
import { ChatContainer, ConversationSidebar } from './';
import type { ConversationSidebarItem } from './ConversationSidebar.vue';
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty';

withDefaults(
    defineProps<{
        /** 是否显示会话侧栏 */
        showSidebar?: boolean;
    }>(),
    { showSidebar: true }
);

const { t } = useI18n();
const chatStore = useChatStore();
const { loadConversations, createConversation, deleteConversation, renameConversation, selectConversation } = useConversations();

const currentConvId = ref<number>(0);
const currentConversation = ref<Conversation | null>(null);

/** 侧边栏会话列表 */
const sidebarItems = computed((): ConversationSidebarItem[] => {
    return chatStore.conversations.map((conv: Conversation) => ({
        key: String(conv.id),
        label: conv.title,
        // 侧栏定宽 w-72：短格式时间避免挤压标题（完整时间悬停气泡可见）
        createTime: formatDate(conv.createTime, 'MM-DD HH:mm'),
        updateTime: formatDate(conv.updateTime),
        // 执行中指示器：turn 级事件维护 + 进入页面时服务端校正
        running: chatStore.isConvRunning(conv.id),
    }));
});

/** 加载会话列表 */
const loadSessions = async () => {
    try {
        await loadConversations();
        if (!currentConvId.value && chatStore.conversations.length > 0) {
            switchSession(chatStore.conversations[0]);
        }
    } finally {
        // loading 由 composable 管理
    }
};

const onChangeSession = (item: ConversationSidebarItem) => {
    const conv = chatStore.conversations.find((c: Conversation) => String(c.id) === item.key);
    if (conv) switchSession(conv);
};

const createNewSession = async () => {
    const conv = await createConversation();
    switchSession(conv);
};

const switchSession = (conv: Conversation) => {
    currentConvId.value = conv.id;
    currentConversation.value = conv;
    selectConversation(conv);
};

const onConversationCreated = async () => {
    await loadSessions();
};

const onRenameSession = async (item: ConversationSidebarItem, title: string) => {
    await renameConversation(Number(item.key), title);
    Msg.operateSuccess();
    await loadSessions();
};

const onDeleteSession = async (convId: number) => {
    await deleteConversation(convId);
    if (currentConvId.value === convId) {
        currentConvId.value = 0;
        currentConversation.value = null;
    }
    await loadSessions();
};

onMounted(async () => {
    loadSessions();
    // 校正执行中指示器：其它端/页面刷新期间开始的 turn 不在本页事件流中，
    // 以服务端运行中列表为准（拉取失败静默降级，指示器仅由本页事件驱动）
    try {
        const running = (await aiApi.listRunningTurns.request()) || [];
        chatStore.syncRunningConvs(running.map((r) => r.conversationId));
    } catch (e) {
        console.error('[ai] load running turns failed:', e);
    }
});
</script>

<style scoped>
.ai-assistant-body {
    display: flex;
    gap: 12px;
    height: 100%;
    min-height: 0;
    background: var(--el-bg-color);
}

.ai-assistant-body__chat {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
}

.ai-assistant-body__empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--el-text-color-placeholder);
}
</style>
