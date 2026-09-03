<template>
    <div class="chat-input" :class="{ 'chat-input--loading': loading, 'chat-input--disabled': disabled }">
        <!-- 输入容器（Composer 卡片：限宽 max-w-3xl 居中，对齐 tokhub ChatInput L401；
             点击空白聚焦编辑器，排除按钮/附件区）；引用芯片直接内联在编辑器内展示 -->
        <div
            class="chat-input__wrapper mx-auto max-w-3xl rounded-2xl border border-border bg-background transition-colors focus-within:ring-2 focus-within:ring-ring/50"
            @click="onWrapperClick"
        >
            <!-- 已选附件预览 -->
            <AttachmentGroup v-if="pendingAttachments.length > 0" class="chat-input__attachments flex-wrap">
                <AttachmentItem
                    v-for="(att, i) in pendingAttachments"
                    :key="i"
                    :attachment="att"
                    removable
                    @remove="removeAttachment(i)"
                />
            </AttachmentGroup>

            <!-- TipTap 编辑器 -->
            <div ref="editorContainer" class="chat-input__editor" />

            <!-- 工具条 -->
            <div class="chat-input__actions">
                <div class="chat-input__actions-left">
                    <Button
                        variant="outline"
                        size="icon"
                        class="rounded-full text-muted-foreground hover:text-foreground"
                        :title="t('ai.attach.add')"
                        :disabled="disabled || shouldQueue"
                        @click.stop="pickFiles"
                    >
                        <PaperclipIcon />
                    </Button>
                    <input
                        ref="fileInputRef"
                        type="file"
                        multiple
                        class="hidden"
                        :accept="ATTACHMENT_ACCEPT"
                        @change="onFilesPicked"
                    />
                    <div v-if="!hintGone" class="chat-input__hint" :class="{ 'chat-input__hint--faded': !hintVisible }">
                        <kbd>/</kbd> {{ t('ai.chat.skillTriggerPlaceholder') }}
                        <kbd>@</kbd> {{ t('ai.chat.resourceTriggerPlaceholder') }}
                    </div>
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
                        :disabled="!canSend"
                        :title="shouldQueue ? t('ai.chat.addToQueue') : t('ai.chat.send')"
                        @click.stop="onSubmit"
                    >
                        <ArrowUpIcon />
                    </Button>
                </div>
            </div>
        </div>

        <!-- 触发器建议菜单：技能列表（通用列表菜单）/ 资源树面板（面板模式由注册表 panelComponent 决定） -->
        <TriggerMenu
            v-if="trigger.visible && !trigger.isPanelMode"
            ref="triggerMenuRef"
            :visible="trigger.visible"
            :items="trigger.items"
            :selected-index="trigger.selectedIndex"
            :menu-style="trigger.menuStyle"
            @select="trigger.selectItem"
        />
        <ResourceTreePanel
            v-else-if="trigger.visible && trigger.isPanelMode"
            ref="resourceTreeRef"
            :visible="trigger.visible"
            :menu-style="trigger.menuStyle"
            :query="trigger.query"
            @select="onResourceTreeSelect"
            @close="trigger.dismiss"
        />
    </div>
</template>

<script setup lang="ts">
/**
 * ChatInput - 基于 TipTap 的富文本输入组件
 * 对齐 tokhub 的 ChatInput.tsx + ChipEditor.tsx；触发器子系统见 triggers/（对齐 tokhub triggers/）
 *
 * 功能：
 * - TipTap 编辑器（Document + Paragraph + Text + HardBreak + ChipNode）
 * - Enter 发送，Shift+Enter 换行
 * - `/` 触发技能选择菜单、`@` 触发资源树面板（注册表驱动，见 triggers/registry.ts）
 * - 芯片节点（原子内联节点，整体选中/删除）
 * - 发送时提取 segments（文本 + 芯片）
 */
