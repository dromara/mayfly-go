<template>
    <div class="assistant-message">
        <!-- parts 驱动分段渲染 -->
        <template v-if="renderSegments.length > 0">
            <template v-for="seg in renderSegments" :key="seg.key">
                <!-- 文本段：跳过空内容；Markdown 排版规则收敛在 MarkdownView -->
                <MarkdownView v-if="seg.type === 'text' && seg.text" :content="seg.text" />
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
                <div v-else-if="seg.type === 'notice'" class="assistant-message__notice" :class="`assistant-message__notice--${seg.part.kind}`">
                    <CircleStopIcon v-if="seg.part.kind === 'stopped'" class="assistant-message__notice-icon" />
                    <CircleAlertIcon v-else class="assistant-message__notice-icon" />
                    <span>{{ seg.part.text }}</span>
                </div>
            </template>
        </template>
        <!-- 无 parts 但有 content（向后兼容） -->
        <MarkdownView v-else-if="content" :content="content" />
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
 * AssistantMessageBubble：parts 驱动分段 + 流式占位
 */
import { CircleAlertIcon, CircleStopIcon } from '@lucide/vue';
import { computed } from 'vue';
import type { InterruptEvent, MessagePart } from '../protocol/types';
import type { InterruptActionEvent } from '../interrupt/types';
import { MarkdownView } from '@/components/markdown';
import ProcessGroup from './ProcessGroup.vue';
import CompactionPart from './events/CompactionPart.vue';
import { groupPartsIntoSegments, type RenderSegment } from './messageSegments';

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

// ==================== Parts 分组逻辑（messageSegments 纯函数） ====================

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
    0%,
    100% {
        opacity: 0.3;
    }
    50% {
        opacity: 1;
    }
}
</style>
