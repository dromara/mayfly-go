/**
 * AI Chat 协议类型定义
 * 对齐后端 protocol/ 包和 tokhub 的 TurnItem 统一协议
 */

// ==================== ContentSegment ====================

export type ContentSegmentType = 'input_text' | 'output_text' | 'skill' | 'resource';

export interface ContentSegment {
    type: ContentSegmentType;
    text: string;
    /** 芯片段元数据（对齐 tokhub typed segment 贯穿持久化：resource 含 resourceType/id/code/ip/port/authCertName/username） */
    extra?: Record<string, unknown>;
}

// ==================== TurnItem (tagged union) ====================

export type TurnItemTypeType = 'message' | 'reasoning' | 'tool_call' | 'context_compaction';

export type TurnItemStatus = 'pending' | 'success' | 'failed' | 'cancelled' | 'interrupted';

/**
 * TurnItem 统一协议（扁平变体，对齐后端 protocol.TurnItem / tokhub item.rs 的 tagged enum：
 * 变体字段直接位于顶层、snake_case 命名，实时流与历史回放共享同一协议形状）
 */
export interface TurnItem {
    type: TurnItemTypeType;
    id: string;

    // message 变体
    role?: string;
    content?: ContentSegment[];

    // reasoning 变体
    text?: string;

    // tool_call 变体
    tool_call_id?: string;
    tool_name?: string;
    arguments?: string;
    status?: TurnItemStatus;
    output?: string;
    duration_ms?: number;

    // context_compaction 变体
    original_tokens?: number;
    compressed_tokens?: number;
    compressed_message_count?: number;
}

// ==================== EventMsg (WebSocket 事件协议) ====================

export type EventType =
    | 'item_started'
    | 'item_updated'
    | 'item_completed'
    | 'turn_started'
    | 'turn_completed'
    | 'interrupted'
    | 'error'
    | 'end'
    | 'heartbeat'
    | 'conversation_created'
    // turn 运行时订阅协议（turn 与 WS 连接解耦：刷新/断线重连后 attach 续流）
    | 'turn_attached'
    | 'turn_not_running';

export interface ItemDelta {
    text?: string;
    arguments?: string;
}

export interface TurnUsage {
    inputTokens: number;
    outputTokens: number;
    totalTokens: number;
}

export interface InterruptEvent {
    actionId: string;
    type: string;
    description: string;
    toolName?: string;
    toolCallId?: string;
    /** 中断所属 turn（用于 resume 时传递给后端） */
    turnId?: string;
    /** 恢复决策状态（取值见 registries/statuses.ts InterruptResumeStatus，缺省视为 pending） */
    status?: string;
    metadata?: Record<string, unknown>;
}

export interface EventMsg {
    type: EventType;
    turnId?: string;
    conversationId?: number;
    item?: TurnItem;
    itemId?: string;
    delta?: ItemDelta;
    status?: string;
    usage?: TurnUsage;
    interrupt?: InterruptEvent;
    error?: string;
    errSource?: string;
}

// ==================== Conversation ====================

export interface Conversation {
    id: number;
    code: string;
    title: string;
    status: number;
    totalMessageCount: number;
    totalTokens: number;
    createTime: string;
    updateTime: string;
}

/** 运行中 turn（会话列表执行中指示器数据源） */
export interface RunningTurnVO {
    conversationId: number;
    turnId: string;
}

// ==================== TurnGroup / TurnItem VO ====================

export interface TurnGroupVO {
    turnId: string;
    items: TurnItemVO[];
}

export interface TurnItemVO {
    id: number;
    turnId: string;
    itemType: string;
    itemId: string;
    item?: TurnItem;
    status: string;
    toolCallId?: string;
    /** 扩展列（对齐 tokhub）：tool_call item 携带中断信息 {"interrupt": InterruptInfo}
     *  （kind/request_id/message/conversation_id/agent_id/metadata/resume 内嵌恢复决策） */
    extra?: Record<string, unknown>;
    createTime: string;
}

// ==================== MessagePart（有序片段，对齐 tokhub） ====================

