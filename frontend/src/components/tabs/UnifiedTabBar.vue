<template>
    <div ref="tablistRef" class="unified-tab-bar" role="tablist" @keydown="onKeydown">
        <div class="tab-scroll">
            <div
                v-for="tab in tabs"
                :key="tab.key"
                class="tab-item"
                :class="{ active: modelValue === tab.key }"
                role="tab"
                :aria-selected="modelValue === tab.key"
                :tabindex="modelValue === tab.key ? 0 : -1"
                @click="$emit('update:modelValue', tab.key)"
                @contextmenu.prevent="onContextmenu($event, tab)"
                @mousedown.middle.prevent
                @auxclick.middle.stop="onMiddleClose(tab)"
            >
                <slot name="prefix" :tab="tab" />
                <slot name="label" :tab="tab">
                    <span class="tab-label" :title="tab.title || tab.label">{{ tab.label }}</span>
                </slot>
                <button v-if="tab.closable !== false" class="tab-close" type="button" :aria-label="$t('common.close')" @click.stop="$emit('close', tab.key)">
                    <el-icon><Close /></el-icon>
                </button>
            </div>

            <!-- 占位条：把「非激活 tab 的底部分隔线」延伸到空白区，保证整条横线贯通；
                 激活 tab 自身底边透明，从而与下方内容区连通（VS Code 编辑器标签的连通感） -->
            <span class="tab-spacer" aria-hidden="true"></span>
        </div>

        <div v-if="$slots.extra" class="tab-extra">
            <slot name="extra" />
        </div>

        <!-- 内置右键菜单：关闭当前 / 关闭其它（共享 useTabContextmenu，页面无需接线） -->
        <Contextmenu ref="contextmenuRef" :dropdown="dropdown" :items="items" />
    </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue';
import { Close } from '@element-plus/icons-vue';
import { Contextmenu } from '@/components/contextmenu';
import type { TabBarEmits, TabBarProps, TabItem } from './types';
import { useTabContextmenu } from './useTabContextmenu';

const props = defineProps<TabBarProps>();
const emit = defineEmits<TabBarEmits>();

// 内置右键菜单（关闭当前 / 其它 / 右侧 / 全部）：所有标签条实现共享同一套逻辑，业务页面零接线即得
const { contextmenuRef, dropdown, items, openTabContextmenu } = useTabContextmenu({
    getTabs: () => props.tabs,
    getActiveKey: () => props.modelValue,
    close: (key) => emit('close', key),
    activate: (key) => emit('update:modelValue', key),
});

// 右键先用内置菜单兜底，再向外抛 contextmenu 事件供页面观察/扩展（默认无需接线）
const onContextmenu = (event: MouseEvent, tab: TabItem) => {
    openTabContextmenu(event, tab);
    emit('contextmenu', event, tab);
};

const tablistRef = ref<HTMLElement>();

// WAI-ARIA 标签页键盘导航：←/→ 循环切换、Home/End 首尾，并把焦点跟随到目标 tab。
// 仅当焦点落在某个 tab 上时响应，避免误捕 extra 区输入框等的方向键
const onKeydown = (event: KeyboardEvent) => {
    const navKeys = ['ArrowLeft', 'ArrowRight', 'Home', 'End'];
    if (!navKeys.includes(event.key) || !(event.target as HTMLElement).closest('[role="tab"]')) {
        return;
    }
    event.preventDefault();
    const total = props.tabs.length;
    if (!total) {
        return;
    }
    const current = props.tabs.findIndex((tab) => tab.key === props.modelValue);
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? total - 1 : event.key === 'ArrowLeft' ? (current - 1 + total) % total : (current + 1) % total;
    emit('update:modelValue', props.tabs[next].key);
    nextTick(() => tablistRef.value?.querySelectorAll<HTMLElement>('[role="tab"]')[next]?.focus());
};

// 中键点击关闭（对齐 VS Code / 浏览器标签习惯）；与 ✕ 同口径，非可关闭标签不响应
const onMiddleClose = (tab: TabItem) => {
    if (tab.closable !== false) {
        emit('close', tab.key);
    }
};
</script>

<style scoped>
/* 标签条底色：与内容区形成一层浅带，激活 tab 用内容底色「 punching 」出来 */
.unified-tab-bar {
    display: flex;
    align-items: stretch;
    background: var(--el-fill-color-light);
}

.tab-scroll {
    display: flex;
    align-items: stretch;
    flex: 1;
    min-width: 0;
    overflow-x: auto;
}

.tab-scroll::-webkit-scrollbar {
    height: 3px;
}
.tab-scroll::-webkit-scrollbar-thumb {
    background: var(--el-border-color-lighter);
    border-radius: 2px;
}

.tab-item {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    height: 30px;
    padding: 0 8px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    border-right: 1px solid var(--el-border-color-lighter);
    background: transparent;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1;
    white-space: nowrap;
    flex-shrink: 0;
    cursor: pointer;
    transition:
        background-color 0.15s ease,
        color 0.15s ease;
}

.tab-item:hover {
    background: var(--el-fill-color);
    color: var(--el-text-color-primary);
}

.tab-item:focus-visible {
    outline: 2px solid var(--el-color-primary-light-5);
    outline-offset: -2px;
}

/* 激活态：底色换成内容区底色并去掉自身底边，与下方内容连成一体；顶部一条主色高亮线 */
.tab-item.active {
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
    border-bottom-color: transparent;
}

.tab-item.active::before {
    content: '';
    position: absolute;
    inset: 0 0 auto 0;
    height: 1px;
    background: var(--el-color-primary);
}

/* 标签宽度随内容自适应，不做硬截断（避免遮住表名）；tab 过多时由 .tab-scroll 横向滚动兑底 */
.tab-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    line-height: normal;
}

.tab-close {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 14px;
    height: 14px;
    padding: 0;
    border: none;
    border-radius: 3px;
    background: transparent;
    color: var(--el-text-color-secondary);
    cursor: pointer;
    opacity: 0;
    transition:
        opacity 0.15s ease,
        background-color 0.15s ease,
        color 0.15s ease;
}

.tab-item:hover .tab-close,
.tab-item.active .tab-close {
    opacity: 1;
}

.tab-close:hover {
    background: var(--el-fill-color-dark);
    color: var(--el-text-color-primary);
}

.tab-spacer {
    flex: 1;
    min-width: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
}

.tab-extra {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 8px;
    flex-shrink: 0;
}
</style>
