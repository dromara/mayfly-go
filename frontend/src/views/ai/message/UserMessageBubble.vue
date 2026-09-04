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
    <Bubble v-else variant="muted" align="end" class="max-w-full gap-1.5">
        <!-- 附件卡片区在正文上方：与输入区一致（待发送附件在编辑器上方），
             阅读顺序为「所指对象 → 对它的提问」；
             图片段转缩略图卡片 + 文本/文件附件（元数据随用户 TurnItem payload
             持久化，历史回显一致）；点击卡片开全屏查看器/展开预览 -->
        <AttachmentGroup v-if="cardAttachments.length" class="flex-wrap">
            <AttachmentItem
                v-for="(att, i) in cardAttachments"
                :key="i"
                :attachment="att"
                :expanded="att.kind === 'text' && att.name === expandedTextName"
                expandable-text
                @toggle="toggleTextPreview"
            />
        </AttachmentGroup>

        <!-- 文本附件原地展开预览：全宽等宽内容块（渐进披露，再点卡片收起）；
             内容本地内联（旧数据/发送前）或经文件服务 fileKey 拉取 -->
        <div v-if="expandedAttachment" class="user-message__text-preview">
            <pre>{{ expandedTextContent }}</pre>
        </div>

        <!-- 结构化 segments：芯片段渲染 InlineChip（对齐 tokhub InlineContentRenderer）；
             后端 buildUserSegments 保证用户消息恒有 segments（无芯片时为单条 input_text 段）；
             纯图片消息无文本段，不渲染空内容区 -->
        <BubbleContent v-if="hasBubbleText" class="text-[14px] leading-[1.6]">
            <template v-for="(seg, i) in segments" :key="i">
                <InlineChip v-if="seg.type === 'resource' || seg.type === 'skill'" :segment="seg" />
                <!-- 图片段不在此渲染：转为缩略图卡片进上方卡片区（与附件卡片同形态） -->
                <template v-else-if="seg.type !== 'image'">{{ seg.text }}</template>
            </template>
        </BubbleContent>
    </Bubble>
</template>

<script setup lang="ts">
/**
 * UserMessageBubble - 用户消息气泡
 * 对齐 tokhub UserMessageBubble：Bubble(muted) + 纯文本 + 附件列表
 * 编辑模式：原地 textarea 自动撑高，Enter 发送 / Esc 取消 / 失焦取消
 */
import { ArrowUpIcon } from '@lucide/vue';
import { computed, nextTick, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Button } from '@/components/ui/button';
import { getFileUrl } from '@/common/request';
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

// ==================== 附件卡片 ====================

/** 是否有非图片的内容段（纯图片消息不渲染空 BubbleContent） */
const hasBubbleText = computed(() => (props.segments ?? []).some((seg) => seg.type !== 'image'));

/**
 * 卡片列表：图片段转为缩略图卡片（与文本/文件附件同形态，点击开全屏查看器；
 * 图片以 image 段为唯一事实源，fileKey 经 /sys/files/{fileKey} 访问，
 * 无持久化 attachments 的消息也一致），后接文本/文件附件
 */
const cardAttachments = computed<MessageAttachment[]>(() => {
    // 本地图片预览源（乐观消息 attachments 携带 dataUrl）：按 name 匹配即时回显，
    // 避免刚上传完还等 /sys/files/{fileKey} 网络响应；历史消息无 dataUrl 自然回落
    const localPreviews = new Map<string, string>();
    for (const a of props.attachments ?? []) {
        if (a.kind === 'image' && a.dataUrl && a.name) localPreviews.set(a.name, a.dataUrl);
    }
    const imageCards: MessageAttachment[] = [];
    for (const seg of props.segments ?? []) {
        if (seg.type !== 'image') continue;
        const fileKey = typeof seg.extra?.fileKey === 'string' ? seg.extra.fileKey : '';
        if (!fileKey) continue;
        const name = typeof seg.extra?.name === 'string' && seg.extra.name ? seg.extra.name : t('ai.attach.imagePlaceholder');
        const dataUrl = localPreviews.get(name);
        imageCards.push({ name, kind: 'image', fileKey, ...(dataUrl ? { dataUrl } : {}) });
    }
    const others = (props.attachments ?? []).filter((a) => a.kind !== 'image');
    return [...imageCards, ...others];
});

/** 文本附件原地展开（按名称受控，再点同一卡片收起；纯前端态不持久化） */
const expandedTextName = ref<string | null>(null);
const toggleTextPreview = (name: string) => {
    expandedTextName.value = expandedTextName.value === name ? null : name;
};
const expandedAttachment = computed(() =>
    cardAttachments.value.find((a) => a.kind === 'text' && a.name === expandedTextName.value),
);

/**
 * 展开预览内容：统一经文件服务 /sys/files/{fileKey} 拉取
 * （发送前已完成上传，乐观消息与历史回显同一路径；按 fileKey 缓存，
 * 拉取失败提示且不缓存）
 */
const expandedTextContent = ref('');
const textContentCache = new Map<string, string>();
watch(
    expandedAttachment,
    async (att) => {
        const key = att?.fileKey ?? '';
        if (!key) {
            expandedTextContent.value = '';
            return;
        }
        const cached = textContentCache.get(key);
        if (cached != null) {
            expandedTextContent.value = cached;
            return;
        }
        try {
            const res = await fetch(getFileUrl(key));
            if (!res.ok) throw new Error(`fetch preview failed: ${res.status}`);
            const text = await res.text();
            textContentCache.set(key, text);
            // 竞态保护：仅当仍是当前展开项时写入
            if (expandedAttachment.value?.fileKey === key) expandedTextContent.value = text;
        } catch {
            if (expandedAttachment.value?.fileKey === key) expandedTextContent.value = t('ai.attach.previewFailed');
        }
    },
    { immediate: true },
);

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

/* 文本附件展开预览：全宽等宽内容块，高度上限滚动（长文件不撑爆消息流） */
.user-message__text-preview pre {
    width: 100%;
    max-height: 320px;
    margin: 0;
    overflow: auto;
    padding: 12px;
    border-radius: 8px;
    background: var(--el-fill-color-light);
    font-family: var(--el-font-family-mono, ui-monospace, monospace);
    font-size: 12px;
    line-height: 1.6;
    white-space: pre-wrap;
    word-break: break-all;
    color: var(--el-text-color-primary);
    animation: user-message-preview-in 0.15s ease-out;
}

@keyframes user-message-preview-in {
    from {
        opacity: 0;
        transform: translateY(-2px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@media (prefers-reduced-motion: reduce) {
    .user-message__text-preview pre {
        animation: none;
    }
}

.user-message-edit__actions {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 6px;
}
</style>
