<template>
    <div class="assistant-message">
        <!-- parts 驱动分段渲染 -->
        <template v-if="renderSegments.length > 0">
            <template v-for="(seg, i) in renderSegments" :key="i">
                <!-- 文本段：跳过空内容，直接渲染 HTML -->
                <div
                    v-if="seg.type === 'text' && seg.text"
                    class="assistant-message__text"
                    v-html="renderMarkdown(seg.text)"
                />
                <!-- 过程段：ProcessGroup 折叠展示 -->
                <ProcessGroup
                    v-else-if="seg.type === 'process'"
                    :events="seg.parts"
                    :status="streaming ? 'active' : 'completed'"
                    :pending-interrupts="pendingInterrupts"
                    :turn-id="turnId"
                    @interrupt-action="(a) => $emit('interrupt-action', a)"
                />
                <!-- 上下文压缩提示节点 -->
                <CompactionPart
                    v-else-if="seg.type === 'compaction'"
                    :original-tokens="seg.part.originalTokens"
                    :compressed-tokens="seg.part.compressedTokens"
                    :compressed-message-count="seg.part.compressedMessageCount"
                />
                <!-- 轮次终止提示（用户停止 / 流式错误） -->
                <div
                    v-else-if="seg.type === 'notice'"
                    class="assistant-message__notice"
                    :class="`assistant-message__notice--${seg.part.kind}`"
                >
                    <CircleStopIcon v-if="seg.part.kind === 'stopped'" class="assistant-message__notice-icon" />
                    <CircleAlertIcon v-else class="assistant-message__notice-icon" />
                    <span>{{ seg.part.text }}</span>
                </div>
            </template>
        </template>
        <!-- 无 parts 但有 content（向后兼容） -->
        <div v-else-if="content" class="assistant-message__text" v-html="renderMarkdown(content)" />
        <!-- 流式占位：等待首个 item -->
        <div v-else-if="streaming" class="assistant-message__placeholder">
            <span class="assistant-message__dot" />
            <span class="assistant-message__dot" />
            <span class="assistant-message__dot" />
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * AssistantMessageBubble - 助手消息气泡
 * 对齐 tokhub AssistantMessageBubble：parts 驱动分段 + 流式占位
 */
import { CircleAlertIcon, CircleStopIcon } from '@lucide/vue';
import { computed } from 'vue';
import type { CompactionPart as CompactionPartType, InterruptEvent, MessagePart, NoticePart } from '../protocol/types';
import type { InterruptActionEvent } from '../interrupt/types';
import { renderMarkdown } from '../utils/markdown';
import ProcessGroup from './ProcessGroup.vue';
import CompactionPart from './events/CompactionPart.vue';

const props = defineProps<{
    content: string;
    parts?: MessagePart[];
    turnId?: string;
    pendingInterrupts?: InterruptEvent[];
    streaming?: boolean;
}>();

defineEmits<{
    (e: 'interrupt-action', action: InterruptActionEvent): void;
}>();

// ==================== Parts 分组逻辑（对齐 tokhub useMessageSegments） ====================

type RenderSegment =
    | { type: 'text'; text: string }
    | { type: 'process'; parts: MessagePart[] }
    | { type: 'compaction'; part: CompactionPartType }
    | { type: 'notice'; part: NoticePart };

/** 判断是否为过程类 part */
function isProcessPart(part: MessagePart): boolean {
    return part.type === 'reasoning' || part.type === 'tool_call';
}

/** 判断是否为压缩提示 part */
function isCompactionPart(part: MessagePart): part is CompactionPartType {
    return part.type === 'context_compaction';
}

/** 判断是否为轮次终止提示 part */
function isNoticePart(part: MessagePart): part is NoticePart {
    return part.type === 'notice';
}

/** 将 parts 分组为渲染段：连续 process parts 合并为一个 process 段 */
function groupPartsIntoSegments(parts: MessagePart[]): RenderSegment[] {
    const segments: RenderSegment[] = [];
    let processBuf: MessagePart[] = [];

    const flushProcess = () => {
        if (processBuf.length > 0) {
            segments.push({ type: 'process', parts: [...processBuf] });
            processBuf = [];
        }
    };

    for (const part of parts) {
        if (isProcessPart(part)) {
            processBuf.push(part);
        } else if (isCompactionPart(part)) {
            flushProcess();
            segments.push({ type: 'compaction', part });
        } else if (isNoticePart(part)) {
            flushProcess();
            segments.push({ type: 'notice', part });
        } else {
            flushProcess();
            if (part.type === 'text' && part.text?.trim()) {
                segments.push({ type: 'text', text: part.text });
            }
        }
    }
    flushProcess();
    return segments;
}

