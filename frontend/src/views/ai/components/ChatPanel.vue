<template>
    <div class="chat-panel">
        <div class="chat-panel__content">
            <!-- ScrollerProvider 包裹消息列表与输入框（ChatInput 在 Provider 内
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
                    mode="chat"
                    @load-more="$emit('load-more')"
                    @interrupt-action="(action) => $emit('interrupt-action', action)"
                    @scroll-anchor-change="onScrollAnchorChange"
                    @edit-message="onEditMessage"
                    @edit-send="onEditSend"
                    @suggest="onSuggest"
                    @cancel-edit="editingMessageId = null"
                />

                <!-- 待发送队列（PendingSendQueue） -->
                <PendingSendQueue
                    :queue="queuedMessages"
                    @edit="onEditQueued"
                    @remove="(id) => store.removeQueuedMessage(convId, id)"
                    @reorder="(from, to) => store.reorderQueuedMessage(convId, from, to)"
                />

                <!-- 输入框（placeholder 由 ChatInput 从触发器注册表自动拼接） -->
                <ChatInput
                    ref="chatInputRef"
                    :loading="senderLoading"
                    :auto-focus="true"
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
 * ChatPanel：消息直接使用 store 的 parts，不做二次合并
 *
 * 编排职责：
 * - shouldQueue 派生态：Agent 未完全空闲（回复中/待中断）时新消息一律入队
 * - 队列空闲出队：shouldQueue 解除后自动发送队首消息
 * - 用户消息原地编辑：编辑后作为新一轮消息发送（空闲）或入队（忙）
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';
import type { InterruptActionEvent } from '../interrupt/types';
import type { ChatMessage, InterruptEvent } from '../protocol/types';
import type { ChatInputSubmitData, SkillItem } from '../input/types';
import type { ChipSegment } from '../input/chipRegistry';
import { buildSubmitData } from '../input/submitData';
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
    /**
     * 待预填的追问草稿（由页面级流程递给「刚为追问新建的那个会话」）。
     * 经 props 而非 store 投递：面板实例很多（聊天页 + keep-alive 缓存 + 全局 FAB 抽屉），
     * 共享一份可变草稿会被先挂载的那个面板抢消费，导致预填落在另一个会话上而丢内容
     */
    draft?: string;
}>();

const emit = defineEmits<{
    (e: 'send', data: ChatInputSubmitData): void;
    (e: 'load-more'): void;
    (e: 'cancel'): void;
    (e: 'interrupt-action', event: InterruptActionEvent): void;
    (e: 'draft-applied'): void;
}>();

const store = useChatStore();
const chatInputRef = ref<InstanceType<typeof ChatInput> | null>(null);
const messageListRef = ref<InstanceType<typeof MessageList> | null>(null);

// ==================== 消息队列（queuedMessages / shouldQueue） ====================

const queuedMessages = computed(() => store.getSlice(props.convId)?.queuedMessages ?? []);

/**
 * 忙判定必须包含「本会话仍有活跃 turn」。
 *
 * 只看 senderLoading + interrupts 会错开闸：拒绝审批的瞬间 interrupts 已清空、
 * senderLoading 也已置 false，但后端那一轮仍在收尾（agent 拿到拒绝结果后还要继续推理
 * 并生成回复），此时把队列消息发过去必被 `conversation has a running turn` 拒掉——
 * 而乐观气泡已经渲染，用户看到「已发送」、库里却没有这条消息（实测复现过两次）。
 */
const shouldQueue = computed(() => props.senderLoading || props.interrupts.length > 0);

const scrollAnchor = computed(() => store.getScrollAnchor(props.convId));

const loadingMore = computed(() => store.getSlice(props.convId)?.isLoadingMoreMessages || false);

const onSubmit = (data: ChatInputSubmitData) => {
    emit('send', data);
    // 发送后贴底（ChatInput 发送后 scrollToEnd）。
    // rAF 等待乐观消息 append 并由引擎处理完 anchor 分支后再执行，
    // 覆盖为 following-bottom 模式并真正贴到最底部
    requestAnimationFrame(() => messageListRef.value?.scrollToEnd());
};

/** 入队（shouldQueue 时 ChatInput 提交转队列，完整提交数据随项保存） */
const onQueue = (data: ChatInputSubmitData) => {
    store.enqueueMessage(props.convId, data);
};

/**
 * 队列空闲出队：预读队首并发送，**只有确认后端接了这轮才摘掉队列项**。
 *
 * 不能沿用「先 dequeue 再发送」：拒绝审批的瞬间 interrupts 已清空（shouldQueue 解除），
 * 但后端那一轮仍在收尾，此时发过去会被 `conversation has a running turn` 拒掉——
 * 队列已空、消息无从找回（实测就丢过一条编辑重发的消息，用户看到的却是「已发送」）。
 *
 * 确认信号用 activeTurnId 的变化，而不是 senderLoading：后者是 onSend 里前端自己无条件
 * 置的，不代表后端接受（上一版就错在这里，复验直接失败）；activeTurnId 在 onItemStarted
 * 里赋新 turn 值，被拒时后端不会起新 turn、它保持不变——「变了」才是被接受的充分证据。
 *
 * 另：不能拿 activeTurnId 非空当「在跑」闸门——它全仓没有在 turn 终态时清空的点（只有
 * 重放后与 attach 前重置），拿它当闸门会让队列永久卡死（实测踩过）。
 */
const FLUSH_CONFIRM_MS = 4000;
let flushingId: string | null = null;
/** 发起 flush 时的活跃 turn；它变成不同值即说明后端真起了新一轮 */
let flushBaseTurnId: string | null = null;
let flushTimer = 0;

