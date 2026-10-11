<template>
    <!--
        覆盖式查找条：常态只留右上角一个放大镜入口，展开后整条浮在树顶部（absolute，不占布局高度）。
        与常驻搜索框的区别就在这里——树是懒加载层级，查找是低频动作，不该为它常年扣掉一行可见层级。
    -->
    <button
        v-if="!visible"
        type="button"
        class="tree-find-trigger"
        :aria-label="$t('tag.treeFindTriggerTip', { shortcut: modKey })"
        :title="$t('tag.treeFindTriggerTip', { shortcut: modKey })"
        @click="emit('expand')"
    >
        <SvgIcon name="search" class="text-xs" />
    </button>

    <div v-else class="tree-find-bar" :class="{ 'is-no-match': !!query && countReady && total === 0 }">
        <SvgIcon name="search" class="shrink-0 text-xs text-(--el-text-color-secondary)" />

        <el-input ref="inputRef" size="small" :model-value="query" :placeholder="$t('tag.treeFindPlaceholder')" @update:model-value="onQueryChange" @keydown="onKeydown" />

        <!-- 生效的类型范围写在这条浮层里：入口 chip 让位时也不能留下一条看不见的过滤规则 -->
        <span v-if="scopeLabelKey" class="tree-find-scope" :title="$t('tag.treeTypeFilterTip')">{{ $t(scopeLabelKey) }}</span>

        <span v-if="query && countReady" class="tree-find-count shrink-0" :title="$t('tag.treeFindNavTip')">{{ countText }}</span>

        <button type="button" class="tree-find-close" :aria-label="$t('tag.treeFindClose')" :title="$t('tag.treeFindClose')" @click="emit('collapse')">
            <SvgIcon name="Close" class="text-[12px]" />
        </button>
    </div>
</template>

<script lang="ts" setup>
/**
 * 资源树查找条（纯展示 + 事件上抛，不持有过滤态）
 *
 * 命中集合与跳转由父组件经树容器的 matchedKeys 驱动：本组件不认识树数据，
 * 因此任何接了 TreeContainer 的面板都能直接复用。
 */
import { computed, useTemplateRef, watch } from 'vue';

import SvgIcon from '@/components/svg-icon/index.vue';

const props = defineProps<{
    /** 展开态：false 时只渲染悬浮入口 */
    visible: boolean;
    /** 当前查找词 */
    query: string;
    /** 命中总数（口径=当前可跳转的命中） */
    total: number;
    /** 当前定位序号（1 基；0 表示尚未跳转） */
    no: number;
    /** 生效的资源类型文案 i18n key；空表示未做类型筛选 */
    scopeLabelKey?: string;
    /** 计数是否已与引擎生效值对齐；false 时宁可不显示，也不报防抖窗口里的旧值/零命中 */
    countReady?: boolean;
}>();

const emit = defineEmits<{
    'update:query': [value: string];
    expand: [];
    collapse: [];
    /** 跳转方向：1 下一个、-1 上一个 */
    step: [delta: number];
}>();

const inputRef = useTemplateRef<{ focus: () => void }>('inputRef');

/** 快捷键修饰键按平台呈现：文案里写死 ⌘ 会让 Windows/Linux 用户按不出这个入口 */
const modKey = /^Mac/i.test(navigator.platform) ? '⌘' : 'Ctrl+';

const countText = computed(() => (props.no ? `${props.no}/${props.total}` : `${props.total}`));

const onQueryChange = (value: string) => emit('update:query', value);

const onKeydown = (event: KeyboardEvent) => {
    // 输入法组合态的 Enter/Escape 属于候选词操作：EP 在 isComposing 期间故意不 emit input，
    // 但 keydown 照常上抛——不拦下就是「中文还没选完词，树已经按旧词跳走了」
    if (event.isComposing || event.keyCode === 229) {
        return;
    }
    if (event.key === 'Enter') {
        emit('step', event.shiftKey ? -1 : 1);
        return;
    }
    if (event.key === 'Escape') {
        // 就地消费：阻止冒泡到页面级 Escape（面板全屏退出）被一次按键连带触发
        event.stopPropagation();
        emit('collapse');
    }
};

// 展开即抢焦点：入口点击/快捷键之后不该还要再点一下输入框。
// flush: 'post' 是必要的——pre 时机下输入框还没渲染，ref 仍为空，焦点会静默丢失
watch(
    () => props.visible,
    (visible) => {
        if (visible) {
            inputRef.value?.focus();
        }
    },
    { flush: 'post' }
);

/** 供页面级快捷键在查找条已展开时把焦点收回输入框（否则再按一次 ⌘F 看着像没反应） */
defineExpose({ focusInput: () => inputRef.value?.focus() });
</script>

<style lang="scss" scoped>
/*
  入口与查找条都 absolute 浮在树右上角，不给树留任何一行（面板高度寸土寸金）：
  代价是首行右角这 20px 会被入口压住——等价操作右键菜单也有，滚动一行即可让开。
*/
.tree-find-trigger {
    position: absolute;
    top: 2px;
    right: 8px;
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 20px;
    height: 20px;
    color: var(--el-text-color-secondary);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    opacity: 0.7;
    transition:
        opacity 0.15s ease,
        color 0.15s ease,
        border-color 0.15s ease;
}

.tree-find-trigger:hover {
    color: var(--el-color-primary);
    border-color: var(--el-color-primary-light-5);
    opacity: 1;
}

.tree-find-bar {
    position: absolute;
    top: 1px;
    right: 8px;
    left: 8px;
    z-index: 3;
    display: flex;
    align-items: center;
    gap: 4px;
    height: 22px;
    padding: 0 6px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-color-primary-light-5);
    border-radius: 12px;
    box-shadow: var(--el-box-shadow-light);
}

/* 零命中：整条转 danger——「已加载的层级里没有这个词」，比在 placeholder 里教育用户更直接 */
.tree-find-bar.is-no-match {
    border-color: var(--el-color-danger-light-5);
}

.tree-find-bar :deep(.el-input__wrapper) {
    height: 100%;
    padding: 0;
    background: transparent;
    box-shadow: none;
}

.tree-find-scope {
    max-width: 6em;
    padding: 0 5px;
    overflow: hidden;
    font-size: 11px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-radius: 8px;
    text-overflow: ellipsis;
    white-space: nowrap;
    flex-shrink: 0;
}

.tree-find-count {
    font-size: 11px;
    color: var(--el-text-color-secondary);
    font-variant-numeric: tabular-nums;
}

.is-no-match .tree-find-count {
    color: var(--el-color-danger);
}

.tree-find-close {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    color: var(--el-text-color-secondary);
    border-radius: 4px;
    transition:
        color 0.15s ease,
        background-color 0.15s ease;
}

.tree-find-close:hover {
    color: var(--el-text-color-primary);
    background: var(--el-fill-color);
}
</style>
