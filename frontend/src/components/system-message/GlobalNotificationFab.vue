<template>
    <div 
        v-if="globalNotificationState.activeCount > 0" 
        class="fixed z-[2000]"
        :style="{ bottom: position.bottom + 'px', right: position.right + 'px' }"
    >
        <el-badge 
            :value="globalNotificationState.activeCount" 
            :max="99" 
            class="cursor-move"
            @mousedown="startDrag"
        >
            <el-button
                circle
                type="primary"
                class="w-[50px] h-[50px] text-xl shadow-lg transition-all duration-300"
                :class="{ 'hover:scale-110 hover:shadow-xl': !isDragging }"
                @click="toggleNotificationPanel"
            >
                <SvgIcon name="Bell" />
            </el-button>
        </el-badge>

        <!-- 展开的通知面板 -->
        <Transition name="slide-fade">
            <div
                v-if="isPanelVisible"
                class="absolute bottom-[60px] right-0 w-[420px] max-h-[500px] bg-white dark:bg-gray-800 rounded-lg shadow-2xl overflow-hidden z-[2001]"
            >
                <div class="flex justify-between items-center p-3 bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
                    <h3 class="m-0 text-base font-semibold text-gray-800 dark:text-gray-200">{{ $t('components.sysmsg.notifications.title') }}</h3>
                    <el-button size="small" text @click="isPanelVisible = false">
                        <SvgIcon name="Close" />
                    </el-button>
                </div>

                <el-scrollbar max-height="400px">
                    <div class="p-4">
                        <!-- 直接展示所有通知 -->
                        <div class="flex flex-col gap-2">
                            <div v-for="task in allTasks" :key="task.id" class="p-2 bg-gray-50 dark:bg-gray-900 rounded border border-gray-200 dark:border-gray-700">
                                <!-- 显示通知标题 -->
                                <div class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">{{ translateTitle(task.options.title) }}</div>
                                <!-- 直接渲染原有组件 -->
                                <component :is="task.component" v-bind="task.componentProps" />
                            </div>
                        </div>

                        <el-empty v-if="globalNotificationState.activeCount === 0" :description="$t('common.noData')" :image-size="80" />
                    </div>
                </el-scrollbar>
            </div>
        </Transition>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useDraggableFab } from '@/hooks/useDraggableFab';
import { activeNotifications, globalNotificationState } from './global-notification-manager';

const { t } = useI18n();

const isPanelVisible = ref(false);

// 拖拽与位置持久化收敛到通用 composable（与 AI 助手悬浮球共享同一实现）
const { position, isDragging, hasMoved, startDrag } = useDraggableFab({
    storageKey: 'global-notification-fab-position',
    defaultPosition: { bottom: 20, right: 20 },
});

// 所有任务列表
const allTasks = computed(() => {
    return Array.from(activeNotifications.values());
});

// 翻译title（支持i18n key和直接文本）
const translateTitle = (title: string): string => {
    // 如果包含点号，说明是i18n key，需要翻译
    if (title.includes('.')) {
        return t(title);
    }
    // 否则直接返回原文本
    return title;
};

const toggleNotificationPanel = () => {
    // 如果发生了拖拽移动，不触发点击事件
    if (hasMoved.value) {
        hasMoved.value = false;
        return;
    }
    isPanelVisible.value = !isPanelVisible.value;
};
</script>

<style scoped>
.slide-fade-enter-active {
    transition: all 0.3s ease;
}

.slide-fade-leave-active {
    transition: all 0.2s ease;
}

.slide-fade-enter-from,
.slide-fade-leave-to {
    transform: translateY(10px);
    opacity: 0;
}
</style>
