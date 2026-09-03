/**
 * Chat Store - 按 conversationId 隔离状态（parts 驱动）
 * 对齐 tokhub 的 Zustand chatStore + MessagePart 模型
 *
 * 核心设计：
 * - 按 convId 隔离的 ConvSlice：messages, pendingInterrupts, streaming 状态
 * - ensureTurnAssistantMessage：同 turn assistant 消息挂载点查找的单一出处
 * - LRU 淘汰：MAX_CACHED_CONVS = 8
 * - 消息队列：Agent 未完全空闲时新消息入队
 * - parts 驱动：每个 assistant message 有有序 parts 数组（reasoning / tool_call / text 交错）
 */

import { defineStore } from 'pinia';
import { ref, computed, reactive } from 'vue';
import type { ChatMessage, Conversation, InterruptEvent, TurnItem, ItemDelta, TurnGroupVO, MessagePart, NoticePart, ToolCallPart } from '../protocol/types';
import { forEachInterruptHandler } from '../registries/interruptRegistry';
import { ToolCallStatus } from '../registries/statuses';

const MAX_CACHED_CONVS = 8;

/** 格式化时间为 HH:mm */
function formatTime(date: Date): string {
    return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
}

/** item 创建时间（TurnItemVO.createTime ISO 字符串）→ 展示时间；
 *  实时流式路径无落库时间，回退当前时间（即真实创建时刻） */
function formatItemTime(createTime?: string): string {
    if (createTime) {
        const d = new Date(createTime);
        if (!Number.isNaN(d.getTime())) return formatTime(d);
    }
    return formatTime(new Date());
}

/** 每个会话的隔离状态 */
export interface ConvSlice {
    messages: ChatMessage[];
    pendingInterrupts: InterruptEvent[];
    active: boolean;
    loaded: boolean;
    activeTurnId: string | null;
    hasMoreMessages: boolean;
    isLoadingMoreMessages: boolean;
    oldestTurnId: string | null;
    /** streaming 临时状态：itemId → TurnItem */
    streamingItems: Map<string, TurnItem>;
    /** 当前正在流式构建的 assistant message ID（同一 turn 内复用） */
    currentAssistantMsgId: string | null;
    /** 消息队列：Agent 未完全空闲时新消息入队 */
    queuedMessages: QueuedMessage[];
    /** 滚动锚点：离开时视口顶部可见消息 ID + 视口内偏移（对齐 tokhub scrollAnchorStore） */
    scrollAnchor: ScrollAnchor | null;
}

/** 滚动锚点：不存 scrollTop（动态高度下不稳定），存消息 ID + 视口内偏移（px，可为负 = 长消息内深度） */
export interface ScrollAnchor {
    id: string;
    offset: number;
}

/** 队列中的消息 */
export interface QueuedMessage {
    id: string;
    content: string;
    timestamp: number;
}