import {
    ArrowUpIcon,
    PaperclipIcon,
    SquareIcon,
} from '@lucide/vue';
import Document from '@tiptap/extension-document';
import HardBreak from '@tiptap/extension-hard-break';
import Paragraph from '@tiptap/extension-paragraph';
import Placeholder from '@tiptap/extension-placeholder';
import Text from '@tiptap/extension-text';
import { Editor } from '@tiptap/vue-3';
import type { Editor as CoreEditor } from '@tiptap/core';
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '@/hooks/useI18n';
import { Button } from '@/components/ui/button';
import { AttachmentGroup } from '@/components/ui/attachment';
import AttachmentItem from '../message/AttachmentItem.vue';
import {
    ATTACHMENT_ACCEPT,
    MAX_ATTACHMENT_SIZE,
    needsUpload,
    readChatAttachment,
} from './attachments';
import { ChipNode } from './chipNode';
import { extractSegmentsFromNode } from './chipRegistry';
// 副作用导入：注册 skill/resource 芯片类型（序列化 extractSegment 依赖注册表，
// 漂浮在此处会被 tree-shake 导致芯片发送时被静默丢弃）
import './chipTypes';
// 副作用导入：注册 skill/resource 触发器（与芯片注册同理，见 triggers/index.ts）
import './triggers';
import type { ChatInputSubmitData, SkillItem } from './types';
import type { MessageAttachment } from '../protocol/types';
import { TriggerMenu, ResourceTreePanel, useTriggerState, selectResourceFromPanel } from './triggers';
import type { ResourceTreeSelectPayload } from './triggers/ResourceTreePanel.vue';

/** 活跃芯片（从编辑器文档同步，仅用于 canSend 判断；展示内联在编辑器内） */
interface ActiveChip {
    id: string;
    label: string;
    kind?: 'skill' | 'resource';
    /** 资源芯片的资源类型（machine / db） */
    resourceType?: string;
    icon?: string;
    color?: { bg: string; text: string; border: string };
}

const props = withDefaults(
    defineProps<{
        placeholder?: string;
        loading?: boolean;
        disabled?: boolean;
        autoFocus?: boolean;
        /** 可用的技能/触发项列表 */
        skills?: SkillItem[];
        /** Agent 未完全空闲（回复中/待中断）时应入队而非直接发送（对齐 tokhub shouldQueue） */
        shouldQueue?: boolean;
    }>(),
    {
        placeholder: '',
        loading: false,
        disabled: false,
        autoFocus: false,
        skills: () => [],
        shouldQueue: false,
    },
);

const emit = defineEmits<{
    (e: 'submit', data: ChatInputSubmitData): void;
    (e: 'cancel'): void;
    (e: 'queue', text: string): void;
}>();

const { t } = useI18n();
const editorContainer = ref<HTMLElement | null>(null);
const editor = ref<Editor | null>(null);

// 触发器子系统（检测/键盘/定位/选中聚合在 useTriggerState，见 triggers/）
const triggerMenuRef = ref<InstanceType<typeof TriggerMenu> | null>(null);
const resourceTreeRef = ref<InstanceType<typeof ResourceTreePanel> | null>(null);
const trigger = reactive(
    useTriggerState({
        getEditor: () => editor.value,
        getMenuEl: () => triggerMenuRef.value?.menuRef ?? resourceTreeRef.value?.panelRef,
        onSubmit: () => onSubmit(),
        skills: () => props.skills,
    }),
);

// 活跃芯片
const activeChips = ref<ActiveChip[]>([]);

/** 从编辑器文档提取芯片节点，重建活跃芯片列表（选中插入与退格删除均自动同步） */
const syncActiveChips = (e: CoreEditor) => {
    activeChips.value = extractSegmentsFromNode(e.state.doc)
        .filter((s) => s.type !== 'input_text')
        .map((s, i) => ({
            id: `${s.type}:${i}:${s.text}`,
            label: s.text,
            kind: s.type as 'skill' | 'resource',
            resourceType: s.extra.resourceType ? String(s.extra.resourceType) : undefined,
        }));
};

// 待发送附件
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

/** 跟踪编辑器文本变化（TipTap 不是 Vue 响应式的，需要手动追踪） */
const editorText = ref('');

const canSend = computed(() => {
    if (props.disabled) return false;
    return editorText.value.trim().length > 0 || activeChips.value.length > 0 || pendingAttachments.value.length > 0;
});

/** 占位符：入队态提示新消息将加入队列（对齐 tokhub queuePlaceholder） */
const currentPlaceholder = computed(() => {
    if (props.shouldQueue) return t('ai.chat.queueMessage');
    return props.placeholder || t('ai.chat.inputPlaceholder');
});

