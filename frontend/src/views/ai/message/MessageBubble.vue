<template>
    <Message :align="role === 'user' ? 'end' : 'start'" class="group/message py-1">
        <!-- 头像（仅助手消息显示） -->
        <div v-if="role === 'assistant'" class="message-bubble__avatar">
            <SvgIcon :size="18" name="icon ai/assistant" color="var(--el-color-primary)" />
        </div>

        <!-- 主体 -->
        <MessageContent class="flex-1">
            <!-- 内容分发（MessageBubble → UserMessageBubble / AssistantMessageBubble） -->
            <UserMessageBubble
                v-if="role === 'user'"
                :content="content"
                :segments="segments"
                :attachments="attachments"
                :is-editing="isEditing"
                @edit-send="(c) => $emit('edit-send', c)"
                @cancel-edit="$emit('cancel-edit')"
            />
            <AssistantMessageBubble
                v-else
                :content="content"
                :parts="parts"
                :turn-id="turnId"
                :pending-interrupts="pendingInterrupts"
                :streaming="streaming"
                @interrupt-action="(a) => $emit('interrupt-action', a)"
            />

            <!-- 底部操作栏 -->
            <MessageFooter class="px-0">
                <!-- 流式中：显示耗时（ActiveMessageFooter） -->
                <template v-if="streaming">
                    <Spinner class="size-3" />
                    <span>{{ t('common.processing') }} {{ elapsed }}s</span>
                </template>
                <template v-else>
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        class="message-bubble__action size-6 text-muted-foreground"
                        :title="t('common.copy')"
                        @click="handleCopy"
                    >
                        <CheckIcon v-if="copied" class="size-3.5 text-success" />
                        <CopyIcon v-else class="size-3.5" />
                    </Button>
                    <!-- 编辑用户消息（MessageTimestamp Pencil，hover 渐现） -->
                    <Button
                        v-if="role === 'user'"
                        variant="ghost"
                        size="icon-sm"
                        class="message-bubble__action size-6 text-muted-foreground"
                        :title="t('ai.chat.editMessage')"
                        @click="$emit('edit')"
                    >
                        <PencilIcon class="size-3.5" />
                    </Button>
                    <span v-if="time" class="message-bubble__time">{{ time }}</span>
                </template>
            </MessageFooter>
        </MessageContent>
    </Message>
</template>

<script setup lang="ts">
/**
 * MessageBubble - 消息气泡编排组件
 *
 * 职责：Message 布局 + 头像 + 内容分发（User/Assistant）+ Footer（时间戳/复制/流式耗时）
 * 内容渲染下沉到 UserMessageBubble / AssistantMessageBubble
 */
import { CheckIcon, CopyIcon, PencilIcon } from '@lucide/vue';
import { onBeforeUnmount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { copyToClipboard } from '@/common/utils/string';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Button } from '@/components/ui/button';
import { Message, MessageContent, MessageFooter } from '@/components/ui/message';
import { Spinner } from '@/components/ui/spinner';
import type { ContentSegment, InterruptEvent, MessageAttachment, MessagePart } from '../protocol/types';
import type { InterruptActionEvent } from '../interrupt/types';
import UserMessageBubble from './UserMessageBubble.vue';
import AssistantMessageBubble from './AssistantMessageBubble.vue';

const props = defineProps<{
    id: string;
    role: 'user' | 'assistant';
    content: string;
    parts?: MessagePart[];
    /** 结构化内容段（用户消息芯片引用回显） */
    segments?: ContentSegment[];
    attachments?: MessageAttachment[];
    time?: string;
    turnId?: string;
    pendingInterrupts?: InterruptEvent[];
    /** 是否正在流式生成中 */
    streaming?: boolean;
    /** 是否处于原地编辑模式（仅用户消息） */
    isEditing?: boolean;
}>();

const emit = defineEmits<{
    (e: 'interrupt-action', action: InterruptActionEvent): void;
    (e: 'edit'): void;
    (e: 'edit-send', content: string): void;
    (e: 'cancel-edit'): void;
}>();

const { t } = useI18n();

// ==================== 复制 ====================

const copied = ref(false);
let copiedTimer: ReturnType<typeof setTimeout> | null = null;

const handleCopy = async () => {
    await copyToClipboard(props.content);
    copied.value = true;
    if (copiedTimer) clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
        copied.value = false;
    }, 1500);
};

// ==================== 流式耗时 ====================

const elapsed = ref(0);
let elapsedTimer: ReturnType<typeof setInterval> | null = null;

watch(
    () => props.streaming,
    (streaming) => {
        if (streaming) {
            elapsed.value = 0;
            elapsedTimer = setInterval(() => {
                elapsed.value++;
            }, 1000);
        } else if (elapsedTimer) {
            clearInterval(elapsedTimer);
            elapsedTimer = null;
        }
    },
    { immediate: true },
);

onBeforeUnmount(() => {
    if (copiedTimer) clearTimeout(copiedTimer);
    if (elapsedTimer) clearInterval(elapsedTimer);
});
</script>

<style scoped>
.message-bubble__avatar {
    flex-shrink: 0;
    width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-top: 3px;
}

.message-bubble__action {
    opacity: 0;
    transition: opacity 0.15s ease-out;
}

.group\/message:hover .message-bubble__action,
.message-bubble__action:focus-visible {
    opacity: 1;
}

.message-bubble__time {
    font-size: 11px;
    color: var(--el-text-color-secondary);
}
</style>
