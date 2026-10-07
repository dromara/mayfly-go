<template>
    <div class="ops-el-tabs-bar">
        <!-- 仅用 el-tabs 渲染标签条（header），内容区由页面自管（v-show），与自研实现保持同一布局语义 -->
        <el-tabs class="ops-el-tabs" :model-value="modelValue" type="card" @tab-change="onTabChange" @tab-remove="onTabRemove">
            <el-tab-pane v-for="tab in tabs" :key="tab.key" :name="tab.key" :closable="tab.closable !== false">
                <template #label>
                    <span
                        class="ops-el-tab-label"
                        :title="tab.title || tab.label"
                        @contextmenu.prevent="onContextmenu($event, tab)"
                        @mousedown.middle.prevent
                        @auxclick.middle.stop="onMiddleClose(tab)"
                    >
                        <slot name="prefix" :tab="tab" />
                        <slot name="label" :tab="tab">{{ tab.label }}</slot>
                    </span>
                </template>
            </el-tab-pane>
        </el-tabs>

        <div v-if="$slots.extra" class="ops-el-tabs-extra">
            <slot name="extra" />
        </div>

        <!-- 内置右键菜单：关闭当前 / 关闭其它（与自研实现共享 useTabContextmenu） -->
        <Contextmenu ref="contextmenuRef" :dropdown="dropdown" :items="items" />
    </div>
</template>

<script setup lang="ts">
import { Contextmenu } from '@/components/contextmenu';
import type { TabBarEmits, TabBarProps, TabItem } from './types';
import { useTabContextmenu } from './useTabContextmenu';

/**
 * Element Plus el-tabs 适配器：把 el-tabs 的事件/插槽翻译成统一契约（TabBarProps/TabBarEmits/同名插槽）。
 * 存在意义是让「换回 Element Plus 原生 tab」成为 index.ts 里一个常量的事，业务页面零改动。
 */
const props = defineProps<TabBarProps>();
const emit = defineEmits<TabBarEmits>();

// el-tabs 回调的 key 为 string | number，统一归一成契约的 string
const onTabChange = (key: string | number) => emit('update:modelValue', String(key));
const onTabRemove = (key: string | number) => emit('close', String(key));

// 内置右键菜单（关闭当前 / 其它 / 右侧 / 全部）：与自研实现共享同一 composable，行为一致
const { contextmenuRef, dropdown, items, openTabContextmenu } = useTabContextmenu({
    getTabs: () => props.tabs,
    getActiveKey: () => props.modelValue,
    close: (key) => emit('close', key),
    activate: (key) => emit('update:modelValue', key),
});

const onContextmenu = (event: MouseEvent, tab: TabItem) => {
    openTabContextmenu(event, tab);
    emit('contextmenu', event, tab);
};

// 中键点击关闭：与自研实现保持同一交互口径
const onMiddleClose = (tab: TabItem) => {
    if (tab.closable !== false) {
        emit('close', tab.key);
    }
};
</script>

<style scoped>
.ops-el-tabs-bar {
    display: flex;
    align-items: center;
}

.ops-el-tabs {
    flex: 1;
    min-width: 0;
}

/* 只保留标签条：隐藏 el-tabs 自带的空内容区与头部外边距，使其高度/占位与自研实现一致 */
.ops-el-tabs :deep(.el-tabs__content) {
    display: none;
}
.ops-el-tabs :deep(.el-tabs__header) {
    margin: 0;
}

.ops-el-tab-label {
    display: inline-flex;
    align-items: center;
    gap: 4px;
}

.ops-el-tabs-extra {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 0 8px;
    flex-shrink: 0;
}
</style>