// ========== 编辑器生命周期 ==========

onMounted(() => {
    if (!editorContainer.value) return;

    editor.value = new Editor({
        element: editorContainer.value,
        extensions: [
            Document,
            Paragraph,
            Text,
            HardBreak.configure({
                keepMarks: true,
            }),
            ChipNode,
            Placeholder.configure({
                placeholder: () => currentPlaceholder.value,
            }),
        ],
        content: '',
        editable: !props.disabled,
        autofocus: props.autoFocus ? 'end' : false,
        editorProps: {
            handleKeyDown: (view, event) => {
                // 触发器菜单打开时，拦截键盘
                if (trigger.visible) {
                    return trigger.handleKeydown(event);
                }

                // Esc 停止流式（对齐 ChatGPT/Claude 键盘语义）
                if (event.key === 'Escape' && props.loading) {
                    event.preventDefault();
                    emit('cancel');
                    return true;
                }

                // Enter 发送（Shift+Enter 换行）
                if (event.key === 'Enter' && !event.shiftKey) {
                    event.preventDefault();
                    onSubmit();
                    return true;
                }
                return false;
            },
        },
        onUpdate: ({ editor: e }) => {
            // 同步文本到响应式 ref（驱动 canSend 更新）
            editorText.value = e.getText();
            // 从文档同步活跃芯片（选中/退格删除芯片均保持一致，驱动 canSend）
            syncActiveChips(e);
            // 触发词检测（菜单开合/切换/过滤，见 triggers/useTriggerState）
            trigger.checkState(e);
        },
    });
});

onBeforeUnmount(() => {
    editor.value?.destroy();
});

watch(
    () => props.disabled,
    (val) => {
        editor.value?.setEditable(!val);
    },
);

// ========== 触发器菜单逻辑 ==========
// 触发器注册表、触发词检测、浮层定位、键盘导航与选中插入
// 全部聚合在 triggers/ 子系统（对齐 tokhub triggers/ 目录），
// 新增触发类型只需 registerTrigger 注册，无需改动本组件分支

/** 资源树叶子选中：删除触发词后插入携带完整定位标识的资源芯片 */
const onResourceTreeSelect = (payload: ResourceTreeSelectPayload) => {
    trigger.selectPanelChip(payload, selectResourceFromPanel);
};

// ========== 附件选择（三分流：图片 dataURL / 文本内联 / 其它不支持） ==========

const pickFiles = () => {
    fileInputRef.value?.click();
};

const onFilesPicked = async (e: Event) => {
    const input = e.target as HTMLInputElement;
    const files = Array.from(input.files || []);
    input.value = '';

    for (const file of files) {
        if (file.size > MAX_ATTACHMENT_SIZE) {
            Msg.warning(t('ai.attach.tooLarge', { name: file.name }));
            continue;
        }
        if (needsUpload(file)) {
            Msg.warning(t('ai.attach.unsupported', { name: file.name }));
            continue;
        }
        try {
            pendingAttachments.value.push(await readChatAttachment(file));
        } catch {
            Msg.error(t('ai.attach.readFailed', { name: file.name }));
        }
    }
};

const removeAttachment = (index: number) => {
    pendingAttachments.value.splice(index, 1);
};

/** 点击卡片空白处聚焦编辑器（排除按钮/附件区） */
const onWrapperClick = (e: MouseEvent) => {
    const target = e.target as HTMLElement;
    if (target.closest('button, .chat-input__attachments')) return;
    focus();
};

// ========== 发送逻辑 ==========

const onSubmit = () => {
    if (!canSend.value || !editor.value) return;
    dismissHint();

    const e = editor.value;
    const text = e.getText().trim();
    const segments = extractSegmentsFromNode(e.state.doc);

    // shouldQueue 时（回复中/待中断）加入队列而非直接发送（对齐 tokhub：参考 Claude Code）
    if (props.shouldQueue) {
        if (text) {
            emit('queue', text);
            clear();
        }
        return;
    }

    emit('submit', {
        text,
        segments,
        attachments: pendingAttachments.value.length > 0 ? [...pendingAttachments.value] : undefined,
    });
    clear();
};