const tryFlush = () => {
    if (shouldQueue.value || flushingId) return;
    const first = store.peekQueuedMessage(props.convId);
    if (!first) return;
    flushingId = first.id;
    flushBaseTurnId = store.getSlice(props.convId)?.activeTurnId ?? null;
    emit('send', first.data);
    // 超时仍未确认被接受：只释放闸门并提示，**绝不主动重发**——emit('send') 已经加过乐观
    // 气泡并真的发出去了，主动重发在「后端其实已接受、只是确认信号没捕获到」时会落库两条重复
    // 消息（比原缺陷更糟）。队列项从未删除，下一次 shouldQueue / 队列变化会自然再触发尝试
    window.clearTimeout(flushTimer);
    flushTimer = window.setTimeout(() => {
        if (flushingId !== first.id) return;
        flushingId = null;
        Msg.warning('ai.chat.queueFlushRejected');
    }, FLUSH_CONFIRM_MS);
};

watch([shouldQueue, queuedMessages] as const, () => tryFlush(), { flush: 'post' });

// 活跃 turn 换了新 id = 后端确实起了新一轮：此刻才把队列项摘掉并重置重试计数
watch(
    () => store.getSlice(props.convId)?.activeTurnId ?? null,
    (turnId) => {
        if (!flushingId || !turnId || turnId === flushBaseTurnId) return;
        store.removeQueuedMessage(props.convId, flushingId);
        flushingId = null;
        window.clearTimeout(flushTimer);
        requestAnimationFrame(() => messageListRef.value?.scrollToEnd());
    }
);

// ==================== 队列项编辑/删除 ====================

/**
 * 编辑队列消息：回填输入框（含已上传附件与芯片段），回填成功后才移出队列。
 *
 * 两个细节都是必要的：只传 text 会让芯片降级成纯文本，而发送/落库靠的是段里的 extra
 * 定位信息（工具参数展开、模型目标识别）——那是信息级丢失，不是显示问题；
 * 而先移出后回填在输入框未就绪时（setValue 返回 false）会把消息从队列里抹掉又不落到
 * 输入框，等于直接丢消息，所以改成「成功才移除」
 */
const onEditQueued = (message: QueuedMessage) => {
    const filled = chatInputRef.value?.setValue(message.data.text, message.data.attachments, message.data.segments);
    if (!filled) {
        return;
    }
    store.removeQueuedMessage(props.convId, message.id);
};

// ==================== 空态建议芯片 ====================

/** 点击建议：回填输入框并聚焦（不直接发送，用户可先 @ 引用资源） */
const onSuggest = (text: string) => {
    chatInputRef.value?.setValue(text);
};

// ==================== 追问草稿预填 ====================

/** 编辑器就绪重试上限（约一帧一次）：超限就放弃，不靠无限轮询掩盖问题 */
const DRAFT_APPLY_RETRIES = 20;

/**
 * 草稿到输入框（不自动发送）：用户可以看到自己将要发出去的完整上下文再决定。
 *
 * TipTap 编辑器在子组件挂载后才有实例，而本面板的首次 post-flush 可能比它更早，
 * 因此 setValue 回返「未就绪」时按帧重试；应用成功才 emit 通知投递方清空，
 * 避免切会话时重复回填
 */
/** 草稿投递的 rAF 句柄：卸载必须取消，否则回调会在已销毁的组件上继续逐帧跑 */
let draftRafId = 0;

const applyDraft = (text: string, attempt = 0) => {
    if (chatInputRef.value?.setValue(text)) {
        emit('draft-applied');
        return;
    }
    if (attempt < DRAFT_APPLY_RETRIES) {
        draftRafId = requestAnimationFrame(() => applyDraft(text, attempt + 1));
        return;
    }
    // 重试耗尽必须留痕：此前是静默丢弃，用户点「带上下文追问」后什么都没发生，
    // 而 store 里的草稿常驻，下次进面板还会再撞一次同样的失败
    console.warn(`[ai] draft not applied after ${DRAFT_APPLY_RETRIES} frames: chat input not ready`);
};

onBeforeUnmount(() => {
    if (draftRafId) {
        cancelAnimationFrame(draftRafId);
        draftRafId = 0;
    }
    // flush 的两个定时器必须在卸载时清掉：否则回调会在已销毁组件上继续 emit('send')
    window.clearTimeout(flushTimer);
});

watch(
    () => props.draft,
    (text) => {
        if (text) applyDraft(text);
    },
    { immediate: true, flush: 'post' }
);

// ==================== 用户消息原地编辑（editingMessageId） ====================

const editingMessageId = ref<string | null>(null);

const onEditMessage = (messageId: string) => {
    editingMessageId.value = messageId;
};

/**
 * 编辑后发送：作为新一轮消息（忙时入队，handleEditSend）
 *
 * 段序列必须随文本一并重发：芯片的定位信息只存在 extra 里，把它置空等于把用户的引用
 * 抹掉——模型收到的只是一句带字面 `@名字` 的文本，完全不知道要操作哪台机器
 */
const onEditSend = (content: string, segments: ChipSegment[]) => {
    // 组装走 buildSubmitData 单点：段序列不携带时会被测试抓出，而不是默默发出去
    const data = buildSubmitData(content, segments);
    if (!data.text) return;
    editingMessageId.value = null;
    if (shouldQueue.value) {
        store.enqueueMessage(props.convId, data);
    } else {
        emit('send', data);
        requestAnimationFrame(() => messageListRef.value?.scrollToEnd());
    }
};

// ==================== 滚动锚点持久化（scrollAnchorStore） ====================

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
    /* 层级：面板不限宽，消息列 max-w-4xl、
       Composer max-w-3xl 各自限宽居中 */
}
</style>