/** 推理过程 part */
export interface ReasoningPart {
    type: 'reasoning';
    id: string;
    text: string;
    /** 是否正在流式生成中 */
    active?: boolean;
}

/** 工具调用 part */
export interface ToolCallPart {
    type: 'tool_call';
    id: string;
    toolCallId: string;
    toolName: string;
    arguments: string;
    status: TurnItemStatus;
    output?: string;
    durationMs?: number;
    active?: boolean;
    /** 恢复决策类型（对齐 tokhub extra.interrupt.resume.type：approved/rejected/params_completed...，
     *  历史从 item extra 还原，实时从已决策中断继承），驱动决议徽章 */
    resumeType?: string;
}

/** 上下文压缩提示 part（对齐 tokhub ContextCompaction 时间线节点） */
export interface CompactionPart {
    type: 'context_compaction';
    id: string;
    originalTokens: number;
    compressedTokens: number;
    compressedMessageCount: number;
}

/** 轮次终止提示 part（用户停止 / 流式错误，仅前端展示不落库） */
export interface NoticePart {
    type: 'notice';
    id: string;
    kind: 'stopped' | 'error';
    text: string;
}

/** 消息有序片段（对齐 tokhub MessagePart） */
export type MessagePart =
    | { type: 'text'; id: string; text: string }
    | ReasoningPart
    | ToolCallPart
    | CompactionPart
    | NoticePart;

// ==================== ChatMessage（前端展示用） ====================

/** 附件元数据（对齐 tokhub extra.attachments，Phase 6 附件链路填充） */
export interface MessageAttachment {
    /** 文件名 */
    name: string;
    /** 附件种类：图片 / 内联文本 / 其它文件 */
    kind: 'image' | 'text' | 'file';
    /** MIME 类型 */
    mime?: string;
    /** 字节大小 */
    size?: number;
    /** 图片 dataURL（kind=image） */
    dataUrl?: string;
    /** 内联文本内容（kind=text） */
    text?: string;
}

export interface ChatMessage {
    id: string;
    role: 'user' | 'assistant' | 'internal';
    content: string;
    /** 有序片段（reasoning / tool_call / text 交错排列） */
    parts?: MessagePart[];
    /** 结构化内容段（用户消息芯片引用，随消息持久化，回显可恢复芯片样式） */
    segments?: ContentSegment[];
    /** 附件列表（用户消息展示用） */
    attachments?: MessageAttachment[];
    time?: string;
    turnId?: string;
    /** 是否正在流式生成中 */
    streaming?: boolean;
}

// ==================== StreamCallbacks ====================

export interface StreamCallbacks {
    onTurnStart?: (turnId: string, conversationId?: number) => void;
    onItemStarted?: (item: TurnItem, turnId: string) => void;
    onItemUpdated?: (itemId: string, delta: ItemDelta, turnId: string) => void;
    onItemCompleted?: (item: TurnItem, turnId: string) => void;
    onInterrupted?: (interrupt: InterruptEvent, turnId: string) => void;
    onTurnCompleted?: (turnId: string, status?: string, usage?: TurnUsage) => void;
    onError?: (error: string, source?: string) => void;
    onEnd?: () => void;
    onConversationCreated?: (conversationId: number) => void;
    /** attach 成功：回放缓存快照完毕，后续为实时事件（续流入口，恢复生成中 UI） */
    onTurnAttached?: (turnId: string, conversationId?: number) => void;
    /** attach 目标无运行中 turn（无需处理，历史已加载） */
    onTurnNotRunning?: (conversationId?: number) => void;
    /** 连接就绪（首次连接/重连成功）：供调用方决定是否重新 attach 运行中 turn */
    onSocketOpen?: () => void;
}

// ==================== WebSocket 发送参数 ====================

/** interruptResume 消息 content 数组元素（对齐后端 resumePrams 批量协议） */
export interface ResumeEntry {
    turnId: string;
    interruptId: string;
    interruptType: string;
    action: string;
    payload?: Record<string, unknown>;
}