/** 清空输入 */
const clear = () => {
    editor.value?.commands.clearContent();
    activeChips.value = [];
    pendingAttachments.value = [];
    editorText.value = '';
    trigger.close();
};

/** 聚焦 */
const focus = () => {
    editor.value?.commands.focus('end');
};

/** 回填内容（队列编辑/外部填充，对齐 tokhub pendingValue 机制） */
const setValue = (content: string) => {
    if (!editor.value) return;
    editor.value.commands.setContent(content);
    editorText.value = content;
    nextTick(() => focus());
};

// 外部仅消费 setValue（队列编辑/回填）；clear/focus 供内部 submit 流程复用
defineExpose({ clear, focus, setValue });
</script>

<style scoped>
.chat-input {
    width: 100%;
    padding: 0 24px 20px;
}

/* Composer 卡片（卡片 token 在模板 Tailwind 类上，此处仅布局） */
.chat-input__wrapper {
    display: flex;
    flex-direction: column;
    cursor: text;
}

/* 已选附件预览 */
.chat-input__attachments {
    margin: 12px 12px 0;
}

/* 移除按钮配色已收敛到 AttachmentItem */

/* 编辑器区域 */
.chat-input__editor {
    min-height: 48px;
    max-height: 200px;
    overflow-y: auto;
    padding: 12px 16px;
    font-size: 14px;
    line-height: 1.6;
    outline: none;
    color: var(--el-text-color-primary);
}

.chat-input__editor :deep(.tiptap) {
    outline: none;
    min-height: 24px;
    color: var(--el-text-color-primary);
}

.chat-input__editor :deep(.tiptap p) {
    margin: 0;
}

.chat-input__editor :deep(.tiptap p.is-editor-empty:first-child::before) {
    content: attr(data-placeholder);
    float: left;
    color: var(--el-text-color-placeholder);
    pointer-events: none;
    height: 0;
}

.chat-input__editor :deep(.chat-chip) {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    border: 1px solid var(--el-color-primary-light-7);
    border-radius: 12px;
    padding: 1px 8px;
    font-size: 12px;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    cursor: default;
    user-select: none;
    vertical-align: baseline;
}

.chat-input__editor :deep(.chat-chip.ProseMirror-selectednode) {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 1px;
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

.chat-input__hint {
    font-size: 12px;
    color: var(--el-text-color-placeholder);
    transition: opacity 0.3s ease-out;
}

.chat-input__hint--faded {
    opacity: 0;
    pointer-events: none;
}

.chat-input__hint kbd {
    display: inline-block;
    padding: 1px 6px;
    font-size: 11px;
    background: var(--el-fill-color);
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    font-family: inherit;
}

.chat-input--loading .chat-input__editor {
    opacity: 0.7;
}

.chat-input--disabled .chat-input__wrapper {
    opacity: 0.5;
    pointer-events: none;
}
</style>

<style>
/* 芯片内联图标：TipTap 动态生成的芯片 DOM 无 scoped 属性，需全局样式；
   mask + currentColor 让图标跟随芯片文字色（lucide 线条图标 data URI） */
.chat-chip__icon {
    display: inline-block;
    width: 12px;
    height: 12px;
    flex-shrink: 0;
    background-color: currentColor;
    -webkit-mask: var(--chip-icon) no-repeat center / contain;
    mask: var(--chip-icon) no-repeat center / contain;
}

.chat-chip__icon[data-icon='zap'] {
    --chip-icon: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M13 2 3 14h9l-1 8 10-12h-9l1-8z'/%3E%3C/svg%3E");
}

.chat-chip__icon[data-icon='server'] {
    --chip-icon: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Crect x='2' y='2' width='20' height='8' rx='2'/%3E%3Crect x='2' y='14' width='20' height='8' rx='2'/%3E%3Cline x1='6' x2='6.01' y1='6' y2='6'/%3E%3Cline x1='6' x2='6.01' y1='18' y2='18'/%3E%3C/svg%3E");
}

.chat-chip__icon[data-icon='database'] {
    --chip-icon: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%23000' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cellipse cx='12' cy='5' rx='9' ry='3'/%3E%3Cpath d='M3 5v14a9 3 0 0 0 18 0V5'/%3E%3Cpath d='M3 12a9 3 0 0 0 18 0'/%3E%3C/svg%3E");
}
</style>
