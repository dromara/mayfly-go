<template>
    <Attachment
        size="sm"
        class="attachment-item"
        :class="{ 'is-clickable': clickable }"
        @click="onClick"
    >
        <template v-if="attachment.kind === 'image' && imageSrc">
            <AttachmentMedia variant="image">
                <img :src="imageSrc" :alt="attachment.name" class="size-full object-cover" />
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
            <!-- 文本附件展开指示：chevron 旋转表达展开态 -->
            <ChevronDownIcon v-if="expandable" class="attachment-item__chevron" :class="{ 'is-expanded': expanded }" />
        </template>
        <AttachmentAction v-if="removable" class="attachment-item__remove" @click.stop="$emit('remove')">
            <XIcon />
        </AttachmentAction>
    </Attachment>

    <!-- 图片预览：ElImageViewer 内置缩放/旋转/全屏工具栏（滚轮/按钮均可缩放）。
         Teleport 到 body：脱离消息列表的层叠/裁剪环境，避免查看器被内联渲染；
         scale 0.6：contain 默认把大图放大到占满视口（观感全屏），初始降至 60%，
         滚轮/工具栏可再缩放与放大，小图在缩放后仍清晰可辨 -->
    <Teleport to="body">
        <ElImageViewer
            v-if="imageViewerVisible && imageSrc"
            :url-list="[imageSrc]"
            :scale="0.6"
            hide-on-click-modal
            @close="imageViewerVisible = false"
        />
    </Teleport>
</template>

<script setup lang="ts">
/**
 * AttachmentItem - 共享附件卡片
 * ChatInput 待发送预览（removable）与 UserMessageBubble 历史回显共用的渲染单元；
 * 点击行为：图片开 ElImageViewer（缩放/旋转/全屏），文本向父层发 toggle
 * （展开预览由父层全宽渲染，避免卡片窄宽限制内容可读性），file 类不可交互；
 * 图片源：本地 dataUrl（发送前即时预览）优先，否则文件服务 /sys/files/{fileKey}
 */
import { ChevronDownIcon, FileIcon, FileTextIcon, XIcon } from '@lucide/vue';
import { computed, ref } from 'vue';
import { ElImageViewer } from 'element-plus';
import { getFileUrl } from '@/common/request';
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

const props = defineProps<{
    attachment: MessageAttachment;
    /** 是否显示移除按钮（仅输入框预览态） */
    removable?: boolean;
    /** 文本附件展开态（父层受控） */
    expanded?: boolean;
    /** 是否启用文本展开交互（消息气泡内 true；输入框预览态不启用，内容即将随消息发送） */
    expandableText?: boolean;
}>();

const emit = defineEmits<{
    (e: 'remove'): void;
    /** 文本附件点击切换展开（父层渲染全宽预览块） */
    (e: 'toggle', name: string): void;
}>();

// ==================== 点击行为 ====================

const imageViewerVisible = ref(false);

/** 图片展示源：本地 dataUrl（发送前上传前的输入框预览）优先，否则经文件服务 fileKey 访问 */
const imageSrc = computed(() =>
    props.attachment.dataUrl || (props.attachment.fileKey ? getFileUrl(props.attachment.fileKey) : ''),
);

/** 有内容载体的附件才可交互（文本需父层启用展开交互，输入框预览态不可点） */
const clickable = computed(
    () => (props.attachment.kind === 'image' && !!imageSrc.value)
        || (props.attachment.kind === 'text' && props.expandableText === true && !!props.attachment.fileKey),
);

/** 文本附件可展开预览（需父层启用展开交互；内容统一经 fileKey 拉取） */
const expandable = computed(
    () => props.expandableText === true && props.attachment.kind === 'text' && !!props.attachment.fileKey,
);

const onClick = () => {
    if (props.attachment.kind === 'image' && imageSrc.value) {
        imageViewerVisible.value = true;
    } else if (expandable.value) {
        emit('toggle', props.attachment.name);
    }
};
</script>

<style scoped>
.attachment-item.is-clickable {
    cursor: pointer;
    transition: background-color 0.15s ease-out;
}

.attachment-item.is-clickable:hover {
    background-color: var(--el-fill-color-light);
}

/* 展开指示 chevron：150ms 旋转表达展开/收起 */
.attachment-item__chevron {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
    color: var(--el-text-color-secondary);
    transition: transform 0.15s ease-out;
}

.attachment-item__chevron.is-expanded {
    transform: rotate(180deg);
}

/* 移除按钮配色（原 chat-input__attachment-remove） */
.attachment-item__remove {
    color: var(--el-text-color-secondary);
}

.attachment-item__remove:hover {
    color: var(--el-color-danger);
}

@media (prefers-reduced-motion: reduce) {
    .attachment-item.is-clickable,
    .attachment-item__chevron {
        transition: none;
    }
}
</style>
