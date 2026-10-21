/**
 * AI Chat 协议类型定义
 * 对齐后端 protocol/ 包的 TurnItem 统一协议
 */

// ==================== ContentSegment ====================

export type ContentSegmentType = 'input_text' | 'output_text' | 'skill' | 'resource' | 'image';

/**
 * 图片段类型的单点常量：展示侧要把它从正文里剔出去、改渲缩略图卡片。
 * 写成裸字面量会在「段渲染 + 是否含正文判定 + 卡片构建 + 产出端」四处各写一遍，
 * 新增段类型时漏改一处就静默丢内容
 */
export const IMAGE_SEGMENT_TYPE = 'image' satisfies ContentSegmentType;

export interface ContentSegment {
    type: ContentSegmentType;
    text: string;
    /** 芯片段元数据（typed segment 贯穿持久化：resource 含 resourceType/id/code/ip/port/authCertName/username） */
    extra?: Record<string, unknown>;
}

// ==================== TurnItem (tagged union) ====================

export type TurnItemTypeType = 'message' | 'reasoning' | 'tool_call' | 'context_compaction' | 'turn_notice';

export type TurnItemStatus = 'pending' | 'success' | 'failed' | 'cancelled' | 'interrupted';

/**
 * TurnItem 统一协议（扁平变体，对齐后端 protocol.TurnItem 的 tagged enum：
 * 变体字段直接位于顶层、snake_case 命名，实时流与历史回放共享同一协议形状）
 */
export interface TurnItem {
    type: TurnItemTypeType;
    id: string;

    // message 变体
    role?: string;
    content?: ContentSegment[];
    /** 用户消息附件元数据（fileKey 轻量引用随 payload 持久化，历史回显卡片/预览；内容已落文件服务） */
    attachments?: MessageAttachment[];

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

    // turn_notice 变体：轮次终止提示的种类（stopped=用户显式停止）。
    // 入库的是种类而不是句子：句子会把写入那一刻的语言固化进历史
    notice_kind?: string;
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
    /** 累计提示词 token（后端 vo.ConversationVO 直接下发，totalTokens = prompt + completion） */
    promptTokens: number;
    /** 累计生成词 token */
    completionTokens: number;
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
    /** 扩展列：tool_call item 携带中断信息 {"interrupt": InterruptInfo}
     *  （kind/request_id/message/conversation_id/agent_id/metadata/resume 内嵌恢复决策） */
    extra?: Record<string, unknown>;
    createTime: string;
}

// ==================== MessagePart（有序片段） ====================

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
    /** 恢复决策类型（extra.interrupt.resume.type：approved/rejected/params_completed...，
     *  历史从 item extra 还原，实时从已决策中断继承），驱动决议徽章 */
    resumeType?: string;
    /**
     * 中断类型（extra.interrupt.type）：决议徽章允许由中断 handler 覆盖文案，
     * 只有 resumeType 而缺它时，历史回放拿不到 handler，徽章会回退成通用措辞
     * 与实时不一致（双路径分叉）
     */
    interruptKind?: string;
}

/** 上下文压缩提示 part（ContextCompaction 时间线节点） */
export interface CompactionPart {
    type: 'context_compaction';
    id: string;
    originalTokens: number;
    compressedTokens: number;
    compressedMessageCount: number;
}

/**
 * 轮次终止提示 part。
 *
 * kind=stopped 现在**也持久化**（后端写一条 turn_notice item）：只活在客户端内存里时，刷新后那轮
 * 只剩一段在句子中途断开的回答，与模型正常答完完全同形。kind=error 仍只由实时事件写入、不落库：
 * 它是本次连接的故障读数，刷新后拿不到当时的错误上下文，落库反而会把瞬时故障固化成永久标记
 */
export interface NoticePart {
    type: 'notice';
    id: string;
    kind: 'stopped' | 'error';
    text: string;
}

// 落库的 `notice_kind` → 文案键与展示归类属展示语义，登记表在 registries/noticeKinds.ts
// （协议层只定线格式字段，不承载文案键）

/** 消息有序片段（MessagePart） */
export type MessagePart = { type: 'text'; id: string; text: string } | ReasoningPart | ToolCallPart | CompactionPart | NoticePart;

/**
 * 消息段元数据（单一真源）：某类段是否属于「过程段」（归入思考/工具折叠区）、
 * 是否具备正文文本语义。
 *
 * 此前这些判定硬编码在三处：AssistantMessageBubble 的 isProcessPart 写死
 * `reasoning | tool_call`、气泡分组分支只认 `text`、ProcessGroupEvents 用 v-if 链。
 * 新增一种段类型会在三处各被静默丢一次（内容凭空消失且无任何报错）
 */
export interface PartMeta {
    /** 过程段：归入「思考 · 工具」折叠区而非正文 */
    isProcess: boolean;
    /** 有正文文本语义（可直接当文本渲染） */
    isTextual: boolean;
}

export const PART_META: Readonly<Record<MessagePart['type'], PartMeta>> = {
    text: { isProcess: false, isTextual: true },
    reasoning: { isProcess: true, isTextual: false },
    tool_call: { isProcess: true, isTextual: false },
    context_compaction: { isProcess: false, isTextual: false },
    notice: { isProcess: false, isTextual: false },
};

/** 未登记的段类型返回 undefined：消费方必须显式处理（给可见提示），不得静默丢弃 */
export function partMeta(type: string): PartMeta | undefined {
    return PART_META[type as MessagePart['type']];
}

// ==================== ChatMessage（前端展示用） ====================

/**
 * 附件元数据（内容经统一文件服务落 local/S3，随 TurnItem payload 持久化的
 * 仅是 fileKey 轻量引用，对齐后端 protocol.AttachmentMeta）
 */
export interface MessageAttachment {
    /** 文件名 */
    name: string;
    /** 附件种类：图片 / 内联文本 / 其它文件 */
    kind: 'image' | 'text' | 'file';
    /** MIME 类型 */
    mime?: string;
    /** 字节大小 */
    size?: number;
    /** 文件服务 key（内容已落 t_sys_file，经 /sys/files/{fileKey} 访问；发送前上传获得） */
    fileKey?: string;
    /** 图片 dataURL（仅发送前本地预览态：上传前输入框预览与乐观回显，不随消息发送） */
    dataUrl?: string;
    /** 内联文本内容（仅发送前本地态：构建 LLM content 用，不随消息发送，历史预览经 fileKey 拉取） */
    text?: string;
    /** 原始 File（仅发送前内存持有，上传后即无意义，不参与序列化/持久化） */
    rawFile?: File;
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
