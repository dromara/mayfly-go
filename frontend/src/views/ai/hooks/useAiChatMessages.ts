import { formatDate } from '@/common/utils/format';
import { Msg } from '@/hooks/useI18n';
import { computed, provide, reactive, ref, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { aiApi, SessionMessage, ToolCall } from '../api';
import { InterruptActionEvent } from '../interrupt/types';

// ==================== 常量定义 ====================
const ROLE = {
    AI: 'assistant',
    USER: 'user',
    TOOL: 'tool',
    INTERNAL: 'internal',
} as const;

const MESSAGE_TYPE = {
    END: 'end',
    ERROR: 'error',
} as const;

const THINK_STATUS = {
    LOADING: 'loading',
    SUCCESS: 'success',
    ERROR: 'error',
} as const;

const THINK_TYPE = {
    REASONING: 'reasoning',
    TOOL: 'tool',
} as const;

const BUBBLE_MAX_WIDTH = '80%';
const TOOL_ERROR_PREFIX = '[tool error]';
const TOOL_RESULT_MAX_LENGTH = 1500;

/**
 * Internal 消息类型
 */
type InternalMessageType = SessionMessage & {
    id?: string; // 内部消息ID
    actionId: string;
    extra?: {
        type?: 'interrupt' | 'resume' | string; // 内部消息类型
        interruptId?: string;
        content?: { title?: string; description?: string; [key: string]: unknown };
        resumeInfo?: { action: string; payload?: Record<string, unknown>; timestamp?: string };
        [key: string]: unknown;
    };
};

type MessageType = SessionMessage & {
    key?: number;
    loading?: boolean; // 是否正在加载中
    thinks?: Array<{
        type?: string;
        status: string;
        id: string;
        title: string;
        isCanExpand: boolean;
        isDefaultExpand: boolean;
        thinkTitle: string;
        thinkContent: string;
        extra?: Record<string, unknown>;
    }>; // 思考链

    internals?: Array<InternalMessageType>; // 内部消息数组
    pendingResumes?: InterruptActionEvent[]; // 待批量提交的中断恢复信息
    unprocessedInterruptCount?: number; // 未处理的中断数量
};

export function useAiChatMessages(
    props: { sessionId: string },
    emit: (event: 'activate', sessionId: string) => void,
    currentSessionId: Ref<string>,
    sendMessage: (type: 'text' | 'interruptResume', content: string) => void
) {
    const { t } = useI18n();

    const state = reactive({
        msgLoading: false,
        senderLoading: false,
        messages: [] as Array<MessageType>,
        // 用于强制触发 messages computed 重新计算的版本号
        // 当修改消息内部状态（如 pendingResumes、internals）时递增
        version: 0,
    });

    // 标记会话是否已激活（用户是否发送过消息）
    const sessionActivated = ref(false);

    /**
     * 统一的中断操作处理器
     * 所有中断操作均缓存到 pendingResumes，当所有中断都处理完毕后自动批量提交
     */
    const handleInterruptAction = async (action: InterruptActionEvent) => {
        // 处理中断操作

        // 先尝试通过 turnId 找到 AI/INTERNAL 消息（中断只属于 AI 消息容器）
        let message = state.messages.find((m: MessageType) => (m.role === ROLE.AI || m.role === ROLE.INTERNAL) && m.turnId && m.turnId === action.turnId);

        // 如果找不到，尝试通过 interruptId 找到包含该中断的 AI/INTERNAL 消息
        if (!message) {
            message = state.messages.find(
                (m: MessageType) =>
                    (m.role === ROLE.AI || m.role === ROLE.INTERNAL) &&
                    m.internals?.some((internal: InternalMessageType) => (internal.actionId || internal.extra?.actionId) === action.interruptId)
            );
        }

        // 最后回退到最后一条 AI/INTERNAL 消息
        if (!message) {
            message = [...state.messages].reverse().find((m: MessageType) => m.role === ROLE.AI || m.role === ROLE.INTERNAL);
        }

        if (!message) {
            console.warn('没有消息可处理中断');
            return;
        }

        // 初始化 pendingResumes
        if (!message.pendingResumes) {
            message.pendingResumes = [];
        }

        // 去重：同一 interruptId 只保留最新的
        const existingIndex = message.pendingResumes.findIndex((r: InterruptActionEvent) => r.interruptId === action.interruptId);
        if (existingIndex >= 0) {
            message.pendingResumes[existingIndex] = action;
        } else {
            message.pendingResumes.push(action);
        }

        // 更新对应 interrupt 的 pending 状态，用于 UI 显示
        const targetInterrupt = message.internals?.find(
            (internal: InternalMessageType) =>
                (internal.actionId || internal.extra?.actionId) === action.interruptId && internal.extra?.type?.startsWith('interrupt_')
        );
        if (targetInterrupt) {
            if (!targetInterrupt.extra) {
                targetInterrupt.extra = {};
            }
            targetInterrupt.extra.pendingResumeInfo = {
                action: action.action,
                payload: action.payload,
            };
        }

        // 递增版本号，强制触发 messages computed 重新计算
        state.version++;
        // 强制替换 messages 数组引用，确保 BubbleList 检测到变化
        state.messages = [...state.messages];

        // 检查是否所有中断都已处理，如果是则自动批量提交
        const unprocessedCount = (message.internals || []).filter(
            (internal: InternalMessageType) => internal.extra?.type?.startsWith('interrupt_') && !internal.extra?.resumeInfo
        ).length;

        if (message.pendingResumes && message.pendingResumes.length === unprocessedCount && unprocessedCount > 0) {
            // 所有中断已处理完毕，自动提交（setTimeout 让当前渲染周期完成）
            setTimeout(() => {
                submitPendingResumes(message.turnId || '');
            }, 0);
        }
    };

    // 提供中断操作处理器给子组件，绕过 Vue 动态组件事件传递问题
    provide('handleInterruptAction', handleInterruptAction);

    /**
     * 提交所有待处理的中断恢复信息（批量提交）
     * @param turnId 轮次ID
     */
    const submitPendingResumes = (turnId: string) => {
        let message = state.messages.find((m: MessageType) => (m.role === ROLE.AI || m.role === ROLE.INTERNAL) && m.turnId && m.turnId === turnId);
        if (!message) {
            message = [...state.messages].reverse().find((m: MessageType) => m.role === ROLE.AI || m.role === ROLE.INTERNAL);
        }
        if (!message || !message.pendingResumes || message.pendingResumes.length === 0) {
            return;
        }

        // 发送批量恢复请求
        sendMessage('interruptResume', JSON.stringify(message.pendingResumes));

        // 将 pending 状态转为正式 resumeInfo，并清空 pending
        for (const resume of message.pendingResumes) {
            const targetInterrupt = message.internals?.find(
                (internal: InternalMessageType) =>
                    (internal.actionId || internal.extra?.actionId) === resume.interruptId && internal.extra?.type?.startsWith('interrupt_')
            );
            if (targetInterrupt) {
                if (!targetInterrupt.extra) {
                    targetInterrupt.extra = {};
                }
                targetInterrupt.extra.resumeInfo = {
                    action: resume.action,
                    payload: resume.payload,
                };
                delete targetInterrupt.extra.pendingResumeInfo;
            }
        }

        message.pendingResumes = [];

        // 递增版本号，强制触发 messages computed 重新计算
        state.version++;
        // 强制替换 messages 数组引用，确保 BubbleList 检测到变化
        state.messages = [...state.messages];
    };

    /**
     * 原子操作：追加工具调用到消息的思考链中
     */
    const appendToolCall = (message: MessageType, toolCall: ToolCall) => {
        if (!message.thinks) {
            message.thinks = [];
        }

        // 检查是否已存在相同的 toolCall
        const existingThink = message.thinks.find((t) => t.type === THINK_TYPE.TOOL && t.extra?.toolCallId === toolCall.id);

        if (existingThink) {
            // 如果已存在，更新内容
            existingThink.thinkContent = JSON.stringify(toolCall);
        } else {
            // 在添加工具调用之前，将所有 loading 状态的 reasoning 思考标记为 success
            for (const think of message.thinks) {
                if (think.type === THINK_TYPE.REASONING && think.status === THINK_STATUS.LOADING) {
                    think.status = THINK_STATUS.SUCCESS;
                }
            }

            // 否则创建新的 think
            message.thinks.push({
                type: THINK_TYPE.TOOL,
                status: THINK_STATUS.LOADING,
                id: String(toolCall.id),
                title: t('ai.chat.toolCall'),
                isCanExpand: true,
                isDefaultExpand: false,
                thinkTitle: t('ai.chat.toolCall') + ' - ' + toolCall.function?.name || '',
                thinkContent: JSON.stringify(toolCall),
                extra: {
                    toolCallId: toolCall.id,
                    toolName: toolCall.function?.name,
                },
            });
        }
    };

    /**
     * 原子操作：追加工具调用结果到消息的思考链中
     */
    const appendToolResult = (message: MessageType, toolCallId: string, content?: string) => {
        if (!message.thinks || !content) {
            return;
        }

        for (let think of message.thinks) {
            if (think.type !== THINK_TYPE.TOOL || !think.extra) {
                continue;
            }
            if (think.extra.toolCallId === toolCallId) {
                const displayContent = content.length > TOOL_RESULT_MAX_LENGTH ? content.substring(0, TOOL_RESULT_MAX_LENGTH) + '...' : content;
                think.thinkContent = content ? `${think.thinkContent}  ${t('ai.chat.toolCallResult')}: ${displayContent}` : displayContent;
                if (content.startsWith(TOOL_ERROR_PREFIX)) {
                    think.status = THINK_STATUS.ERROR;
                } else {
                    think.status = THINK_STATUS.SUCCESS;
                }
                return;
            }
        }
    };

    /**
     * 原子操作：追加内部消息到消息中
     */
    const appendInternal = (message: MessageType, internalMsg: SessionMessage) => {
        if (internalMsg.type == MESSAGE_TYPE.ERROR) {
            Msg.error(internalMsg.content);
            // 内部错误也要取消 loading 状态
            state.senderLoading = false;
            return;
        }
        // 处理会话结束或错误
        if (internalMsg.type == MESSAGE_TYPE.END) {
            handleTurnEnd(message, internalMsg);
            return;
        }

        // 处理 resume 类型的内部消息：合并到对应的 interrupt 中
        if (internalMsg.extra?.type === 'resume') {
            mergeResumeToInterrupt(message, internalMsg);
            return;
        }

        if (!message.internals) {
            message.internals = [];
        }
        message.internals.push(internalMsg as InternalMessageType);
    };

    /**
     * 将 resume 消息合并到对应的 interrupt 消息中
     */
    const mergeResumeToInterrupt = (message: MessageType, resumeMsg: SessionMessage) => {
        if (!message.internals || message.internals.length === 0) {
            console.warn('No internals found to merge resume message');
            return;
        }

        const resumeData = resumeMsg.extra?.content as any;
        const interruptId = resumeData?.interruptId;

        if (!interruptId) {
            console.warn('Resume message missing interruptId');
            return;
        }

        // 查找对应的 interrupt 消息
        const targetInterrupt = message.internals.find(
            (internal) => (internal.actionId || internal.extra?.actionId) === interruptId && internal.extra?.type?.startsWith('interrupt_')
        );

        if (!targetInterrupt) {
            console.warn(`Interrupt with id ${interruptId} not found`);
            return;
        }

        // 更新 interrupt 的状态和内容
        if (!targetInterrupt.extra) {
            targetInterrupt.extra = {};
        }

        // 记录操作信息
        targetInterrupt.extra.resumeInfo = {
            action: resumeData.action,
            timestamp: resumeMsg.time,
            payload: resumeData.payload,
        };
    };

    /**
     * 处理会话结束或错误
     */
    const handleTurnEnd = (message: MessageType, chunkMsg: SessionMessage) => {
        message.time = chunkMsg.time;

        // 会话结束，重置 loading 状态
        state.senderLoading = false;

        // 结束可能存在的思考链
        if (message.thinks && message.thinks.length > 0) {
            for (let think of message.thinks) {
                if (think.status == THINK_STATUS.LOADING && think.type == THINK_TYPE.REASONING) {
                    think.status = THINK_STATUS.SUCCESS;
                }
            }
        }

        // 如果是新会话，触发会话激活事件
        const isNewSession = props.sessionId === '-1';
        if (isNewSession && !sessionActivated.value) {
            sessionActivated.value = true;
            setTimeout(() => {
                emit('activate', currentSessionId.value);
            }, 500);
        }
    };

    /**
     * 原子操作：追加推理内容到消息的思考链中
     */
    const appendReasoning = (message: MessageType, reasoningContent: string) => {
        if (!reasoningContent) {
            return;
        }
        const title = t('ai.chat.thinking');

        const thinks = message.thinks;

        // 创建think对象的辅助函数
        const createThink = (status: string, id: number | string) => ({
            type: THINK_TYPE.REASONING,
            status,
            id: String(id),
            title: title,
            isCanExpand: true,
            isDefaultExpand: false,
            thinkTitle: title,
            thinkContent: reasoningContent,
        });

        // 如果没有thinks数组，初始化
        if (!thinks || thinks.length == 0) {
            message.thinks = [createThink(THINK_STATUS.LOADING, 1)];
            return;
        }

        const thinkIndex = thinks.length - 1;
        const think = thinks[thinkIndex];

        // 如果title不同，结束当前think并创建新的
        if (think.title != title) {
            // 将上一个 thinking 标记为 success
            if (think.status === THINK_STATUS.LOADING) {
                think.status = THINK_STATUS.SUCCESS;
            }
            thinks.push(createThink(THINK_STATUS.LOADING, thinkIndex + 2));
            return;
        }

        // 如果当前think还在loading状态，追加内容
        if (think.status == THINK_STATUS.LOADING) {
            think.thinkContent += reasoningContent;
            if (!reasoningContent) {
                think.status = THINK_STATUS.SUCCESS;
            }
            return;
        }

        // 否则创建新的think
        thinks.push(createThink(THINK_STATUS.LOADING, thinkIndex + 2));
    };

    /**
     * 原子操作：追加普通文本内容到消息
     */
    const appendContent = (message: MessageType, content: string) => {
        if (!content) {
            return;
        }
        message.content += content;

        if (message.loading) {
            message.loading = false;
        }
    };

    /**
     * 统一的消息处理器
     * 无论是 WebSocket chunk 还是历史消息，都通过此函数处理
     */
    const processMessage = (message: SessionMessage, targetMessage: MessageType) => {
        switch (message.role) {
            case ROLE.USER:
                // 用户消息：直接追加内容
                appendContent(targetMessage, message.content);
                break;

            case ROLE.AI:
                // AI 消息：处理内容、推理、工具调用
                const isToolCall = message.toolCalls && message.toolCalls.length > 0;

                if (!isToolCall) {
                    appendContent(targetMessage, message.content);
                } else {
                    // 如果是工具调用，并且存在内容，则添加为推理内容
                    if (message.content) {
                        message.reasoningContent = message.content;
                    }
                }

                // 处理推理内容
                if (message.reasoningContent) {
                    appendReasoning(targetMessage, message.reasoningContent);
                }

                // 处理工具调用
                if (isToolCall && message.toolCalls) {
                    for (let toolCall of message.toolCalls) {
                        appendToolCall(targetMessage, toolCall);
                    }
                }
                break;

            case ROLE.TOOL:
                // 工具结果：更新对应的工具调用状态
                if (message.toolCallId) {
                    appendToolResult(targetMessage, message.toolCallId, message.content || '');
                }
                break;

            case ROLE.INTERNAL:
                // 内部消息：添加到 internals 数组
                appendInternal(targetMessage, message);
                break;

            default:
                console.warn('Unknown message role:', message.role);
        }
    };

    /**
     * 处理 WebSocket 消息块
     */
    const handleChunk = (chunkMsg: SessionMessage) => {
        // 处理系统错误消息：使用 ElMessage.error 提示，不添加到聊天列表
        if (chunkMsg.type === 'error') {
            Msg.error(chunkMsg.content);
            return;
        }

        const nowMsgIndex = state.messages.length - 1;
        const message = state.messages[nowMsgIndex];
        if (!message) {
            console.warn('No message to append chunk to: ', nowMsgIndex);
            return;
        }

        // 同步 turnId：后端返回的 chunk 消息携带 turnId，需要赋值给当前消息容器
        // 使用 != null 以支持空字符串 turnId（后端可能发送空字符串）
        if (chunkMsg.turnId != null && !message.turnId) {
            message.turnId = chunkMsg.turnId;
        }

        // 使用统一的消息处理器
        processMessage(chunkMsg, message);
    };

    /**
     * 加载历史消息
     */
    const loadMessage = async () => {
        if (!props.sessionId || props.sessionId === '-1') {
            return;
        }
        try {
            state.msgLoading = true;
            const messages = await aiApi.listMessages.request({ sessionKey: props.sessionId });
            state.messages = converterMessages(messages);

            // 检查最后一条 AI 消息是否还在 loading 状态，如果是则保持 senderLoading 为 true
            const lastMessage = state.messages[state.messages.length - 1];
            if (lastMessage && lastMessage.role === ROLE.AI && lastMessage.loading) {
                state.senderLoading = true;
            }
        } finally {
            state.msgLoading = false;
        }
    };

    /**
     * 转换消息格式
     */
    const converterMessages = (messages: SessionMessage[]) => {
        // 按 turnId 分组消息
        const turnGroups = new Map<
            string,
            {
                userMessage?: MessageType;
                aiMessage?: MessageType;
                toolCalls: SessionMessage[];
                toolResults: Map<string, SessionMessage>; // toolCallId -> toolResult
                internals: SessionMessage[];
            }
        >();

        // 第一轮遍历：按 turnId 分类所有消息
        for (let message of messages) {
            if (!message.turnId) {
                console.warn('Message without turnId skipped:', message);
                continue;
            }

            if (!turnGroups.has(message.turnId)) {
                turnGroups.set(message.turnId, {
                    userMessage: undefined,
                    aiMessage: undefined,
                    toolCalls: [],
                    toolResults: new Map(),
                    internals: [],
                });
            }

            const group = turnGroups.get(message.turnId)!;

            // 用户消息
            if (message.role === ROLE.USER) {
                group.userMessage = message;
                continue;
            }

            // 工具调用消息
            if (message.toolCalls && message.toolCalls.length > 0) {
                group.toolCalls.push(message);
                continue;
            }

            // 工具调用结果
            if (message.role === ROLE.TOOL && message.toolCallId) {
                group.toolResults.set(message.toolCallId, message);
                continue;
            }

            // AI 助手消息
            if (message.role === ROLE.AI) {
                group.aiMessage = message;
                continue;
            }

            // 内部消息（中断等）
            if (message.role === ROLE.INTERNAL) {
                group.internals.push(message);
                continue;
            }
        }

        // 第二轮遍历：构建最终消息列表，使用统一的消息处理逻辑
        const finalMessages: MessageType[] = [];

        for (let [turnId, group] of turnGroups) {
            // 用户消息直接添加
            if (group.userMessage) {
                finalMessages.push({
                    ...group.userMessage,
                    time: formatDate(group.userMessage.time),
                });
            }

            // 创建主消息容器
            let mainMessage: MessageType;

            if (group.aiMessage) {
                mainMessage = {
                    ...group.aiMessage,
                    loading: false,
                    thinks: [],
                    internals: [],
                };
            } else {
                // 如果没有 AI 消息，但有其他内容，创建一个 loading 状态的 AI 消息容器
                if (group.internals.length > 0 || group.toolCalls.length > 0 || group.toolResults.size > 0) {
                    mainMessage = {
                        role: ROLE.AI,
                        content: '',
                        loading: true,
                        thinks: [],
                        internals: [],
                    };
                } else {
                    // 没有任何内容，跳过
                    continue;
                }
            }

            // 处理工具调用消息
            for (let toolCallMsg of group.toolCalls) {
                processMessage(toolCallMsg, mainMessage);
            }

            // 处理工具调用结果
            for (let [, toolResultMsg] of group.toolResults) {
                processMessage(toolResultMsg, mainMessage);
            }

            // 处理内部消息
            for (let internalMsg of group.internals) {
                processMessage(internalMsg, mainMessage);
            }

            finalMessages.push(mainMessage);
        }

        return finalMessages;
    };

    /**
     * 转为组件需要的数据
     */
    // BubbleList 强制重新渲染的 key，基于 version 变化
    const bubbleListKey = computed(() => `bubble-list-${state.version}`);

    const messages = computed(() => {
        return state.messages.map((item: MessageType) => {
            const role = item.role;
            const unprocessedInterruptCount = (item.internals || []).filter(
                (internal: InternalMessageType) => internal.extra?.type?.startsWith('interrupt_') && !internal.extra?.resumeInfo
            ).length;
            const pendingCount = item.pendingResumes?.length || 0;
            // key 需包含 pendingCount 和 unprocessedInterruptCount，确保 BubbleList 在中断状态变化时重新渲染
            // param-completion 组件内部通过 watch 监听 pendingResumeInfo，可在重新挂载后恢复用户输入状态
            const key = item.key || `${item.turnId || 'no-turn'}-${pendingCount}-${unprocessedInterruptCount}`;
            return {
                key,
                role: item.role,
                content: item.content,
                time: formatDate(item.time),
                placement: role === ROLE.AI || role == ROLE.INTERNAL ? 'start' : 'end',
                variant: role === ROLE.AI ? 'filled' : 'outlined', // 气泡的样式
                isFog: role === ROLE.AI, // AI 消息开启雾化效果
                maxWidth: BUBBLE_MAX_WIDTH,
                thinks: item.thinks,
                internals: JSON.parse(JSON.stringify(item.internals || [])) as InternalMessageType[], // 深拷贝一份 internals，避免响应式问题
                loading: item.loading,
                extra: item.extra,
                turnId: item.turnId,
                reasoningContent: item.reasoningContent,
                pendingResumes: item.pendingResumes,
                unprocessedInterruptCount,
            };
        }) as MessageType[];
    });

    /**
     * 重置消息
     */
    const resetMessages = () => {
        state.messages = [];
        state.version = 0;
        sessionActivated.value = false;
    };

    /**
     * 添加用户消息
     */
    const addUserMessage = (content: string) => {
        state.messages.push({
            content: content,
            role: ROLE.USER,
            time: new Date().toISOString(),
        });

        state.messages.push({
            content: '',
            role: ROLE.AI,
            loading: true,
        });
    };

    return {
        state,
        messages,
        bubbleListKey,
        handleInterruptAction,
        handleChunk,
        loadMessage,
        resetMessages,
        addUserMessage,
    };
}
