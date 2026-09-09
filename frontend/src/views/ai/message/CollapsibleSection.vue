<template>
    <div class="collapsible-section">
        <button
            type="button"
            class="collapsible-section__trigger"
            :disabled="forceOpen"
            :aria-expanded="open"
            @click="toggle"
        >
            <slot name="icon" />
            <span class="collapsible-section__title">{{ title }}</span>
            <span v-if="summary" class="collapsible-section__summary">{{ summary }}</span>
            <slot name="extra" />
            <!-- chevron 在行尾（icon → title → meta → chevron） -->
            <ChevronRightIcon
                class="collapsible-section__arrow"
                :class="{ 'is-expanded': open }"
            />
        </button>
        <!-- 展开内容：grid 高度动画 + 内层 opacity 淡入（CollapsibleSection） -->
        <div class="collapsible-section__collapse" :class="{ 'is-expanded': open }">
            <div class="collapsible-section__clip">
                <div class="collapsible-section__fade">
                    <div class="collapsible-section__content">
                        <slot />
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
/**
 * CollapsibleSection - 通用折叠区块
 * CollapsibleSection：标题行（箭头 + 标题 + 摘要）+ 可展开内容
 *
 * 动画：grid-template-rows 0fr→1fr 高度过渡 + 内容 opacity 淡入，
 *       chevron 默认隐藏、hover 或展开时显现（渐进披露）。
 *       供 ProcessGroup / ToolCallPart / ReasoningPart 复用
 */
import { ChevronRightIcon } from '@lucide/vue';
import { computed, ref } from 'vue';

const props = defineProps<{
    /** 标题 */
    title: string;
    /** 标题右侧摘要文本 */
    summary?: string;
    /** 强制展开（如 pending 中断），禁用手动收起 */
    forceOpen?: boolean;
}>();

const innerOpen = ref(false);
const open = computed(() => props.forceOpen || innerOpen.value);

const toggle = () => {
    if (!props.forceOpen) innerOpen.value = !innerOpen.value;
};
</script>

<style scoped>
.collapsible-section__trigger {
    display: flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    margin: 0 -4px;
    padding: 5px 4px;
    border: none;
    background: transparent;
    font-family: inherit;
    font-size: 12px;
    text-align: left;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    user-select: none;
    border-radius: 6px;
    transition: background-color 0.15s ease-out;
}

.collapsible-section__trigger:hover {
    background-color: var(--el-fill-color-light);
    color: var(--el-text-color-regular);
}

.collapsible-section__trigger:disabled {
    cursor: default;
}

.collapsible-section__trigger:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: -1px;
}

/* 箭头渐进披露：默认隐藏，hover 或展开时显现 */
.collapsible-section__arrow {
    width: 12px;
    height: 12px;
    flex-shrink: 0;
    opacity: 0;
    transition:
        transform 0.2s ease-out,
        opacity 0.2s ease-out;
}

.collapsible-section__trigger:hover .collapsible-section__arrow,
.collapsible-section__arrow.is-expanded {
    opacity: 1;
}

.collapsible-section__arrow.is-expanded {
    transform: rotate(90deg);
}

/* 触屏设备无 hover：折叠态保持半透明箭头以提示可展开 */
@media (hover: none) {
    .collapsible-section__arrow {
        opacity: 0.5;
    }
}

.collapsible-section__title {
    /* 不伸缩占满：meta 紧跟标题内联；仅行内溢出时收缩截断 */
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.collapsible-section__summary {
    flex-shrink: 0;
    font-size: 11px;
    color: var(--el-text-color-placeholder);
}

/* grid 高度动画：0fr→1fr，内容层独立淡入 */
.collapsible-section__collapse {
    display: grid;
    grid-template-rows: 0fr;
    transition: grid-template-rows 0.25s cubic-bezier(0.22, 1, 0.36, 1);
}

.collapsible-section__collapse.is-expanded {
    grid-template-rows: 1fr;
}

.collapsible-section__clip {
    overflow: hidden;
    min-height: 0;
}

.collapsible-section__fade {
    opacity: 0;
    transition: opacity 0.2s ease-out;
}

.is-expanded .collapsible-section__fade {
    opacity: 1;
}

/* 内容面板：浅色圆角面取代旧的侧边色条。
   不设 max-height/overflow：滚动职责下沉到各内容面板（参数/结果/思考），
   避免双层嵌套滚动导致尾部被裁且滚动条不可感知（展开即应看全） */
.collapsible-section__content {
    margin-top: 3px;
    padding: 6px 10px;
    background: var(--el-fill-color-lighter);
    border-radius: 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

@media (prefers-reduced-motion: reduce) {
    .collapsible-section__trigger,
    .collapsible-section__arrow,
    .collapsible-section__collapse,
    .collapsible-section__fade {
        transition-duration: 0.01ms;
    }
}
</style>
