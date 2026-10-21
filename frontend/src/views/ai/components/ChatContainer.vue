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
            :draft="draft"
            @send="onSend"
            @load-more="onLoadMore"
            @cancel="onStop"
            @interrupt-action="onInterruptAction"
            @draft-applied="$emit('draft-applied')"
        />
    </div>
</template>

<script setup lang="ts" name="ChatContainer">
/**
 * ChatContainer - 编排层
 * ChatContainer 编排模式
 *
 * 职责：
 * - 管理 WebSocket 连接生命周期
 * - 编排 useChatStream + useChatCallbacks + useChatMessages
 * - 将数据传递给 ChatPanel（纯 UI 层）
 */
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';
import { useChatStore } from '../stores/chatStore';
import { useChatStream } from '../stream/useChatStream';
import { useChatCallbacks } from '../stream/useChatCallbacks';
import { useChatMessages } from '../hooks/useChatMessages';
import { useConvSlice } from '../hooks/useConvSlice';
import { usePromptSkills } from '../input/usePromptSkills';
import { useChatResume } from '../stream/useChatResume';
import type { Conversation, ResumeEntry } from '../protocol/types';
import type { ChatInputSubmitData, SkillItem } from '../input/types';
import { buildImageSegments, buildMessageContent } from '../input/attachments';
import { aiApi } from '../api';
import type { InterruptActionEvent } from '../interrupt/types';
import ChatPanel from './ChatPanel.vue';

const props = defineProps<{
    conversation: Conversation;
    /** 追问草稿：原样交给 ChatPanel 预填（本层不消费、不解释它） */
    draft?: string;
}>();

const emit = defineEmits<{
    (e: 'conversation-created', id: number): void;
    (e: 'draft-applied'): void;
}>();

const store = useChatStore();
const { t } = useI18n();

const convId = computed(() => props.conversation.id);

// 会话切片的只读视图（与运维轨迹 RunTranscript 共用同一读取层）
const { messages, hasMore, pendingInterrupts: interrupts } = useConvSlice(convId);

const msgLoading = ref(false);
const senderLoading = ref(false);

// 技能引用数据源（'/' 触发菜单）：获取与失败降级收敛在 usePromptSkills 单点，
// 表单里的提示词编辑器（PromptEditor）用同一份缓存；资源引用由 @ 触发的资源树面板懒加载
const { skills, load: loadSkills } = usePromptSkills();

onMounted(() => {
    void loadSkills();
});

// 初始化 composables（convId 为 computed，内部自动跟踪变化）
const callbacks = useChatCallbacks(convId);
const { initSocket, sendMessage, sendStop, sendAttach } = useChatStream();
const { loadMessages, loadMoreMessages, addUserMessage, reloadMessages } = useChatMessages(convId);

// 中断恢复编排：决策池 + 批量恢复 + 失败回滚（useChatResume）
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
/**
 * 本轮是否已被后端接受（收到过 onTurnStart）与乐观气泡 id。
 *
 * 为什么要记：sendMessage 返回 true 只说明字节写进了 socket，不代表后端接了这轮——
 * 上一轮还在收尾时后端会直接拒收（`conversation has a running turn`）。那时乐观气泡已经渲染、
 * 却永远不会有回复，界面留着一条骗人的消息（实测：被拒的 flush 在流里留了个永久空泡）。
 * 只有「本轮从未被接受」才回滚，中途失败保留气泡（内容已落库，回滚反而抹掉真实记录）。
 */
let turnAccepted = false;
let lastOptimisticId: string | null = null;

const streamCallbacks = computed(() => ({
    ...callbacks,
    // 后端起了新一轮 = 这轮真的被接受了（区分「未送达」与「中途失败」的唯一可靠信号）
    onTurnStart(turnId: string, conversationId?: number) {
        turnAccepted = true;
        lastOptimisticId = null;
        callbacks.onTurnStart(turnId, conversationId);
    },
    onEnd() {
        callbacks.onEnd();
        senderLoading.value = false;
    },
    onError(error: string) {
        callbacks.onError(error);
        senderLoading.value = false;
        // 本轮根本没进模型（典型为后端仍有 running turn 而拒收）：撤掉乐观气泡。
        // 队列场景下该项仍在待发队列里（flush 闸门不删），用户可从队列重新发出；
        // 不撤的话，流里会多出一条永不被回答的气泡，与库里实际内容不一致
        if (!turnAccepted && lastOptimisticId !== null) {
            store.removeMessage(props.conversation.id, lastOptimisticId);
            lastOptimisticId = null;
            Msg.warning('ai.chat.messageNotDelivered');
        }
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
    { immediate: true }
);

const onSend = async (data: ChatInputSubmitData) => {
    await ensureSocket();
    // 附件已在 ChatInput 提交时上传（入队/直接发送共享同一前置链路，
    // 队列项均已带 fileKey），此处不再处理上传
    const attachments = data.attachments;
    // 回显：结构化 segments 透传（芯片渲染为 InlineChip，ContentSegment 贯穿），
    // 纯文本 content 作回退/编辑用；发送给 LLM 的完整注入文本由后端 buildChatContent 生成；
    // 图片段合并后乐观回显与 WS 持久化用同一份 segments（图片卡片仅从 image 段构建，
    // 乐观消息缺段则实时不渲染、刷新后才出现——同一组装出处避免双路径不一致）
    const segments = [...(data.segments ?? []), ...buildImageSegments(attachments, t('ai.attach.imagePlaceholder'))];
    turnAccepted = false;
    const optimisticId = addUserMessage(data.text, attachments, segments);
    lastOptimisticId = optimisticId;
    const delivered = sendMessage({
        conversationId: props.conversation.id,
        type: 'text',
        // 文本类附件内联进 content；图片以 image 段贯穿（fileKey 由后端解析为
        // 多模态输入，展示侧经 segments 回显渲染），不再以文本占位重复描述
        content: buildMessageContent(data.text, attachments, {
            // 附件标签走 i18n：这段正文会落库并进 LLM 上下文，不能写死中文
            textFile: (name) => t('ai.attach.textFileBlock', { name }),
            otherFile: (name) => t('ai.attach.fileBlock', { name }),
        }),
        segments,
        // 附件元数据（fileKey 轻量引用）随消息持久化（历史回显卡片/预览，
        // 对齐后端 form.ChatAttachment）；剥离 dataUrl/text/rawFile 本地态字段
        attachments: attachments?.map(({ name, kind, mime, size, fileKey }) => ({
            name,
            kind,
            ...(mime !== undefined ? { mime } : {}),
            ...(size !== undefined ? { size } : {}),
            ...(fileKey !== undefined ? { fileKey } : {}),
        })),
    });
    // 投递失败（socket 未就绪）：回滚乐观气泡，并且不进入「生成中」。否则永远等不到
    // turn 终态事件，输入框会始终锁在 loading，后续消息还全被当成「在生成」而进队列
    // （sendResume / sendStop 都已检查过返回值，这里是漏网的一处）
    if (!delivered) {
        store.removeMessage(props.conversation.id, optimisticId);
        return;
    }
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

/** 默认标题哨兵（对齐后端 headless.DefaultConversationTitle，命中时不覆盖用户手动命名） */
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
    aiApi.renameConversation.request({ id: props.conversation.id, title }).catch(() => {
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
