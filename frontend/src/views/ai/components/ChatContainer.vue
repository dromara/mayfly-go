<template>
    <div class="chat-container">
        <ChatPanel
            :messages="messages"
            :loading="msgLoading"
            :sender-loading="senderLoading"
            :interrupts="interrupts"
            :has-more="hasMore"
            :conv-id="convId"
            :skills="skills"
            @send="onSend"
            @load-more="onLoadMore"
            @cancel="onStop"
            @interrupt-action="onInterruptAction"
        />
    </div>
</template>

<script setup lang="ts" name="ChatContainer">
/**
 * ChatContainer - 编排层
 * 对齐 tokhub 的 ChatContainer 编排模式
 *
 * 职责：
 * - 管理 WebSocket 连接生命周期
 * - 编排 useChatStream + useChatCallbacks + useChatMessages
 * - 将数据传递给 ChatPanel（纯 UI 层）
 */
import { computed, onMounted, ref, watch } from 'vue';
import { useChatStore } from '../stores/chatStore';
import { useChatStream } from '../stream/useChatStream';
import { useChatCallbacks } from '../stream/useChatCallbacks';
import { useChatMessages } from '../hooks/useChatMessages';
import { useChatResume } from '../stream/useChatResume';
import type { Conversation, ResumeEntry } from '../protocol/types';
import type { ChatInputSubmitData, SkillItem } from '../input/types';
import { buildMessageContent } from '../input/attachments';
import { aiApi } from '../api';
import type { InterruptActionEvent } from '../interrupt/types';
import ChatPanel from './ChatPanel.vue';

const props = defineProps<{
    conversation: Conversation;
}>();

const emit = defineEmits<{
    (e: 'conversation-created', id: number): void;
    (e: 'interrupt-action', event: InterruptActionEvent): void;
}>();

const store = useChatStore();

const convId = computed(() => props.conversation.id);

// 从 Store 获取当前会话的消息和中断
const messages = computed(() => {
    const slice = store.getSlice(convId.value);
    return slice?.messages || [];
});

const interrupts = computed(() => {
    const slice = store.getSlice(convId.value);
    return slice?.pendingInterrupts || [];
});

const hasMore = computed(() => {
    const slice = store.getSlice(convId.value);
    return slice?.hasMoreMessages || false;
});

const msgLoading = ref(false);
const senderLoading = ref(false);

// 技能引用数据源（'/' 触发菜单，一次性加载；资源引用由 @ 触发的资源树面板懒加载）
const skills = ref<SkillItem[]>([]);

onMounted(async () => {
    try {
        // Api 实例需调用 .request() 发起请求（直接调用实例会 TypeError 且被 catch 静默吞掉）
        skills.value = (await aiApi.listSkills.request()) || [];
    } catch (e) {
        // 加载失败时静默降级：触发菜单显示空列表，不影响聊天主流程
        console.error('[ai] load skills failed:', e);
    }
});

// 初始化 composables（convId 为 computed，内部自动跟踪变化）
const callbacks = useChatCallbacks(convId);
const { initSocket, sendMessage, sendStop, sendAttach } = useChatStream();
const { loadMessages, loadMoreMessages, addUserMessage, reloadMessages } = useChatMessages(convId);

// 中断恢复编排：决策池 + 批量恢复 + 失败回滚（对齐 tokhub useChatResume）
const sendResume = async (entries: ResumeEntry[]): Promise<boolean> => {
    // 确保 WebSocket 已连接（刷新页面后中断恢复时需要）
    await ensureSocket();
    return sendMessage({
        conversationId: props.conversation.id,
        type: 'interruptResume',
        content: JSON.stringify(entries),
    });
};
const { recordDecision } = useChatResume(convId, { send: sendResume });

// 构建 StreamCallbacks
const streamCallbacks = computed(() => ({
    ...callbacks,
    onEnd() {
        callbacks.onEnd();
        senderLoading.value = false;
    },
    onError(error: string) {
        callbacks.onError(error);
        senderLoading.value = false;
    },
    onConversationCreated(id: number) {
        callbacks.onConversationCreated(id);
        emit('conversation-created', id);
    },
    // attach 成功：回放完毕，恢复生成中 UI（停止按钮/输入框禁用），后续实时事件续上
    onTurnAttached(turnId: string, conversationId?: number) {
        callbacks.onTurnAttached?.(turnId, conversationId);
        senderLoading.value = true;
        didResetForAttach = false;
    },
    onTurnNotRunning() {
        // 无运行中 turn：确保生成中 UI 不残留（如切换会话后的边缘场景）
        callbacks.onTurnNotRunning?.();
        senderLoading.value = false;
        // 若本次 attach 前重置过流式中间态而目标 turn 已结束，
        // 被删的流式消息需从服务端落库终态回填，否则整轮回复丢失
        if (didResetForAttach) {
            didResetForAttach = false;
            void reloadMessages();
        }
    },
    // 连接就绪（重连成功）：该会话存在流式中的 turn 时重新 attach 续流
    onSocketOpen() {
        if (store.getSlice(convId.value)?.activeTurnId) {
            attachRunningTurn();
        }
    },
}));

