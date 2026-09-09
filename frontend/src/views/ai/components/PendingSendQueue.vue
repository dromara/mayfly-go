<template>
    <div v-if="queue.length > 0" class="pending-queue">
        <!-- 队列标题栏 -->
        <div class="pending-queue__header">
            <div class="pending-queue__rule" />
            <span class="pending-queue__title">{{ t('ai.chat.pendingQueue', { count: queue.length }) }}</span>
            <div class="pending-queue__rule" />
        </div>

        <!-- 队列列表（原生 HTML5 DnD 拖拽排序，PendingSendQueue） -->
        <TransitionGroup name="pending-queue-item" tag="div" class="pending-queue__list">
            <div
                v-for="(item, index) in queue"
                :key="item.id"
                class="pending-queue__item group/queue-item"
                :class="{ 'is-dragging': draggingIndex === index, 'is-drag-over': dragOverIndex === index }"
                draggable="true"
                @dragstart="onDragStart(index, $event)"
                @dragend="onDragEnd"
                @dragover.prevent="dragOverIndex = index"
                @dragleave="dragOverIndex = -1"
                @drop.prevent="onDrop(index)"
            >
                <!-- 拖拽手柄 -->
                <GripVerticalIcon class="pending-queue__grip" />
                <!-- 序号徽章 -->
                <span class="pending-queue__index">{{ index + 1 }}</span>
                <!-- 消息预览（纯附件无文本时回退为附件名列表） -->
                <span class="pending-queue__content">{{
                    item.data.text || item.data.attachments?.map((a) => a.name).join(', ')
                }}</span>
                <!-- 操作按钮 -->
                <button
                    type="button"
                    class="pending-queue__action"
                    :title="t('ai.chat.editQueuedMessage')"
                    @click="$emit('edit', item)"
                >
                    <PencilIcon />
                </button>
                <button
                    type="button"
                    class="pending-queue__action pending-queue__action--remove"
                    :title="t('ai.chat.removeFromQueue')"
                    @click="$emit('remove', item.id)"
                >
                    <XIcon />
                </button>
            </div>
        </TransitionGroup>
    </div>
</template>

<script setup lang="ts">
/**
 * PendingSendQueue - 消息待发送队列展示组件
 * - 序号徽章 + 单行预览，明确发送顺序
 * - 原生 HTML5 DnD 拖拽排序（零外部依赖）
 * - 编辑（回填输入框）/ 删除
 */
import { GripVerticalIcon, PencilIcon, XIcon } from '@lucide/vue';
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';
import type { QueuedMessage } from '../stores/chatStore';

defineProps<{
    queue: QueuedMessage[];
}>();

const emit = defineEmits<{
    (e: 'edit', message: QueuedMessage): void;
    (e: 'remove', id: string): void;
    (e: 'reorder', fromIndex: number, toIndex: number): void;
}>();

const { t } = useI18n();

const draggingIndex = ref(-1);
const dragOverIndex = ref(-1);

const onDragStart = (index: number, e: DragEvent) => {
    e.dataTransfer?.setData('text/plain', String(index));
    if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
    draggingIndex.value = index;
};

const onDragEnd = () => {
    draggingIndex.value = -1;
    dragOverIndex.value = -1;
};

const onDrop = (toIndex: number) => {
    const fromIndex = Number(draggingIndex.value);
    draggingIndex.value = -1;
    dragOverIndex.value = -1;
    if (!Number.isNaN(fromIndex) && fromIndex !== toIndex) {
        emit('reorder', fromIndex, toIndex);
    }
};
</script>

<style scoped>
.pending-queue {
    padding: 0 24px 6px;
    flex-shrink: 0;
    /* 与消息列同宽居中（max-w-4xl） */
    width: 100%;
    max-width: 56rem;
    margin: 0 auto;
}

.pending-queue__header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
}

.pending-queue__rule {
    flex: 1;
    height: 1px;
    background: var(--el-border-color-lighter);
}

.pending-queue__title {
    font-size: 11px;
    font-weight: 500;
    color: var(--el-text-color-placeholder);
    font-variant-numeric: tabular-nums;
}

.pending-queue__list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 120px;
    overflow-y: auto;
}

.pending-queue__item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-lighter);
    cursor: grab;
    transition: background 0.15s ease-out, border-color 0.15s ease-out, opacity 0.15s ease-out, transform 0.15s ease-out;
}

.pending-queue__item:hover {
    background: var(--el-fill-color);
    border-color: var(--el-border-color-light);
}

.pending-queue__item.is-dragging {
    opacity: 0.4;
    transform: scale(0.98);
}

.pending-queue__item.is-drag-over {
    border-color: var(--el-color-primary-light-5);
    background: var(--el-color-primary-light-9);
}

.pending-queue__grip {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    color: var(--el-text-color-placeholder);
    opacity: 0.5;
    transition: opacity 0.15s ease-out;
}

.pending-queue__item:hover .pending-queue__grip {
    opacity: 0.8;
}

.pending-queue__index {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    flex-shrink: 0;
    border-radius: 50%;
    font-size: 10px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-darker);
    font-variant-numeric: tabular-nums;
}

.pending-queue__content {
    flex: 1;
    min-width: 0;
    font-size: 13px;
    line-height: 1.3;
    color: var(--el-text-color-regular);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.pending-queue__action {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    flex-shrink: 0;
    border: none;
    border-radius: 4px;
    padding: 0;
    background: transparent;
    cursor: pointer;
    color: var(--el-text-color-placeholder);
    opacity: 0;
    transition: opacity 0.15s ease-out, color 0.15s ease-out, background 0.15s ease-out;
}

.pending-queue__item:hover .pending-queue__action,
.pending-queue__action:focus-visible {
    opacity: 1;
}

.pending-queue__action:hover {
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-darker);
}

.pending-queue__action--remove:hover {
    color: var(--el-color-danger);
    background: var(--el-color-danger-light-9);
}

.pending-queue__action svg {
    width: 12px;
    height: 12px;
}

/* 入场/退场动画（motion 入场/退场） */
.pending-queue-item-enter-active {
    transition: opacity 0.15s ease-out, transform 0.15s cubic-bezier(0.22, 1, 0.36, 1);
}

.pending-queue-item-leave-active {
    transition: opacity 0.15s ease-out, transform 0.15s ease-out;
}

.pending-queue-item-enter-from {
    opacity: 0;
    transform: translateY(4px);
}

.pending-queue-item-leave-to {
    opacity: 0;
    transform: translateX(-12px);
}

@media (prefers-reduced-motion: reduce) {
    .pending-queue-item-enter-active,
    .pending-queue-item-leave-active {
        transition-duration: 0.01ms;
    }
}
</style>
