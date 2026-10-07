<template>
    <div class="mf-tabs">
        <template v-if="tabs.length">
            <!-- 标签条：委托给可换皮实现（见 tab-bar.ts）；透传其全部事件与插槽 -->
            <TabBar
                :tabs="tabs"
                :model-value="modelValue"
                @update:model-value="onActivate"
                @close="emit('close', $event)"
                @contextmenu="(e, t) => emit('contextmenu', e, t)"
            >
                <template v-if="$slots.prefix" #prefix="scope">
                    <slot name="prefix" v-bind="scope" />
                </template>
                <template v-if="$slots.label" #label="scope">
                    <slot name="label" v-bind="scope" />
                </template>
                <template v-if="$slots.extra" #extra>
                    <slot name="extra" />
                </template>
            </TabBar>

            <!-- 内容面板：内置 v-for + v-show「常驻挂载」保活，切换只显隐、不销毁，页面不必再手写样板 -->
            <div class="mf-tabs__content" :class="contentClass">
                <div v-for="tab in tabs" :key="tab.key" v-show="tab.key === modelValue" class="mf-tabs__pane" :class="paneClass">
                    <slot :tab="tab" :active="tab.key === modelValue" />
                </div>
            </div>
        </template>

        <!-- 无标签时的空态（可选）：由使用方通过 #empty 提供 -->
        <div v-else-if="$slots.empty" class="mf-tabs__empty">
            <slot name="empty" />
        </div>
    </div>
</template>

<script setup lang="ts" generic="T extends TabItem">
import { TabBar } from './tab-bar';
import type { TabItem, TabsEmits } from './types';

/**
 * 通用标签页组件 = 标签条（可换皮）+ 内容面板（内置保活）。
 *
 * 相比直接用低层 TabBar，本组件替使用方托管内容区：内部对 tabs 做 v-for 并为每项包一层
 * v-show「常驻挂载」，切换标签只显隐、不销毁，编辑器 / 表格等重状态得以保留；页面只需通过
 * 默认作用域插槽按 tab 渲染内容，无需再手写 v-for + v-show 样板。
 *
 * 泛型 T 允许 tabs 携带业务字段（如 db 的 TabInfo），并原样透传给默认插槽，插槽内即得完整类型。
 * 标签条的右键关闭菜单（关闭当前 / 其它 / 右侧 / 全部）由 TabBar 内置，Tabs 透传其事件与插槽。
 * 切换副作用挂 @change（v-model 仅负责绑定），无需监听较难记的 update:model-value。
 */
const props = defineProps<{
    /** 标签列表；泛型 T 允许携带业务字段，会原样透传给默认插槽 */
    tabs: T[];
    /** 当前激活标签 key（v-model） */
    modelValue: string;
    /** 追加到内容容器的类（如 overflow-hidden、左右/底部内边距）；顶部内边距组件已给默认值，覆盖需 important（如 pt-0!） */
    contentClass?: string;
    /** 追加到每个面板的类（如 flex flex-col gap-2） */
    paneClass?: string;
}>();

const emit = defineEmits<TabsEmits>();

// v-model 只负责绑定；change 是「激活项确实切换」的通知，供页面挂切换副作用。
// 以 props.modelValue 作为旧值判定，顺带抹平两种皮肤对「点击当前激活 tab」的行为差异
const onActivate = (key: string) => {
    emit('update:modelValue', key);
    if (key !== props.modelValue) {
        emit('change', key);
    }
};
</script>

<style scoped>
.mf-tabs {
    display: flex;
    flex-direction: column;
    min-height: 0;
}

.mf-tabs__content {
    flex: 1;
    min-height: 0;
    /* 标签条与内容间的默认呼吸空间。本规则未分层、优先级高于工具类，
       页面如需不同值须用 content-class 传 important 工具类覆盖（如 pt-0! / pt-2!） */
    padding-top: 8px;
}

.mf-tabs__pane {
    height: 100%;
}

.mf-tabs__empty {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    min-height: 0;
}
</style>