/**
 * attach 运行中 turn 续流：先重置该会话流式中间态再订阅，
 * 服务端回放缓冲快照全量重建（重复 attach 不叠加旧 parts）。
 * 若确实删除了流式消息而 attach 后发现 turn 已结束，
 * onTurnNotRunning 会触发 reloadMessages 从落库终态回填
 */
let didResetForAttach = false;
const attachRunningTurn = () => {
    didResetForAttach = store.resetConvStreaming(props.conversation.id);
    void sendAttach(props.conversation.id);
};

// 初始化 WebSocket（首次发送时连接）
let socketPromise: Promise<void> | null = null;
const ensureSocket = (): Promise<void> => {
    if (!socketPromise) {
        socketPromise = initSocket(streamCallbacks.value).then(() => {});
    }
    return socketPromise;
};

// 加载消息
watch(
    () => props.conversation.id,
    async (newId) => {
        if (!newId) return;
        msgLoading.value = true;
        try {
            await loadMessages();
            // 续流：服务端该会话存在运行中 turn 时 attach 回放续上
            // （刷新页面/断线重连场景；turn 已结束则回 turn_not_running 静默忽略）。
            // attach 内部静默等待连接就绪，不 await 以免拖住会话切换的 loading
            await ensureSocket();
            attachRunningTurn();
        } finally {
            msgLoading.value = false;
        }
    },
    { immediate: true },
);

const onSend = async (data: ChatInputSubmitData) => {
    await ensureSocket();
    // 回显：结构化 segments 透传（芯片渲染为 InlineChip，对齐 tokhub ContentSegment 贯穿），
    // 纯文本 content 作回退/编辑用；发送给 LLM 的完整注入文本由后端 buildChatContent 生成
    addUserMessage(data.text, data.attachments, data.segments);
    sendMessage({
        conversationId: props.conversation.id,
        type: 'text',
        // 后端零改动：文本类附件内联进 content，图片以说明占位（展示由消息附件渲染承担）
        content: buildMessageContent(data.text, data.attachments),
        segments: data.segments,
    });
    senderLoading.value = true;
    autoRenameFromFirstMessage(data.text);
};

// ==================== 停止生成 ====================

/**
 * 用户主动停止：发送 stop 协议消息（服务端取消 turn ctx，真正中断 agent 的
 * LLM 请求与工具执行）。不断开连接；终态与提示由 turn_completed(status=stopped)
 * 事件驱动（callbacks.onTurnCompleted 统一处理），服务端无运行中 turn 时回错误事件
 */
const onStop = () => {
    if (!senderLoading.value) return;
    void sendStop(props.conversation.id);
};

// ==================== 会话自动命名 ====================

/** 默认标题（对齐后端 conversation.go 空标题兜底值，命中时不覆盖用户手动命名） */
const DEFAULT_TITLE = 'New Chat';

/** 首条消息自动命名：取首行前 30 字符（对齐 ChatGPT/Claude 自动命名行为） */
const autoRenameFromFirstMessage = (text: string) => {
    const title = (text.trim().split('\n')[0] || '').slice(0, 30);
    if (!title) return;
    if (props.conversation.title && props.conversation.title !== DEFAULT_TITLE) return;
    // 乐观同步 store（侧栏列表与当前会话均从 store.conversations 读取，避免直接变更 props）
    const conv = store.conversations.find((c) => c.id === props.conversation.id);
    const prevTitle = conv?.title ?? '';
    if (conv) conv.title = title;
    aiApi.renameConversation
        .request({ id: props.conversation.id, title })
        .catch(() => {
            // 命名失败回滚，侧栏保留默认标题
            if (conv) conv.title = prevTitle;
        });
};

const onLoadMore = async () => {
    await loadMoreMessages();
};

// 中断决策：委托 useChatResume（决策池 → 双写 → 同 turn 全部决策后批量恢复）
const onInterruptAction = (event: InterruptActionEvent) => {
    recordDecision(event);
};
</script>

<style scoped>
.chat-container {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
}
</style>
