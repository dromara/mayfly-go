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
                :draft="forkDraft"
                @conversation-created="onConversationCreated"
                @draft-applied="forkDraft = ''"
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
import { computed, onActivated, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';
import type { Conversation } from '../protocol/types';
import { useConversations } from '../hooks/useConversations';
import { useChatStore } from '../stores/chatStore';
import { aiApi } from '../api';
import { ChatContainer, ConversationSidebar } from './';
import type { ConversationSidebarItem } from './ConversationSidebar.vue';
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from '@/components/ui/empty';

const props = withDefaults(
    defineProps<{
        /** 是否显示会话侧栏 */
        showSidebar?: boolean;
        /**
         * 是否由路由页面挂载（只有 `AiAssistant.vue` 会传 true，全局 FAB 抽屉不传）。
         *
         * 为什么需要这个开关：本组件同时被路由页与全局抽屉挂载，实例不止一个；而 `route`
         * 是全局响应对象，跳转后两个实例的 `route.name === 'AiAssistant'` 同时成立。若不加限定，
         * 隐藏的全局抽屉实例会与页面实例抢同一份路由意图（实测：草稿被抽屉抢走、一次追问建出两个会话）。
         * 路由级意图只属于被路由驱动的那个实例
         */
        routeDriven?: boolean;
    }>(),
    { showSidebar: true, routeDriven: false }
);

const { t } = useI18n();
const route = useRoute();
const chatStore = useChatStore();
const { loadConversations, createConversation, deleteConversation, renameConversation, selectConversation } = useConversations();

const currentConvId = ref<number>(0);
const currentConversation = ref<Conversation | null>(null);

/**
 * 待投递给「本次追问新建会话」的草稿。
 *
 * 只在页面级流程里存活（从 store 取走 → 交给对应会话面板 → 应用完清空），
 * 不让多个 ChatPanel 实例共享 store 里那份可变草稿：keep-alive 缓存的旧会话面板会抢先消费，
 * 草稿被填到一个即将被切走的会话里，表现为「点了追问但输入框是空」（实测踩过）
 */
const forkDraft = ref('');

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

/** 加载会话列表；未选中任何会话时选中首个 */
const loadSessions = async () => {
    await loadConversations();
    if (!currentConvId.value && chatStore.conversations.length > 0) {
        switchSession(chatStore.conversations[0]);
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

// ==================== 「带上下文追问」投递入口 ====================

/**
 * 消费待投递草稿：有草稿就为它开一个新会话并预填输入框，返回是否消费了草稿。
 *
 * 为什么靠「页面挂载/激活时取草稿」而不是路由 query（?compose=1）：用 query 就得在跳转后再
 * replace 掉参数，同一窗口内两次路由变更会撞上布局层的 `<transition mode="out-in">`——后一次
 * 导航落在前一次 leave 未结束时，router-view 会永久只渲染空占位节点（实测必现）。
 * 草稿是一次性的内存态，取走即空，刷新或重复激活都不会再建会话，因此也不需要清 query。
 *
 * 为什么只由 routeDriven 实例处理：本组件同时被路由页与全局 FAB 抽屉挂载，隐藏的那个实例若也来
 * 抢，草稿就会被填进看不见的会话里（实测：一次追问建出两个 New Chat、页面上那个却是空的）
 */
const consumeForkDraft = async (): Promise<boolean> => {
    if (!props.routeDriven || !chatStore.draft) return false;
    // 先取走再建会话：取走即清空，保证一份草稿只被一次追问流程消费
    forkDraft.value = chatStore.consumeDraft();
    // 追问的对象是「那次运行」，一律开新会话：既不复用用户正在进行的其它会话（上下文串味），
    // 也不复用系统运行会话——它是审计产物，服务端已拒绝对其发起新对话
    await createNewSession();
    return true;
};

onMounted(async () => {
    // 追问草稿优先；没有草稿才走常规的「加载列表 + 选首个会话」
    if (!(await consumeForkDraft())) void loadSessions();
    // 校正执行中指示器：其它端/页面刷新期间开始的 turn 不在本页事件流中，
    // 以服务端运行中列表为准（拉取失败静默降级，指示器仅由本页事件驱动）
    try {
        const running = (await aiApi.listRunningTurns.request()) || [];
        chatStore.syncRunningConvs(running.map((r) => r.conversationId));
    } catch (e) {
        console.error('[ai] load running turns failed:', e);
    }
});

// keep-alive 缓存的页面再次进入不会重跑 onMounted，故激活时补消费一次草稿
onActivated(() => {
    void consumeForkDraft();
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
