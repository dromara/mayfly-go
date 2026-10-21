<template>
    <div class="chat-input" :class="{ 'chat-input--loading': loading }">
        <!-- 输入容器（Composer 卡片：限宽 max-w-3xl 居中）；点击空白聚焦编辑器，排除按钮/附件区；
             引用芯片直接内联在编辑器内展示 -->
        <div
            class="chat-input__wrapper mx-auto max-w-3xl rounded-2xl border border-border bg-background transition-colors focus-within:ring-2 focus-within:ring-ring/50"
            @click="onWrapperClick"
        >
            <!-- 已选附件预览 -->
            <AttachmentGroup v-if="pendingAttachments.length > 0" class="chat-input__attachments flex-wrap">
                <AttachmentItem v-for="(att, i) in pendingAttachments" :key="i" :attachment="att" removable @remove="removeAttachment(i)" />
            </AttachmentGroup>

            <!-- 编辑器（内核见 usePromptEditor：TipTap + 芯片 + 触发器） -->
            <div ref="editorContainerRef" class="chat-input__editor" />

            <!-- 工具条 -->
            <div class="chat-input__actions">
                <div class="chat-input__actions-left">
                    <Button
                        variant="outline"
                        size="icon"
                        class="rounded-full text-muted-foreground hover:text-foreground"
                        :title="t('ai.attach.add')"
                        :disabled="shouldQueue"
                        @click.stop="pickFiles"
                    >
                        <PaperclipIcon />
                    </Button>
                    <input ref="fileInputRef" type="file" multiple class="hidden" :accept="ATTACHMENT_ACCEPT" @change="onFilesPicked" />
                    <TriggerHints v-if="!hintGone" class="chat-input__hint" :class="{ 'chat-input__hint--faded': !hintVisible }" />
                </div>
                <div class="chat-input__actions-right">
                    <Button
                        v-if="loading"
                        variant="outline"
                        size="icon"
                        class="rounded-full active:scale-95 transition-transform"
                        :title="t('ai.chat.stop')"
                        @click.stop="$emit('cancel')"
                    >
                        <SquareIcon class="size-3.5 fill-current" />
                    </Button>
                    <Button
                        v-else
                        size="icon"
                        class="rounded-full active:scale-95 transition-transform"
                        :disabled="!canSend || submitting"
                        :title="shouldQueue ? t('ai.chat.addToQueue') : t('ai.chat.send')"
                        @click.stop="onSubmit"
                    >
                        <ArrowUpIcon />
                    </Button>
                </div>
            </div>
        </div>

        <!-- 触发器浮层（技能菜单 / 资源树面板），与表单里的提示词编辑器共用同一实现 -->
        <TriggerOverlays :prompt="prompt" />
    </div>
</template>

<script setup lang="ts">
/**
 * ChatInput - 聊天输入框（Composer）
 *
 * 职责边界：本组件只负责聊天专属交互——附件选择/粘贴转附件、发送与停止按钮、入队提示、
 * kbd 提示淡出。文档模型、芯片节点、触发器菜单与段序列化都在 usePromptEditor /
 * TriggerOverlays（表单里的 PromptEditor 用的是同一套内核），避免两处各养一份编辑器装配。
 */
