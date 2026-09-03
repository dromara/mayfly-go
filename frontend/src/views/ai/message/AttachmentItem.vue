<template>
    <Attachment size="sm">
        <template v-if="attachment.kind === 'image' && attachment.dataUrl">
            <AttachmentMedia variant="image">
                <img :src="attachment.dataUrl" :alt="attachment.name" class="size-full object-cover" />
            </AttachmentMedia>
            <AttachmentContent>
                <AttachmentTitle>{{ attachment.name }}</AttachmentTitle>
            </AttachmentContent>
        </template>
        <template v-else>
            <AttachmentMedia>
                <FileTextIcon v-if="attachment.kind === 'text'" />
                <FileIcon v-else />
            </AttachmentMedia>
            <AttachmentContent>
                <AttachmentTitle>{{ attachment.name }}</AttachmentTitle>
                <AttachmentDescription v-if="attachment.size != null">
                    {{ formatAttachmentSize(attachment.size) }}
                </AttachmentDescription>
            </AttachmentContent>
        </template>
        <AttachmentAction v-if="removable" class="attachment-item__remove" @click.stop="$emit('remove')">
            <XIcon />
        </AttachmentAction>
    </Attachment>
</template>

<script setup lang="ts">
/**
 * AttachmentItem - 共享附件卡片
 * ChatInput 待发送预览（removable）与 UserMessageBubble 历史回显共用的渲染单元
 */
import { FileIcon, FileTextIcon, XIcon } from '@lucide/vue';
import { formatAttachmentSize } from '../input/attachments';
import type { MessageAttachment } from '../protocol/types';
import {
    Attachment,
    AttachmentAction,
    AttachmentContent,
    AttachmentDescription,
    AttachmentMedia,
    AttachmentTitle,
} from '@/components/ui/attachment';

defineProps<{
    attachment: MessageAttachment;
    /** 是否显示移除按钮（仅输入框预览态） */
    removable?: boolean;
}>();

defineEmits<{
    (e: 'remove'): void;
}>();
</script>

<style scoped>
/* 移除按钮配色（原 chat-input__attachment-remove） */
.attachment-item__remove {
    color: var(--el-text-color-secondary);
}

.attachment-item__remove:hover {
    color: var(--el-color-danger);
}
</style>
