<template>
    <div class="prompt-editor" :class="{ 'prompt-editor--disabled': disabled }" @click="onBodyClick">
        <!-- 编辑区（TipTap 挂载点）：区内光标由 ProseMirror 定位，只有卡片其余空白的点击才聚焦 -->
        <div ref="containerRef" class="prompt-editor__body" :style="bodyStyle"></div>

        <!-- 引用快捷键提示：数据与样式均收口在 TriggerHints（注册表驱动），新增触发类型零改动自动回显 -->
        <TriggerHints class="prompt-editor__hints" />

        <TriggerOverlays :prompt="prompt" />
    </div>
</template>

<script setup lang="ts" name="PromptEditor">
/**
 * PromptEditor - 表单里的提示词编辑器（可引用技能 / 资源）
 *
 * 与聊天输入框共用同一编辑器内核（usePromptEditor）与浮层（TriggerOverlays），差别只在交互：
 * 没有附件、没有发送/停止按钮、**Enter 是换行而不是提交**（表单里回车即提交是误操作温床）。
 *
 * 受控同步：文档变化时向上 emit text + segments；外部灌入（打开另一条任务、表单重置）时
 * 只有在「传入值 ≠ 我们刚 emit 的值」时才重建文档，否则每次输入都会触发一次 setContent，
 * 光标会跳回开头。
 */
import { computed, onMounted, watch } from 'vue';
import TriggerOverlays from './TriggerOverlays.vue';
import TriggerHints from './TriggerHints.vue';
import type { ChipSegment } from './chipRegistry';
import { usePromptEditor } from './usePromptEditor';
import { usePromptSkills } from './usePromptSkills';

const props = withDefaults(
    defineProps<{
        /** 指令文本（芯片以其注册表的文本形态内联在文中） */
        text?: string;
        /** 芯片引用段（含技能码与资源定位信息，与 text 同源） */
        segments?: ChipSegment[];
        placeholder?: string;
        disabled?: boolean;
        /**
         * 编辑区高度：>0 为固定高度（表单场景，布局不随输入行数上涨，超出内部滚动）；
         * 传 0 为自适应并用 maxHeight 封顶（消息气泡的原地编辑态，不能占掉半屏）
         */
        height?: number;
        /** 自适应模式下的最大高度（仅 height=0 时生效） */
        maxHeight?: number;
        /**
         * Enter 是否提交（emit submit）。默认 false：表单里 Enter 必须是换行；
         * 消息气泡的原地编辑态需要 Enter 即重发，所以做成开关而不是另写一套编辑器
         */
        submitOnEnter?: boolean;
    }>(),
    { text: '', segments: () => [], placeholder: '', disabled: false, height: 240, maxHeight: 240, submitOnEnter: false }
);

const emit = defineEmits<{
    (e: 'update:text', value: string): void;
    (e: 'update:segments', value: ChipSegment[]): void;
    /** Enter 提交（仅 submitOnEnter=true 时发出） */
    (e: 'submit'): void;
    /** Escape（仅 submitOnEnter=true 时消费并发出，否则留给外层抽屉关闭） */
    (e: 'escape'): void;
}>();

/** 编辑区尺寸：固定高度或自适应 + 上限 */
const bodyStyle = computed(() =>
    props.height > 0 ? { height: `${props.height}px` } : { minHeight: '1.6em', maxHeight: `${props.maxHeight}px` },
);

const { skills, load: loadSkills } = usePromptSkills();

// 最近一次向上抛出的内容：用于区分「外部灌入」与「自己刚 emit 的回声」
let lastEmittedText = props.text ?? '';
let lastEmittedSegmentsJson = JSON.stringify(props.segments ?? []);

/**
 * 把文档现状上抛给表单（text 与 segments 同源，缺一即断链）。
 *
 * 这里引用了下面才创建的 prompt：调用只发生在编辑器 onUpdate / props 变化时，
 * 那时 prompt 已初始化，不存在 TDZ 问题
 */