export const useChatStore = defineStore('ai-chat', () => {
    const selectedConvId = ref<number | null>(null);
    const convs = reactive(new Map<number, ConvSlice>());
    const conversations = ref<Conversation[]>([]);

    // ===== 运行中会话（会话列表执行中指示器） =====
    // 全局集合（跨 ConvSlice 生命周期）：turn 级事件增删，进入页面时用
    // GET /ai/chat/running-turns 校正（其它端/页面刷新期间开始的 turn）
    const runningConvIds = reactive(new Set<number>());

    function markConvRunning(convId: number) {
        runningConvIds.add(convId);
    }

    function markConvIdle(convId: number) {
        runningConvIds.delete(convId);
    }

    /** 服务端运行中列表整体校正：server 列表 ∪ 本页正在驱动的会话（activeTurnId 非空），
     *  避免拉取与事件流之间的竞态把本页刚开始的 turn 误清 */
    function syncRunningConvs(ids: number[]) {
        const merged = new Set(ids);
        for (const [id, slice] of convs) {
            if (slice.activeTurnId) merged.add(id);
        }
        runningConvIds.clear();
        for (const id of merged) runningConvIds.add(id);
    }

    function isConvRunning(convId: number): boolean {
        return runningConvIds.has(convId);
    }

    // ===== Slice 管理 =====

    function getOrCreateSlice(convId: number): ConvSlice {
        let slice = convs.get(convId);
        if (!slice) {
            slice = {
                messages: [],
                pendingInterrupts: [],
                active: true,
                loaded: false,
                activeTurnId: null,
                hasMoreMessages: true,
                isLoadingMoreMessages: false,
                oldestTurnId: null,
                streamingItems: new Map(),
                currentAssistantMsgId: null,
                queuedMessages: [],
                scrollAnchor: null,
            };
            convs.set(convId, slice);
            evictIfNeeded();
        }
        return slice;
    }

    function evictIfNeeded() {
        if (convs.size <= MAX_CACHED_CONVS) return;
        for (const [id, slice] of convs) {
            if (!slice.active && id !== selectedConvId.value) {
                convs.delete(id);
                if (convs.size <= MAX_CACHED_CONVS) break;
            }
        }
    }

    function getSlice(convId: number): ConvSlice | undefined {
        return convs.get(convId);
    }

    function selectConv(conv: Conversation) {
        selectedConvId.value = conv.id;
        getOrCreateSlice(conv.id);
    }

    // ===== 消息操作 =====

    function addMessage(convId: number, msg: ChatMessage) {
        const slice = getOrCreateSlice(convId);
        if (!msg.time) msg.time = formatTime(new Date());
        slice.messages.push(msg);
    }

    function updateMessages(convId: number, msgs: ChatMessage[]) {
        const slice = getOrCreateSlice(convId);
        slice.messages = msgs;
        slice.loaded = true;
    }

    function prependMessages(convId: number, msgs: ChatMessage[]) {
        const slice = getOrCreateSlice(convId);
        slice.messages = [...msgs, ...slice.messages];
    }

    // ===== Turn 级事件管理 =====

    function loadTurnGroups(convId: number, groups: TurnGroupVO[]) {
        const slice = getOrCreateSlice(convId);
        // 历史组装 = 事件重放（与流式共用 reducer），新历史拼接在已有消息之前
        replayTurnGroups(convId, groups);
        slice.loaded = true;

        // 从最新 turn group 恢复 pending 中断（仅检查最新一轮，旧轮的中断已处理）
        if (groups.length > 0) {
            const latestGroup = groups[0]; // API 返回最新在前
            // 该 turn 中已执行完（success/failed/cancelled，cancelled 为恢复后被拒的
            // 真实终态）的工具调用：对应中断已被用户处理过，不应再恢复为待补全
            const resolvedToolCallIds = new Set<string>();
            for (const itemVO of latestGroup.items) {
                if (
                    itemVO.itemType === 'tool_call'
                    && (itemVO.status === ToolCallStatus.Success
                        || itemVO.status === ToolCallStatus.Failed
                        || itemVO.status === ToolCallStatus.Cancelled)
                    && itemVO.toolCallId
                ) {
                    resolvedToolCallIds.add(itemVO.toolCallId);
                }
            }
            for (const itemVO of latestGroup.items) {
                const item = itemVO.item;
                // 中断信息存于 tool_call item 的 extra 列（对齐 tokhub，不产生独立 internal item）
                if (!item || item.type !== 'tool_call') continue;
                const itemExtra = itemVO.extra;
                const interruptInfo = itemExtra?.interrupt as Record<string, unknown> | undefined;
                // kind 为中断类型短名（approval / param_completion，对齐 tokhub）
                const kind = interruptInfo ? String(interruptInfo.kind || '') : '';
                if (!kind) continue;
                // 尝试用各处理器恢复中断
                forEachInterruptHandler((handler) => {
                    const state = handler.buildFromHistoryItem(item, itemExtra);
                    if (state && handler.isPending(state)) {
                        // 中断对应的工具调用已有执行结果，说明用户已补全/审批过，跳过恢复
                        if (state.toolCallId && resolvedToolCallIds.has(state.toolCallId)) return;
                        // 构造 InterruptEvent 并添加到 pendingInterrupts
                        const interruptEvt: InterruptEvent = {
                            actionId: state.interruptId,
                            type: state.kind,
                            description: state.description,
                            toolName: state.toolName,
                            toolCallId: state.toolCallId,
                            turnId: latestGroup.turnId,
                            metadata: state.metadata,
                        };
                        // 去重：按 actionId 或 toolCallId 避免重复添加
                        const exists = slice.pendingInterrupts.some(
                            (i) => i.actionId === interruptEvt.actionId
                                || (interruptEvt.toolCallId && i.toolCallId === interruptEvt.toolCallId && i.type === interruptEvt.type),
                        );
                        if (!exists) {
                            slice.pendingInterrupts.push(interruptEvt);
                        }
                    }
                });
            }
        }
    }

    /**
     * 用户 TurnItem → ChatMessage
     * 流式 onItemStarted 与历史加载共用的组装单一出处（避免双路径改一漏一）；
     * 结构化 segments 透传使芯片引用回显可恢复样式
     */
    function userItemToMessage(item: TurnItem, turnId?: string, createTime?: string): ChatMessage {
        const segments = item.content ?? [];
        return {
            id: item.id,
            role: 'user',
            content: extractTextFromSegments(segments),
            segments,
            ...(turnId ? { turnId } : {}),
            time: formatItemTime(createTime),
        };
    }

    /**
     * 将 TurnGroupVO 转换为 ChatMessage[]：事件重放式组装。
     * 与流式推送共用同一 reducer（onItemStarted → onItemCompleted），
     * 消除「历史组装/流式组装」双套逻辑——渲染效果天然一致，
     * TurnItem 模型加字段不再需要双路径同步
     */
    function replayTurnGroups(convId: number, groups: TurnGroupVO[]): ChatMessage[] {
        const slice = getOrCreateSlice(convId);
        const prevMessages = slice.messages;
        // 在干净状态上重放（不污染当前流式中间态），完成后与已有消息拼接
        slice.messages = [];
        slice.streamingItems.clear();
        slice.currentAssistantMsgId = null;
        // API 返回最新在前（ORDER BY MIN(turn_id) DESC），反转恢复时间正序后逐 item 重放
        for (const group of [...groups].reverse()) {
            for (const itemVO of group.items) {
                const item = itemVO.item;
                if (!item) continue;
                // createTime 随 item_started 传入：消息时间显示落库创建时间而非本地当前时间
                onItemStarted(convId, item, group.turnId, itemVO.extra, itemVO.createTime);
                onItemCompleted(convId, item, group.turnId, itemVO.extra);
            }
            // 历史落库 item 均已终态：重放完立即定格，清除 onItemStarted 留下的
            // streaming/active 标志。否则刷新后 attach 前置重置（resetConvStreaming）
            // 会把这些历史 assistant 消息误判为流式中间态整批删除，导致 AI 回复丢失
            finalizeTurn(convId, group.turnId);
        }
        const replayed = slice.messages;
        slice.messages = [...replayed, ...prevMessages];
        slice.streamingItems.clear();
        slice.currentAssistantMsgId = null;
        // 重放非流式场景，不应残留 activeTurnId（否则 onSocketOpen 会误触发 attach）
        slice.activeTurnId = null;
        return replayed;
    }

    /** 从 ContentSegment 数组提取纯文本 */
    function extractTextFromSegments(segments: Array<{ type: string; text?: string }>): string {
        if (!Array.isArray(segments)) return '';
        return segments
            .filter((s) => s.type === 'output_text' || s.type === 'input_text')
            .map((s) => s.text || '')
            .join('');
    }

    /** 从 parts 数组构建 content 字符串（用于 fallback 显示） */
    function buildContentFromParts(parts: MessagePart[]): string {
        return parts
            .map((p) => {
                if (p.type === 'text') return p.text;
                if (p.type === 'reasoning') return p.text;
                if (p.type === 'tool_call') return `[${p.toolName}]`;
                return '';
            })
            .join('');
    }

    // ===== Streaming 状态管理 =====

    /** createTime：落库 item 创建时间（历史重放传入），实时流式路径不传即用当前时间 */
    function onItemStarted(convId: number, item: TurnItem, turnId: string, extra?: Record<string, unknown>, createTime?: string) {
        const slice = getOrCreateSlice(convId);
        slice.activeTurnId = turnId;
        slice.streamingItems.set(item.id, { ...item });

        // 根据 item 类型决定如何创建/更新 message
        if (item.type === 'message' && item.role === 'user') {
            // 用户消息：独立一条（组装逻辑与历史加载共用 userItemToMessage）
            slice.messages.push(userItemToMessage(item, turnId, createTime));
        } else if (isAssistantType(item.type)) {
            // 助手类型：检查是否已有当前 turn 的 assistant message
            let assistantMsg = findCurrentAssistantMessage(slice, turnId);
            if (!assistantMsg) {
                // 创建新的 assistant message（id 与 ensureTurnAssistantMessage/历史重放一致，
                // 三处统一后 scrollAnchor 按消息 id 恢复在切换/刷新后仍有效）
                assistantMsg = {
                    id: `assistant-${turnId}`,
                    role: 'assistant',
                    content: '',
                    parts: [],
                    turnId,
                    // 首个 assistant item 的落库创建时间（历史重放）；实时流式为当前时间
                    time: formatItemTime(createTime),
                    streaming: true,
                };
                slice.messages.push(assistantMsg);
                slice.currentAssistantMsgId = assistantMsg.id;
            }
            // tool_call 类型：resume 流程中 itemId 与原始 part 不同，
            // 通过 toolCallId 匹配已有 part，避免创建重复 part
            if (item.type === 'tool_call' && item.tool_call_id) {
                const tcId = item.tool_call_id;
                const existingPart = assistantMsg.parts?.find(
                    (p): p is ToolCallPart => p.type === 'tool_call' && p.toolCallId === tcId,
                );
                if (existingPart) {
                    // 更新 part 的 id 为新 itemId，使后续 onItemCompleted 能匹配
                    existingPart.id = item.id;
                    return;
                }
            }
            // 同 id part 已存在则跳过：attach 回放会把 item_started 重发一遍，
            // 若不幂等会为 reasoning/message 重复 push 空 part（残留一堆「思考中...」占位）
            if (assistantMsg.parts?.some((p) => p.id === item.id)) {
                return;
            }
            // 添加对应的 part
            const part = itemToPart(item, extra);
            if (part) {
                assistantMsg.parts!.push(part);
            }
        }
    }

    function onItemUpdated(convId: number, itemId: string, delta: ItemDelta, turnId: string) {
        const slice = getOrCreateSlice(convId);
        const streamingItem = slice.streamingItems.get(itemId);
        if (!streamingItem) return;

        // 找到对应的 message 并更新 part
        const assistantMsg = findCurrentAssistantMessage(slice, turnId);
        if (!assistantMsg || !assistantMsg.parts) return;

        let partIndex = assistantMsg.parts.findIndex((p) => p.id === itemId);
        // fallback: 通过 toolCallId 匹配（resume 流程中 part id 可能尚未更新）
        if (partIndex === -1 && streamingItem.type === 'tool_call') {
            const tcId = streamingItem.tool_call_id;
            if (tcId) {
                partIndex = assistantMsg.parts.findIndex(
                    (p): p is ToolCallPart => p.type === 'tool_call' && p.toolCallId === tcId,
                );
            }
        }
        if (partIndex === -1) return;

        const part = assistantMsg.parts[partIndex];
        if (delta.text) {
            if (part.type === 'reasoning') {
                part.text += delta.text;
            } else if (part.type === 'tool_call') {
                part.arguments += delta.text;
            } else if (part.type === 'text') {
                part.text += delta.text;
            }
            // 同步更新 content（用于 fallback 显示）
            assistantMsg.content = buildContentFromParts(assistantMsg.parts);
        }
    }

    function onItemCompleted(convId: number, item: TurnItem, turnId: string, extra?: Record<string, unknown>) {
        const slice = getOrCreateSlice(convId);
        slice.streamingItems.delete(item.id);

        // 更新对应 part 为终态
        const assistantMsg = findCurrentAssistantMessage(slice, turnId);
        if (assistantMsg && assistantMsg.parts) {
            let partIndex = assistantMsg.parts.findIndex((p) => p.id === item.id);
            // fallback: 通过 toolCallId 匹配（resume 流程中 itemId 与原始 part 不同）
            if (partIndex === -1 && item.type === 'tool_call' && item.tool_call_id) {
                const tcId = item.tool_call_id;
                partIndex = assistantMsg.parts.findIndex(
                    (p): p is ToolCallPart => p.type === 'tool_call' && p.toolCallId === tcId,
                );
            }
            if (partIndex !== -1) {
                // 用完整的 item 数据替换 part
                const completedPart = itemToPart(item, extra);
                if (completedPart) {
                    // 实时完成事件无 extra：继承原 part 的恢复决策类型，
                    // 避免替换后决议徽章（已批准/已完善等）丢失
                    const prevPart = assistantMsg.parts[partIndex];
                    if (
                        completedPart.type === 'tool_call' && !completedPart.resumeType
                        && prevPart?.type === 'tool_call'
                    ) {
                        completedPart.resumeType = prevPart.resumeType;
                    }
                    assistantMsg.parts[partIndex] = completedPart;
                }
                // content fallback 统一从 parts 重建（与流式增量路径同语义，
                // 多段正文时不再互相覆盖）
                assistantMsg.content = buildContentFromParts(assistantMsg.parts);
            }
        }

        // 注意：不在此处清除 currentAssistantMsgId！
        // 同一 turn 内可能有多个 item（tool_call → text → tool_call ...），
        // 过早清除会导致后续 item 创建新的 assistant message，造成消息分裂。
        // 由 finalizeTurn 在 turn 结束时统一清除。
    }

    /**
     * 确保该 turn 存在 assistant 消息（无则创建并缓存为当前流式消息），返回该消息。
     * 恢复流挂载（useChatResume）/ attach 续流（useChatCallbacks）/ notice 写入
     * （appendTurnNoticePart）共用的查找与创建单一出处
     */
    function ensureTurnAssistantMessage(convId: number, turnId: string): ChatMessage {
        const slice = getOrCreateSlice(convId);
        // 与 findCurrentAssistantMessage 回退语义一致：取该 turn 最后一条 assistant 消息
        for (let i = slice.messages.length - 1; i >= 0; i--) {
            const msg = slice.messages[i];
            if (msg.role === 'assistant' && msg.turnId === turnId) return msg;
        }
        const msg: ChatMessage = {
            id: `assistant-${turnId}`,
            role: 'assistant',
            content: '',
            parts: [],
            turnId,
            time: formatTime(new Date()),
        };
        slice.messages.push(msg);
        slice.currentAssistantMsgId = msg.id;
        return msg;
    }

    /**
     * 向指定 turn 的 assistant 消息追加提示 part（用户停止/流式错误）。
     * turn 内无 assistant 消息时创建一条（与流式创建同 ID 规则，历史加载时
     * notice part 不落库、自然消失）
     */
    function appendTurnNoticePart(convId: number, turnId: string | null | undefined, part: NoticePart) {
        const slice = getOrCreateSlice(convId);
        let msg: ChatMessage;
        if (turnId) {
            msg = ensureTurnAssistantMessage(convId, turnId);
        } else {
            msg = {
                id: `assistant-${Date.now()}`,
                role: 'assistant',
                content: '',
                parts: [],
                time: formatTime(new Date()),
            };
            slice.messages.push(msg);
        }
        (msg.parts ??= []).push(part);
    }

    /**
     * attach 回放前置重置：清除该会话的流式中间态（流式 assistant message +
     * streamingItems），由回放事件全量重建。否则重复 attach（切会话来回/
     * 断线重连）会在旧 parts 上叠加，item_updated 只更新首个匹配，残留大量空占位。
     * 返回是否删除了流式消息：若 attach 后发现 turn 已结束（turn_not_running），
     * 调用方需重新拉取历史回填，否则被删内容无法恢复
     */
    function resetConvStreaming(convId: number): boolean {
        const slice = convs.get(convId);
        if (!slice) return false;
        const hadStreaming = slice.messages.some((m) => m.role === 'assistant' && m.streaming);
        slice.messages = slice.messages.filter((m) => !(m.role === 'assistant' && m.streaming));
        slice.streamingItems.clear();
        slice.currentAssistantMsgId = null;
        slice.activeTurnId = null;
        return hadStreaming;
    }

    /** turn 结束时清除流式状态（对齐 tokhub cleanupStream） */
    function finalizeTurn(convId: number, turnId: string) {
        const slice = getOrCreateSlice(convId);
        // 清除 assistant 消息的 streaming 标志
        for (const msg of slice.messages) {
            if (msg.turnId === turnId && msg.streaming) {
                msg.streaming = false;
                // 清除 process parts 的 active 标志
                if (msg.parts) {
                    for (const part of msg.parts) {
                        if (part.type === 'reasoning' || part.type === 'tool_call') {
                            part.active = false;
                        }
                    }
                }
            }
        }
        // 注意：不清除 pendingInterrupts，中断需要等用户操作后才移除
    }

    /** 判断是否为助手类型 item（历史重放与流式共用，新增类型两路径同时生效） */
    function isAssistantType(type: string): boolean {
        return type === 'message' || type === 'reasoning' || type === 'tool_call' || type === 'context_compaction';
    }

    /** 查找当前 turn 的 assistant message */
    function findCurrentAssistantMessage(slice: ConvSlice, turnId: string): ChatMessage | undefined {
        // 优先使用缓存的 ID
        if (slice.currentAssistantMsgId) {
            const msg = slice.messages.find((m) => m.id === slice.currentAssistantMsgId);
            if (msg && msg.turnId === turnId) return msg;
        }
        // 回退：找最后一条同 turn 的 assistant message
        for (let i = slice.messages.length - 1; i >= 0; i--) {
            const msg = slice.messages[i];
            if (msg.role === 'assistant' && msg.turnId === turnId) {
                return msg;
            }
        }
        return undefined;
    }

    /** 将 TurnItem 转换为 MessagePart（历史加载与流式共用的类型映射单一出处） */
    function itemToPart(item: TurnItem, extra?: Record<string, unknown>): MessagePart | null {
        switch (item.type) {
            case 'message':
                if (!item.content) return null;
                return { type: 'text', id: item.id, text: extractTextFromSegments(item.content) };
            case 'reasoning':
                return {
                    type: 'reasoning',
                    id: item.id,
                    text: item.text || '',
                };
            case 'tool_call': {
                if (!item.tool_call_id) return null;
                // 恢复决策类型：历史路径从 item extra.interrupt.resume 还原（对齐 tokhub，
                // 决策内嵌于 interrupt 对象）；实时路径无 extra，由已决策中断继承
                const interruptInfo = extra?.interrupt as Record<string, unknown> | undefined;
                const resume = interruptInfo?.resume as Record<string, unknown> | undefined;
                return {
                    type: 'tool_call',
                    id: item.id,
                    toolCallId: item.tool_call_id,
                    toolName: item.tool_name ?? '',
                    arguments: item.arguments ?? '',
                    status: item.status ?? ToolCallStatus.Pending,
                    output: item.output ?? '',
                    durationMs: item.duration_ms,
                    resumeType: resume ? String(resume.type || '') || undefined : undefined,
                };
            }
            case 'context_compaction':
                return {
                    type: 'context_compaction',
                    id: item.id,
                    originalTokens: item.original_tokens ?? 0,
                    compressedTokens: item.compressed_tokens ?? 0,
                    compressedMessageCount: item.compressed_message_count ?? 0,
                };
            default:
                return null;
        }
    }

    // ===== 中断管理 =====

    function addInterrupt(convId: number, interrupt: InterruptEvent) {
        const slice = getOrCreateSlice(convId);
        // 去重：按 actionId 或 toolCallId+type 避免重复
        const exists = slice.pendingInterrupts.some(
            (i) => i.actionId === interrupt.actionId
                || (interrupt.toolCallId && i.toolCallId === interrupt.toolCallId && i.type === interrupt.type),
        );
        if (!exists) {
            slice.pendingInterrupts.push(interrupt);
        }
    }

    function removeInterrupt(convId: number, event: InterruptEvent) {
        const slice = convs.get(convId);
        if (!slice) return;
        slice.pendingInterrupts = slice.pendingInterrupts.filter((i) => {
            // 按 actionId 精确匹配
            if (i.actionId === event.actionId) return false;
            // 按 toolCallId + type 匹配（处理 actionId 不一致的情况）
            if (event.toolCallId && i.toolCallId === event.toolCallId && i.type === event.type) return false;
            return true;
        });
    }

    // ===== 消息队列 =====

    function enqueueMessage(convId: number, content: string) {
        const slice = getOrCreateSlice(convId);
        slice.queuedMessages.push({
            id: `queued_${Date.now()}_${Math.random().toString(36).slice(2)}`,
            content,
            timestamp: Date.now(),
        });
    }

    function dequeueMessage(convId: number): QueuedMessage | undefined {
        const slice = convs.get(convId);
        if (!slice || slice.queuedMessages.length === 0) return undefined;
        return slice.queuedMessages.shift();
    }

    function hasQueuedMessages(convId: number): boolean {
        const slice = convs.get(convId);
        return (slice?.queuedMessages.length ?? 0) > 0;
    }

    /** 从队列移除指定消息（对齐 tokhub removeQueuedMessage） */
    function removeQueuedMessage(convId: number, messageId: string) {
        const slice = convs.get(convId);
        if (!slice) return;
        slice.queuedMessages = slice.queuedMessages.filter((m) => m.id !== messageId);
    }

    /** 队列拖拽重排序（对齐 tokhub reorderQueuedMessage） */
    function reorderQueuedMessage(convId: number, fromIndex: number, toIndex: number) {
        const slice = convs.get(convId);
        if (!slice) return;
        const list = slice.queuedMessages;
        if (fromIndex < 0 || fromIndex >= list.length || toIndex < 0 || toIndex >= list.length) return;
        const [moved] = list.splice(fromIndex, 1);
        list.splice(toIndex, 0, moved);
    }

    /** 更新队列指定消息内容（编辑后写回） */
    function updateQueuedMessage(convId: number, messageId: string, content: string) {
        const slice = convs.get(convId);
        const item = slice?.queuedMessages.find((m) => m.id === messageId);
        if (item) item.content = content;
    }

    /** 记录会话滚动锚点（视口顶部可见消息 + 偏移，对齐 tokhub scrollAnchorStore） */
    function setScrollAnchor(convId: number, anchor: ScrollAnchor) {
        getOrCreateSlice(convId).scrollAnchor = anchor;
    }

    function getScrollAnchor(convId: number): ScrollAnchor | null {
        return convs.get(convId)?.scrollAnchor ?? null;
    }

    // ===== 分页 =====

    function setHasMore(convId: number, hasMore: boolean) {
        const slice = getOrCreateSlice(convId);
        slice.hasMoreMessages = hasMore;
    }

    function setLoadingMore(convId: number, loading: boolean) {
        const slice = getOrCreateSlice(convId);
        slice.isLoadingMoreMessages = loading;
    }

    function setOldestTurnId(convId: number, turnId: string | null) {
        const slice = getOrCreateSlice(convId);
        slice.oldestTurnId = turnId;
    }

    function resetSlice(convId: number) {
        const slice = convs.get(convId);
        if (!slice) return;
        slice.messages = [];
        slice.pendingInterrupts = [];
        slice.activeTurnId = null;
        slice.streamingItems.clear();
        slice.currentAssistantMsgId = null;
        slice.queuedMessages = [];
        slice.loaded = false;
        slice.hasMoreMessages = true;
        slice.oldestTurnId = null;
    }

    // ===== Getters =====

    const currentMessages = computed(() => {
        if (selectedConvId.value === null) return [];
        const slice = convs.get(selectedConvId.value);
        return slice?.messages || [];
    });

    const currentInterrupts = computed(() => {
        if (selectedConvId.value === null) return [];
        const slice = convs.get(selectedConvId.value);
        return slice?.pendingInterrupts || [];
    });

    const isLoading = computed(() => {
        if (selectedConvId.value === null) return false;
        const slice = convs.get(selectedConvId.value);
        return slice?.activeTurnId !== null;
    });

    return {
        selectedConvId,
        conversations,
        // Slice 管理
        selectConv,
        getSlice,
        getOrCreateSlice,
        // 消息操作
        addMessage,
        updateMessages,
        prependMessages,
        // Turn 级事件
        loadTurnGroups,
        // Streaming
        onItemStarted,
        onItemUpdated,
        onItemCompleted,
        ensureTurnAssistantMessage,
        appendTurnNoticePart,
        finalizeTurn,
        // 中断
        addInterrupt,
        removeInterrupt,
        // 消息队列
        enqueueMessage,
        dequeueMessage,
        hasQueuedMessages,
        removeQueuedMessage,
        reorderQueuedMessage,
        updateQueuedMessage,
        // 滚动锚点持久化
        setScrollAnchor,
        getScrollAnchor,
        // 分页
        setHasMore,
        setLoadingMore,
        setOldestTurnId,
        resetSlice,
        // Getters
        currentMessages,
        currentInterrupts,
        isLoading,
        // 运行中会话指示器
        markConvRunning,
        markConvIdle,
        syncRunningConvs,
        isConvRunning,
        // attach 回放前置重置
        resetConvStreaming,
    };
});