const renderSegments = computed((): RenderSegment[] => {
    if (!props.parts || props.parts.length === 0) return [];
    return groupPartsIntoSegments(props.parts);
});
</script>

<style scoped>
.assistant-message {
    display: flex;
    flex-direction: column;
    gap: 4px;
    line-height: 1.6;
    word-break: break-word;
    font-size: 14px;
    color: var(--el-text-color-primary);
}

.assistant-message__text :deep(pre) {
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
    padding: 12px;
    margin: 8px 0;
    overflow-x: auto;
    font-size: 13px;
    line-height: 1.5;
}

.assistant-message__text :deep(code) {
    background: var(--el-fill-color-lighter);
    border-radius: 3px;
    padding: 2px 6px;
    font-size: 13px;
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
}

.assistant-message__text :deep(pre code) {
    background: transparent;
    padding: 0;
}

.assistant-message__text :deep(a) {
    color: var(--el-color-primary);
    text-decoration: none;
}

.assistant-message__text :deep(a:hover) {
    text-decoration: underline;
}

.assistant-message__text :deep(strong) {
    font-weight: 600;
}

.assistant-message__text :deep(em) {
    font-style: italic;
}

/* markdown 标题/段落间距 */
.assistant-message__text :deep(h1),
.assistant-message__text :deep(h2),
.assistant-message__text :deep(h3),
.assistant-message__text :deep(h4) {
    margin: 12px 0 6px;
    font-weight: 600;
    line-height: 1.4;
}

.assistant-message__text :deep(h1):first-child,
.assistant-message__text :deep(h2):first-child,
.assistant-message__text :deep(h3):first-child {
    margin-top: 0;
}

.assistant-message__text :deep(h1) {
    font-size: 18px;
}

.assistant-message__text :deep(h2) {
    font-size: 16px;
}

.assistant-message__text :deep(h3),
.assistant-message__text :deep(h4) {
    font-size: 14px;
}

.assistant-message__text :deep(p) {
    margin: 4px 0;
}

/* 列表（Tailwind preflight 会重置 list-style，需恢复） */
.assistant-message__text :deep(ul),
.assistant-message__text :deep(ol) {
    margin: 4px 0;
    padding-left: 20px;
}

.assistant-message__text :deep(ul) {
    list-style-type: disc;
}

.assistant-message__text :deep(ol) {
    list-style-type: decimal;
}

.assistant-message__text :deep(li) {
    margin: 2px 0;
}

/* 表格 */
.assistant-message__text :deep(table) {
    border-collapse: collapse;
    margin: 8px 0;
    width: 100%;
    font-size: 13px;
}

.assistant-message__text :deep(th),
.assistant-message__text :deep(td) {
    border: 1px solid var(--el-border-color-lighter);
    padding: 6px 10px;
    text-align: left;
}

.assistant-message__text :deep(th) {
    background: var(--el-fill-color-lighter);
    font-weight: 600;
}

/* 引用块：1px 全边框淡底取代侧边色条（detect: side-tab） */
.assistant-message__text :deep(blockquote) {
    margin: 8px 0;
    padding: 6px 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
    color: var(--el-text-color-secondary);
}

/* 轮次终止提示：轻量行内提示（stopped 中性 / error 语义色） */
.assistant-message__notice {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 0;
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.assistant-message__notice--error {
    color: var(--el-color-danger);
}

.assistant-message__notice-icon {
    width: 13px;
    height: 13px;
    flex-shrink: 0;
}

.assistant-message__text :deep(hr) {
    border: none;
    border-top: 1px solid var(--el-border-color-lighter);
    margin: 12px 0;
}

/* 流式占位：三点脉冲 */
.assistant-message__placeholder {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 0;
}

.assistant-message__dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--el-text-color-placeholder);
    animation: assistant-dot-pulse 1.2s ease-in-out infinite;
}

.assistant-message__dot:nth-child(2) {
    animation-delay: 0.2s;
}

.assistant-message__dot:nth-child(3) {
    animation-delay: 0.4s;
}

@keyframes assistant-dot-pulse {
    0%, 100% {
        opacity: 0.3;
    }
    50% {
        opacity: 1;
    }
}
</style>