const emitChange = () => {
    const text = prompt.text.value;
    const segments = prompt.segments();
    lastEmittedText = text;
    lastEmittedSegmentsJson = JSON.stringify(segments);
    emit('update:text', text);
    emit('update:segments', segments);
};

const prompt = usePromptEditor({
    placeholder: () => props.placeholder,
    skills: () => skills.value,
    disabled: () => !!props.disabled,
    // 键入与插芯片都必须上抛：漏接这一项时编辑器看着一切正常，但表单提交的仍是打开时的旧值，
    // 用户加的引用会在保存后凭空消失（实测踩过）
    onChange: emitChange,
    // Enter 提交由 submitOnEnter 控制：同一个内核既服务表单（Enter 换行）也服务气泡原地编辑（Enter 重发）
    onSubmit: () => emit('submit'),
    submitOnEnter: () => props.submitOnEnter,
    // Esc 只在气泡编辑态消费（取消编辑）；表单里 Esc 要留给外层抽屉，不能被我拦下
    onEscape: () => {
        if (!props.submitOnEnter) {
            return false;
        }
        emit('escape');
        return true;
    },
});

/** 把编辑器内容灌回文档：有段序列时按段还原（芯片可见可删），否则按纯文本处理 */
const applyExternal = () => {
    const incoming = props.segments ?? [];
    if (incoming.length > 0) {
        prompt.setSegments(incoming);
        return;
    }
    prompt.setText(props.text ?? '');
};

watch(
    () => [props.text, props.segments] as const,
    () => {
        const incomingJson = JSON.stringify(props.segments ?? []);
        if ((props.text ?? '') === lastEmittedText && incomingJson === lastEmittedSegmentsJson) {
            return; // 自己刚抛出去的回声
        }
        applyExternal();
        // 外部灌入后同步一次，保证 lastEmitted* 反映当前文档
        emitChange();
    }
);

const containerRef = prompt.editorContainer;

onMounted(() => {
    void loadSkills();
    // 编辑器在 usePromptEditor 的 onMounted 里创建（先于此处注册），有初始内容时回填一次
    if ((props.text ?? '') !== '' || (props.segments ?? []).length > 0) {
        applyExternal();
    }
});

const onBodyClick = (e: MouseEvent) => {
    const el = e.target as HTMLElement;
    // 点在编辑区内不干预：ProseMirror 自己会把光标定位到点击处，这里再 focus 会把它抢回末尾（实测缺陷）
    if (el.closest('.ProseMirror') || el.closest('.prompt-editor__hints')) return;
    // 点到卡片边框等编辑区以外的空白：聚焦但不改选区，保住「接着刚才的位置写」
    prompt.focus();
};

defineExpose({ focus: () => prompt.focus('end'), clear: () => prompt.clear() });
</script>

<style scoped>
.prompt-editor {
    position: relative;
    width: 100%;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    background: var(--el-bg-color);
    transition: border-color 0.2s;
}

.prompt-editor:focus-within {
    border-color: var(--el-color-primary);
}

.prompt-editor--disabled {
    opacity: 0.6;
    pointer-events: none;
}

.prompt-editor__body {
    padding: 8px 11px;
    font-size: 14px;
    line-height: 1.6;
    /* 高度由内联 style 固定（height prop），内容超出内部滚动 */
    overflow-y: auto;
    cursor: text;
}

/* TipTap 占位符（float:left 避免把空段落撑高，与聊天输入框同一处理） */
.prompt-editor__body :deep(.tiptap p.is-editor-empty:first-child::before) {
    content: attr(data-placeholder);
    float: left;
    color: var(--el-text-color-placeholder);
    pointer-events: none;
    height: 0;
}

/* 只保留容器职责（内边距与分隔线）：布局、字号、颜色与 kbd 外观均在 TriggerHints 内部 */
.prompt-editor__hints {
    padding: 2px 11px 4px;
    border-top: 1px dashed var(--el-border-color-lighter);
}
</style>
