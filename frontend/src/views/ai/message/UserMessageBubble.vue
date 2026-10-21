<template>
    <!-- 编辑模式：原地复用提示词编辑器（芯片能保持引用状态）。不能用 textarea：
         芯片是原子节点，textarea 里只能显示成字面 `@aliyun(root)`，用户看到的引用状态会消失，
         更糟的是重发时段序列被置空，模型收到的是没有任何定位信息的纯文本 -->
    <div v-if="isEditing" ref="editContainer" class="user-message-edit">
        <PromptEditor
            ref="editEditor"
            v-model:text="editValue"
            v-model:segments="editSegments"
            :placeholder="t('ai.chat.inputPlaceholder')"
            :height="0"
            :max-height="240"
            submit-on-enter
            @submit="sendEdit"
            @escape="$emit('cancel-edit')"
        />
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
            <Button size="sm" class="h-7 rounded-lg px-2.5 text-xs" :disabled="!editValue.trim()" @mousedown.prevent @click="sendEdit">
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

        <!-- 结构化 segments：芯片段渲染 InlineChip（InlineContentRenderer）；
             后端 buildUserSegments 保证用户消息恒有 segments（无芯片时为单条 input_text 段）；
             纯图片消息无文本段，不渲染空内容区 -->
        <BubbleContent v-if="hasBubbleText" class="text-[14px] leading-[1.6]">
            <template v-for="(seg, i) in segments" :key="i">
                <InlineChip v-if="getChipType(seg.type)" :segment="seg" />
                <!-- 图片段不在此渲染：转为缩略图卡片进上方卡片区（与附件卡片同形态） -->
                <template v-else-if="seg.type !== IMAGE_SEGMENT_TYPE">{{ seg.text }}</template>
            </template>
        </BubbleContent>
    </Bubble>
</template>

<script setup lang="ts">
/**
 * UserMessageBubble - 用户消息气泡
 * UserMessageBubble：Bubble(muted) + 纯文本 + 附件列表
 * 编辑模式：原地复用提示词编辑器（芯片可保形回显），Enter 发送 / Esc 取消 / 点击外部取消
 */
import { ArrowUpIcon } from '@lucide/vue';
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Button } from '@/components/ui/button';
import { getFileUrl } from '@/common/request';
import type { ContentSegment, MessageAttachment } from '../protocol/types';
import { IMAGE_SEGMENT_TYPE } from '../protocol/types';
import { getChipType, type ChipSegment } from '../input/chipRegistry';
import PromptEditor from '../input/PromptEditor.vue';
import InlineChip from './InlineChip.vue';
import AttachmentItem from './AttachmentItem.vue';
import { buildCardAttachments } from './attachmentCards';
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
    /** 重发时必须同时带上段序列：芯片的定位信息只在 extra 里，光有文本模型无法定位资源 */
    (e: 'edit-send', content: string, segments: ChipSegment[]): void;
    (e: 'cancel-edit'): void;
}>();

const { t } = useI18n();

// ==================== 附件卡片 ====================

/** 是否有非图片的内容段（纯图片消息不渲染空 BubbleContent） */
const hasBubbleText = computed(() => (props.segments ?? []).some((seg) => seg.type !== IMAGE_SEGMENT_TYPE));

/**
 * 卡片列表：image 段为主事实源 + 旧消息仅 attachments 的图片兜底 + 文本/文件附件。
 * 装配逻辑在 attachmentCards 纯函数里（含 fileKey 去重与本地预览匹配），可单测
 */
const cardAttachments = computed<MessageAttachment[]>(() =>
    buildCardAttachments(props.segments, props.attachments, t('ai.attach.imagePlaceholder'))
);

/** 文本附件原地展开（按名称受控，再点同一卡片收起；纯前端态不持久化） */
const expandedTextName = ref<string | null>(null);
const toggleTextPreview = (name: string) => {
    expandedTextName.value = expandedTextName.value === name ? null : name;
};
const expandedAttachment = computed(() => cardAttachments.value.find((a) => a.kind === 'text' && a.name === expandedTextName.value));

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
            // 此处故意用 fetch 而不是 Api.newGet：取回的是文件原始文本（res.text()），
            // 而 Api 请求层会按 `{code,msg,data}` 解 JSON 包体——任意文本文件不是包体形状，
            // 走 Api 会把内容解没（与上传不同：上传返回的确实是解包后的 fileKey）
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
    { immediate: true }
);

// ==================== 原地编辑 ====================

const editContainer = ref<HTMLElement | null>(null);
const editEditor = ref<InstanceType<typeof PromptEditor> | null>(null);
const editValue = ref('');
const editSegments = ref<ChipSegment[]>([]);

// 进入编辑态：把段序列灌回编辑器（芯片保持可点可删的引用状态），聚焦并把光标放到末尾
watch(
    () => props.isEditing,
    (editing) => {
        if (!editing) return;
        editValue.value = props.content;
        // ContentSegment.extra 可选而 ChipSegment 必填：补空对象，不让下游各自判空分叉
        editSegments.value = (props.segments ?? []).map((s) => ({ type: s.type, text: s.text ?? '', extra: s.extra ?? {} }));
        nextTick(() => {
            editEditor.value?.focus();
            editContainer.value?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        });
    }
);

const sendEdit = () => {
    const trimmed = editValue.value.trim();
    if (!trimmed) return;
    emit('edit-send', trimmed, editSegments.value);
};

/**
 * 点击编辑区外部即取消编辑。
 *
 * 不用 blur/focusout 判定：① 原实现把 `@blur` 挂在容器 div 上，而 blur 不冒泡，
 * 处理函数从不执行（功能静默失效）；② 就算换成 focusout，编辑器内部焦点移动
 * （点芯片、点按钮）也会被误判为「离开」。pointerdown 只由真实交互产生，
 * 内部点击用 closest 放行；捕获阶段监听不受内部 stopPropagation 影响
 */
const onDocumentPointerDown = (e: PointerEvent) => {
    if (!props.isEditing) return;
    const target = e.target as HTMLElement | null;
    if (target?.closest('.user-message-edit')) return;
    emit('cancel-edit');
};

document.addEventListener('pointerdown', onDocumentPointerDown, true);
onBeforeUnmount(() => document.removeEventListener('pointerdown', onDocumentPointerDown, true));
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
