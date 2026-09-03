<template>
    <!-- 编辑模式：原地 textarea（对齐 tokhub UserMessageBubble 编辑态） -->
    <div v-if="isEditing" ref="editContainer" class="user-message-edit" @blur="onEditBlur">
        <textarea
            ref="editTextarea"
            v-model="editValue"
            class="user-message-edit__textarea"
            rows="1"
            @keydown="onEditKeydown"
            @input="autoResize"
        ></textarea>
        <div class="user-message-edit__actions">
            <Button
                variant="ghost"
                size="sm"
                class="h-7 rounded-lg px-2.5 text-xs text-muted-foreground hover:text-foreground"
                @mousedown.prevent
                @click="$emit('cancel-edit')"
            >
                <XIcon class="size-3" />
                {{ t('common.cancel') }}
            </Button>
            <Button
                size="sm"
                class="h-7 rounded-lg px-2.5 text-xs"
                :disabled="!editValue.trim()"
                @mousedown.prevent
                @click="sendEdit"
            >
                <ArrowUpIcon class="size-3" />
                {{ t('ai.chat.send') }}
            </Button>
        </div>
    </div>

    <!-- 展示模式 -->
    <Bubble v-else variant="muted" align="end" class="max-w-full">
        <!-- 14px + 1.6 行高显式对齐 AssistantMessageBubble（根字号被 app.scss 设为 14px，
             BubbleContent 默认 text-sm = 0.875rem 会缩成 12.25px，导致两侧字号不一致） -->
        <!-- 结构化 segments：芯片段渲染 InlineChip（对齐 tokhub InlineContentRenderer）；
             后端 buildUserSegments 保证用户消息恒有 segments（无芯片时为单条 input_text 段） -->
        <BubbleContent class="text-[14px] leading-[1.6]">
            <template v-for="(seg, i) in segments" :key="i">
                <InlineChip v-if="seg.type === 'resource' || seg.type === 'skill'" :segment="seg" />
                <template v-else>{{ seg.text }}</template>
            </template>
        </BubbleContent>

        <!-- 附件列表（元数据来自消息 extra.attachments，Phase 6 附件链路填充） -->
        <AttachmentGroup v-if="attachments?.length" class="flex-wrap">
            <AttachmentItem v-for="(att, i) in attachments" :key="i" :attachment="att" />
        </AttachmentGroup>
    </Bubble>
</template>

<script setup lang="ts">
/**
 * UserMessageBubble - 用户消息气泡
 * 对齐 tokhub UserMessageBubble：Bubble(muted) + 纯文本 + 附件列表
 * 编辑模式：原地 textarea 自动撑高，Enter 发送 / Esc 取消 / 失焦取消
 */
import { ArrowUpIcon } from '@lucide/vue';
import { nextTick, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Button } from '@/components/ui/button';
import type { ContentSegment, MessageAttachment } from '../protocol/types';
import InlineChip from './InlineChip.vue';
import AttachmentItem from './AttachmentItem.vue';
import { AttachmentGroup } from '@/components/ui/attachment';
import { Bubble, BubbleContent } from '@/components/ui/bubble';

const props = defineProps<{
    content: string;
    /** 结构化内容段（芯片引用保留元数据，随消息持久化）；content 仅用于编辑重发 */
    segments?: ContentSegment[];
    attachments?: MessageAttachment[];
    /** 是否处于原地编辑模式 */
    isEditing?: boolean;
}>();

const emit = defineEmits<{
    (e: 'edit-send', content: string): void;
    (e: 'cancel-edit'): void;
}>();

const { t } = useI18n();

// ==================== 原地编辑 ====================

const editContainer = ref<HTMLElement | null>(null);
const editTextarea = ref<HTMLTextAreaElement | null>(null);
const editValue = ref('');

// 进入编辑态：同步内容 + 聚焦 + 光标到末尾 + 自动撑高
watch(
    () => props.isEditing,
    (editing) => {
        if (!editing) return;
        editValue.value = props.content;
        nextTick(() => {
            const ta = editTextarea.value;
            if (!ta) return;
            ta.focus();
            ta.setSelectionRange(ta.value.length, ta.value.length);
            autoResize();
            // 编辑态下滚动到可见
            editContainer.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        });
    },
);

const autoResize = () => {
    const ta = editTextarea.value;
    if (!ta) return;
    ta.style.height = 'auto';
    ta.style.height = `${ta.scrollHeight}px`;
};

const sendEdit = () => {
    const trimmed = editValue.value.trim();
    if (!trimmed) return;
    emit('edit-send', trimmed);
};

const onEditKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendEdit();
    } else if (e.key === 'Escape') {
        e.preventDefault();
        emit('cancel-edit');
    }
};

// 失焦取消编辑（焦点移入容器内按钮时按钮已 preventDefault 阻断失焦，不会触发）
const onEditBlur = (e: FocusEvent) => {
    const related = e.relatedTarget as Node | null;
    if (related && editContainer.value?.contains(related)) return;
    emit('cancel-edit');
};
</script>

<style scoped>
.user-message-edit {
    display: flex;
    flex-direction: column;
    gap: 8px;
    /* 限宽右对齐（用户消息在右侧）：避免编辑态撑满整列、
       呼吸环直抵消息列边界（对齐气泡视觉宽度） */
    width: 100%;
    max-width: 42rem;
    /* 呼吸环缓冲：padding 在列内留出 ring 空间。
       不可用负 margin 越界抵消——越界部分会被 message-list-wrapper
       的 overflow:hidden / 视口横向裁剪切掉（右侧环缺口 bug） */
    margin-left: auto;
    padding: 3px;
}

.user-message-edit__textarea {
    width: 100%;
    min-height: 40px;
    max-height: 160px;
    resize: none;
    border: 1px solid var(--el-border-color);
    border-radius: 12px;
    background: var(--el-bg-color);
    padding: 8px 12px;
    font-size: 14px;
    line-height: 1.6;
    color: var(--el-text-color-primary);
    outline: none;
    transition: border-color 0.15s ease-out, box-shadow 0.15s ease-out;
    overflow-y: auto;
}

.user-message-edit__textarea:focus {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 3px var(--el-color-primary-light-8);
}

.user-message-edit__actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 6px;
}
</style>
