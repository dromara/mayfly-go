<template>
    <Transition name="trigger-menu">
        <div
            v-if="visible"
            ref="menuRef"
            class="trigger-menu"
            :style="menuStyle"
        >
            <div v-if="items.length === 0" class="trigger-menu__empty">
                {{ t('ai.chat.noMatchingItems') }}
            </div>
            <div
                v-for="(item, index) in items"
                :key="item.id"
                class="trigger-menu__item"
                :class="{ 'is-selected': index === activeIndex }"
                @mousedown.prevent="selectItem(index)"
                @mouseenter="hoverIndex = index"
            >
                <!-- 资源菜单按类型分组：类型切换处插入分组标题（对齐资源树选择体验） -->
                <div
                    v-if="item.kind === 'resource' && (index === 0 || items[index - 1]?.resourceType !== item.resourceType)"
                    class="trigger-menu__group"
                >
                    {{ item.resourceType === 'machine' ? t('ai.chat.machineGroup') : t('ai.chat.dbGroup') }}
                </div>
                <div class="trigger-menu__item-row">
                    <div class="trigger-menu__item-icon">
                        <slot name="icon" :item="item">
                            <ServerIcon v-if="item.kind === 'resource' && item.resourceType === 'machine'" class="trigger-menu__zap" />
                            <DatabaseIcon v-else-if="item.kind === 'resource'" class="trigger-menu__zap" />
                            <ZapIcon v-else-if="!item.icon" class="trigger-menu__zap" />
                            <span v-else v-html="item.icon" />
                        </slot>
                    </div>
                    <div class="trigger-menu__item-content">
                        <div class="trigger-menu__item-label">{{ item.label }}</div>
                        <div v-if="item.description" class="trigger-menu__item-desc">{{ item.description }}</div>
                    </div>
                </div>
            </div>
        </div>
    </Transition>
</template>

<script setup lang="ts">
/**
 * TriggerMenu - 浮动建议菜单
 * 由 @floating-ui/dom 定位到光标附近
 */
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { DatabaseIcon, ServerIcon, ZapIcon } from '@lucide/vue';
import type { TriggerMenuItem } from './types';

const props = defineProps<{
    visible: boolean;
    items: TriggerMenuItem[];
    selectedIndex: number;
    menuStyle?: Record<string, string>;
}>();

const emit = defineEmits<{
    (e: 'select', index: number): void;
}>();

const { t } = useI18n();
const menuRef = ref<HTMLElement | null>(null);

/** 供父组件用 floating-ui 定位（floating 参照元素必须是菜单自身 DOM） */
defineExpose({ menuRef });

// 鼠标悬停优先于键盘选中（避免直接突变 prop）
const hoverIndex = ref<number | null>(null);
const activeIndex = computed(() => hoverIndex.value ?? props.selectedIndex);

const selectItem = (index: number) => {
    emit('select', index);
};
</script>

<style scoped>
.trigger-menu {
    position: fixed;
    z-index: 1000;
    min-width: 220px;
    max-width: 360px;
    max-height: 280px;
    overflow-y: auto;
    background: var(--el-bg-color-overlay);
    border: 1px solid var(--el-border-color-light);
    border-radius: 10px;
    box-shadow: var(--el-box-shadow-light);
    padding: 4px;
}

/* 入场/退场：轻微上移淡入（指数 ease-out） */
.trigger-menu-enter-active {
    transition:
        opacity 0.15s ease-out,
        transform 0.15s cubic-bezier(0.22, 1, 0.36, 1);
}

.trigger-menu-leave-active {
    transition:
        opacity 0.1s ease-out,
        transform 0.1s ease-out;
}

.trigger-menu-enter-from,
.trigger-menu-leave-to {
    opacity: 0;
    transform: translateY(-4px) scale(0.98);
}

@media (prefers-reduced-motion: reduce) {
    .trigger-menu-enter-active,
    .trigger-menu-leave-active {
        transition-duration: 0.01ms;
    }
}

.trigger-menu__empty {
    padding: 8px 12px;
    font-size: 13px;
    color: var(--el-text-color-placeholder);
    text-align: center;
}

/* 资源分组标题（机器/数据库） */
.trigger-menu__group {
    padding: 6px 12px 2px;
    font-size: 11px;
    color: var(--el-text-color-secondary);
    user-select: none;
}

.trigger-menu__item-row {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
}

.trigger-menu__item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    border-radius: 6px;
    cursor: pointer;
    transition: background-color 0.15s;
}

.trigger-menu__item:hover,
.trigger-menu__item.is-selected {
    background: var(--el-fill-color-light);
}

.trigger-menu__zap {
    width: 16px;
    height: 16px;
}

.trigger-menu__item-icon {
    flex-shrink: 0;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 16px;
}

.trigger-menu__item-content {
    flex: 1;
    min-width: 0;
}

.trigger-menu__item-label {
    font-size: 13px;
    font-weight: 500;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.trigger-menu__item-desc {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
</style>