import { ArrowUpIcon, PaperclipIcon, SquareIcon } from '@lucide/vue';
import { computed, nextTick, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';
import { Button } from '@/components/ui/button';
import { AttachmentGroup } from '@/components/ui/attachment';
import AttachmentItem from '../message/AttachmentItem.vue';
import { ATTACHMENT_ACCEPT, isSupportedAttachment, readChatAttachment, uploadChatAttachments } from './attachments';
import type { ChipSegment } from './chipRegistry';
import { buildSubmitData } from './submitData';
import TriggerOverlays from './TriggerOverlays.vue';
import TriggerHints from './TriggerHints.vue';
import { placeholderWithHints } from './useTriggerHints';
import { usePromptEditor } from './usePromptEditor';
import type { ChatInputSubmitData, SkillItem } from './types';
import type { MessageAttachment } from '../protocol/types';

const props = withDefaults(
    defineProps<{
        placeholder?: string;
        loading?: boolean;
        autoFocus?: boolean;
        /** 可用的技能/触发项列表 */
        skills?: SkillItem[];
        /** Agent 未完全空闲（回复中/待中断）时应入队而非直接发送（shouldQueue） */
        shouldQueue?: boolean;
    }>(),
    {
        placeholder: '',
        loading: false,
        autoFocus: false,
        skills: () => [],
        shouldQueue: false,
    }
);

const emit = defineEmits<{
    (e: 'submit', data: ChatInputSubmitData): void;
    (e: 'cancel'): void;
    (e: 'queue', data: ChatInputSubmitData): void;
}>();

const { t } = useI18n();

/** 占位符：入队态提示新消息将加入队列；正常态由触发器注册表拼接（组装单点见 useTriggerHints，
 *  与表单里的提示词编辑器共用同一实现） */
const currentPlaceholder = computed(() => {
    if (props.shouldQueue) return t('ai.chat.queueMessage');
    if (props.placeholder) return props.placeholder;
    return placeholderWithHints('ai.chat.inputPlaceholder');
});

// 附件（聊天专属）
const pendingAttachments = ref<MessageAttachment[]>([]);
const fileInputRef = ref<HTMLInputElement | null>(null);

/** kbd 提示可见性：首次成功发送后淡出（localStorage 记忆，不再常驻占行） */
const HINT_USED_KEY = 'ai.chat.hintUsed';
const hintVisible = ref(!localStorage.getItem(HINT_USED_KEY));
const hintGone = ref(!hintVisible.value);
const dismissHint = () => {
    if (!hintVisible.value) return;
    hintVisible.value = false;
    try {
        localStorage.setItem(HINT_USED_KEY, '1');
    } catch {
        // localStorage 不可用时忽略：提示仅本会话内淡出
    }
    // 淡出动画结束后移出 DOM，回收占位宽度
    setTimeout(() => (hintGone.value = true), 350);
};

/**
 * 提交重入守卫：附件上传是异步的，而 clear() 在最后一行——期间 canSend 仍为 true、
 * 文本未清，连按两次 Enter（或连点两次发送）会并发跑两次上传（服务端留下重复文件，
 * 两次各自拿到新附件对象，幂等去重也拦不住）并 emit 两次 submit，最终是两条同样的消息与两个 turn。
 */
const submitting = ref(false);

const onSubmit = async () => {
    if (!canSend.value || submitting.value) return;
    submitting.value = true;
    try {
        dismissHint();

        const text = prompt.text.value.trim();
        const segments = prompt.segments();

        // 附件在入队/发送前统一上传（内容落 t_sys_file，消息体只随 fileKey 轻量引用）；
        // 失败中止且保留输入态（编辑器内容与附件可重试），入队与直接发送共享同一前置链路
        let attachments = pendingAttachments.value;
        if (attachments.length > 0) {
            try {
                attachments = await uploadChatAttachments(attachments);
                pendingAttachments.value = attachments;
            } catch {
                Msg.error('ai.attach.uploadFailed');
                return;
            }
        }
        const data = buildSubmitData(text, segments, attachments);

        // shouldQueue 时（回复中/待中断）加入队列而非直接发送
        if (props.shouldQueue) {
            emit('queue', data);
        } else {
            emit('submit', data);
        }
        clear();
    } finally {
        submitting.value = false;
    }
};

// 编辑器内核：Enter 提交、Esc 停止、粘贴文件转附件
const prompt = usePromptEditor({
    placeholder: () => currentPlaceholder.value,
    skills: () => props.skills ?? [],
    autoFocus: () => props.autoFocus,
    onSubmit,
    onEscape: () => {
        if (!props.loading) return false;
        emit('cancel');
        return true;
    },
    handlePaste: (event) => {
        const files = filesFromClipboard(event.clipboardData);
        if (files.length === 0) return false;
        addAttachmentFiles(files);
        return true;
    },
});

// TipTap 挂载点：必须绑到 setup 作用域里的同名标识符。
// 写成 ref="prompt.editorContainer" 不会解析路径，容器 ref 恒为 null，
// usePromptEditor 的 onMounted 会直接 return —— 整个聊天输入框不渲染（实测踩过）
const editorContainerRef = prompt.editorContainer;

const canSend = computed(() => {
    return prompt.text.value.trim().length > 0 || prompt.chips.value.length > 0 || pendingAttachments.value.length > 0;
});

// ========== 附件选择（三分流：图片 dataURL / 文本内联 / 其它不支持） ==========

const pickFiles = () => {
    fileInputRef.value?.click();
};

const onFilesPicked = (e: Event) => {
    const input = e.target as HTMLInputElement;
    const files = Array.from(input.files || []);
    input.value = '';
    addAttachmentFiles(files);
};

/** 从剪贴板提取文件（files 为空时回退遍历 items，兼容部分浏览器截图粘贴） */
const filesFromClipboard = (dt: DataTransfer | null): File[] => {
    if (dt?.files?.length) return Array.from(dt.files);
    const out: File[] = [];
    for (const item of dt?.items ?? []) {
        if (item.kind === 'file') {
            const f = item.getAsFile();
            if (f) out.push(f);
        }
    }
    return out;
};

/** 批量添加附件（文件选择与粘贴共用）：校验走 isSupportedAttachment（大小/类型限制的单点），读取为消息附件 */
const addAttachmentFiles = async (files: File[]) => {
    for (const file of files) {
        if (!isSupportedAttachment(file)) {
            Msg.warning('ai.attach.tooLarge', { name: file.name });
            continue;
        }
        try {
            pendingAttachments.value.push(await readChatAttachment(file));
        } catch {
            Msg.error('ai.attach.readFailed', { name: file.name });
        }
    }
};

const removeAttachment = (index: number) => {
    pendingAttachments.value.splice(index, 1);
};

/** 点击卡片空白处聚焦编辑器（编辑区内交给 ProseMirror 定位光标，按钮/附件区不干预） */
const onWrapperClick = (e: MouseEvent) => {
    const target = e.target as HTMLElement;
    if (target.closest('.ProseMirror, button, .chat-input__attachments')) return;
    prompt.focus();
};

/** 清空输入（编辑器内容与附件一并清） */
const clear = () => {
    prompt.clear();
    pendingAttachments.value = [];
};

/**
 * 回填内容（队列编辑/追问草稿/外部填充）；attachments 为队列消息携带的已上传附件，
 * segments 为带芯片的段序列（有则按段还原，芯片可见可删）。
 * 返回 false 表示编辑器尚未就绪，调用方可重试（先清后填会在未就绪时把草稿吞掉）
 */
const setValue = (content: string, attachments?: MessageAttachment[], segments?: ChipSegment[]): boolean => {
    // 编辑器尚未创建（组件挂载前的跳页草稿回填）：必须返回 false 让调用方按帧重试。
    // 内核的 setText/setSegments 在未就绪时是静默 no-op，这里直接返回 true 会让调用方
    // 在内容实际没落地的情况下清空草稿信箱（正是本函数注释声明要防的「吞草稿」）
    if (!prompt.ready()) {
        return false;
    }
    if (segments && segments.length > 0) {
        prompt.setSegments(segments);
    } else {
        prompt.setText(content);
    }
    if (attachments?.length) pendingAttachments.value = [...attachments];
    nextTick(() => prompt.focus('end'));
    return true;
};

defineExpose({ clear, focus: () => prompt.focus('end'), setValue });
</script>

<style scoped>
.chat-input {
    width: 100%;
    padding: 0 24px 20px;
}

/* 输入容器（Composer 卡片） */
.chat-input__wrapper {
    display: flex;
    flex-direction: column;
    cursor: text;
}

/* 已选附件预览 */
.chat-input__attachments {
    display: flex;
    gap: 8px;
    padding: 12px 12px 0;
}

/* 编辑器区域 */
.chat-input__editor {
    min-height: 48px;
    max-height: 200px;
    overflow-y: auto;
    padding: 12px 14px;
    font-size: 14px;
    line-height: 1.6;
}

/* TipTap 占位符 */
.chat-input__editor :deep(.tiptap p.is-editor-empty:first-child::before) {
    content: attr(data-placeholder);
    float: left;
    color: var(--el-text-color-placeholder);
    pointer-events: none;
    height: 0;
}

/* 工具条 */
.chat-input__actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 12px 10px;
}

.chat-input__actions-left {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
}

.chat-input__actions-right {
    display: flex;
    align-items: center;
}

/* 只负责「首次发送后淡出」的过渡：提示条本身的布局/字号/颜色/kbd 均在 TriggerHints 内 */
.chat-input__hint {
    transition: opacity 0.3s ease-out;
}

.chat-input__hint--faded {
    opacity: 0;
    pointer-events: none;
}

.chat-input--loading .chat-input__editor {
    opacity: 0.7;
}
</style>
