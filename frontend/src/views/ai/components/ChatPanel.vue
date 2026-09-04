<template>
    <div class="chat-panel">
        <div class="chat-panel__content">
            <!-- ScrollerProvider 包裹消息列表与输入框（对齐 tokhub：ChatInput 在 Provider 内
                 可 useMessageScroller().scrollToEnd 实现发送后锚定） -->
            <MessageScrollerProvider :auto-scroll="true" default-scroll-position="last-anchor">
                <!-- 消息列表 -->
                <MessageList
                    ref="messageListRef"
                    :messages="messages"
                    :loading="loading"
                    :sender-loading="senderLoading"
                    :has-more="hasMore"
                    :loading-more="loadingMore"
                    :pending-interrupts="interrupts"
                    :editing-message-id="editingMessageId"
                    :conversation-id="convId"
                    :scroll-anchor="scrollAnchor"
                    @load-more="$emit('load-more')"
                    @interrupt-action="(action) => $emit('interrupt-action', action)"
                    @scroll-anchor-change="onScrollAnchorChange"
                    @edit-message="onEditMessage"
                    @edit-send="onEditSend"
                    @suggest="onSuggest"
                    @cancel-edit="editingMessageId = null"
                />

                <!-- 待发送队列（对齐 tokhub PendingSendQueue） -->
                <PendingSendQueue
                    :queue="queuedMessages"
                    @edit="onEditQueued"
                    @remove="(id) => store.removeQueuedMessage(convId, id)"
                    @reorder="(from, to) => store.reorderQueuedMessage(convId, from, to)"
                />

                <!-- 输入框 -->
                <ChatInput
                    ref="chatInputRef"
                    :loading="senderLoading"
                    :disabled="false"
                    :auto-focus="true"
                    :placeholder="t('ai.chat.inputPlaceholder')"
                    :skills="skills"
                    :should-queue="shouldQueue"
                    @submit="onSubmit"
                    @queue="onQueue"
                    @cancel="$emit('cancel')"
                />
            </MessageScrollerProvider>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * ChatPanel - 纯 UI 层（parts 驱动）
 * 对齐 tokhub 的 ChatPanel：消息直接使用 store 的 parts，不做二次合并
 *
 * 编排职责（对齐 tokhub ChatPanel.tsx）：
 * - shouldQueue 派生态：Agent 未完全空闲（回复中/待中断）时新消息一律入队
 * - 队列空闲出队：shouldQueue 解除后自动发送队首消息
 * - 用户消息原地编辑：编辑后作为新一轮消息发送（空闲）或入队（忙）
 */
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { InterruptActionEvent } from '../interrupt/types';
import type { ChatMessage, InterruptEvent } from '../protocol/types';
import type { ChatInputSubmitData, SkillItem } from '../input/types';
import ChatInput from '../input/ChatInput.vue';
import MessageList from '../message/MessageList.vue';
import PendingSendQueue from './PendingSendQueue.vue';
import { useChatStore } from '../stores/chatStore';
import type { QueuedMessage } from '../stores/chatStore';
import { MessageScrollerProvider } from '@/components/ui/message-scroller';

const { t } = useI18n();

const props = defineProps<{
    messages: ChatMessage[];
    loading: boolean;
    senderLoading: boolean;
    interrupts: InterruptEvent[];
    hasMore: boolean;
    /** 可用技能列表 */
    skills?: SkillItem[];
    /** 当前会话 ID（队列/滚动锚点按会话隔离） */
    convId: number;
}>();

const emit = defineEmits<{
    (e: 'send', data: ChatInputSubmitData): void;
    (e: 'load-more'): void;
    (e: 'cancel'): void;
    (e: 'interrupt-action', event: InterruptActionEvent): void;
}>();

const store = useChatStore();
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null);
const messageListRef = ref<InstanceType<typeof MessageList> | null>(null);

// ==================== 消息队列（对齐 tokhub queuedMessages / shouldQueue） ====================

const queuedMessages = computed(() => store.getSlice(props.convId)?.queuedMessages ?? []);

/** Agent 未完全空闲（回复中或待处理中断）时入队，参考 Claude Code / Codex */
const shouldQueue = computed(() => props.senderLoading || props.interrupts.length > 0);

const scrollAnchor = computed(() => store.getScrollAnchor(props.convId));

const loadingMore = computed(() => store.getSlice(props.convId)?.isLoadingMoreMessages || false);

const onSubmit = (data: ChatInputSubmitData) => {
    emit('send', data);
    // 发送后贴底（对齐 tokhub：ChatInput 发送后 scrollToEnd）。
    // rAF 等待乐观消息 append 并由引擎处理完 anchor 分支后再执行，
    // 覆盖为 following-bottom 模式并真正贴到最底部
    requestAnimationFrame(() => messageListRef.value?.scrollToEnd());
};

/** 入队（shouldQueue 时 ChatInput 提交转队列，完整提交数据随项保存） */
const onQueue = (data: ChatInputSubmitData) => {
    store.enqueueMessage(props.convId, data);
};

/** 队列空闲出队：shouldQueue 解除且队列非空时发送队首（对齐 tokhub dequeue 时机） */
watch(
    [shouldQueue, queuedMessages] as const,
    ([busy, queue]) => {
        if (busy || queue.length === 0) return;
        const first = store.dequeueMessage(props.convId);
        if (first) {
            emit('send', first.data);
            requestAnimationFrame(() => messageListRef.value?.scrollToEnd());
        }
    },
    { flush: 'post' },
);

// ==================== 队列项编辑/删除 ====================

/** 编辑队列消息：移出队列并回填输入框（含已上传附件，对齐 tokhub handleEditQueued） */
const onEditQueued = (message: QueuedMessage) => {
    store.removeQueuedMessage(props.convId, message.id);
    chatInputRef.value?.setValue(message.data.text, message.data.attachments);
};

// ==================== 空态建议芯片 ====================

/** 点击建议：回填输入框并聚焦（不直接发送，用户可先 @ 引用资源） */
const onSuggest = (text: string) => {
    chatInputRef.value?.setValue(text);
};

// ==================== 用户消息原地编辑（对齐 tokhub editingMessageId） ====================

const editingMessageId = ref<string | null>(null);

const onEditMessage = (messageId: string) => {
    editingMessageId.value = messageId;
};

/** 编辑后发送：作为新一轮消息（忙时入队，对齐 tokhub handleEditSend；纯文本重发无附件） */
const onEditSend = (content: string) => {
    const trimmed = content.trim();
    if (!trimmed) return;
    editingMessageId.value = null;
    const data: ChatInputSubmitData = { text: trimmed, segments: [] };
    if (shouldQueue.value) {
        store.enqueueMessage(props.convId, data);
    } else {
        emit('send', data);
        requestAnimationFrame(() => messageListRef.value?.scrollToEnd());
    }
};

// ==================== 滚动锚点持久化（对齐 tokhub scrollAnchorStore） ====================

const onScrollAnchorChange = (anchor: { id: string; offset: number }) => {
    store.setScrollAnchor(props.convId, anchor);
};
</script>

<style scoped>
.chat-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    background: var(--el-bg-color);
}

.chat-panel__content {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    width: 100%;
    /* 对齐 tokhub 层级：面板不限宽，消息列 max-w-4xl、
       Composer max-w-3xl 各自限宽居中 */
}
</style>
