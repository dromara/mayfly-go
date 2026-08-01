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
import { computed, ref, onMounted, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { activeNotifications, globalNotificationState } from './global-notification-manager';

const { t } = useI18n();

const isPanelVisible = ref(false);

// 拖拽相关
const STORAGE_KEY = 'global-notification-fab-position';
const position = ref({ bottom: 20, right: 20 }); // 默认位置（对应 bottom-5 right-5）
const isDragging = ref(false);
const dragStart = ref({ x: 0, y: 0, initialBottom: 0, initialRight: 0 });
const hasMoved = ref(false); // 标记是否发生了移动

const startDrag = (event: MouseEvent) => {
    // 只在左键拖拽时生效
    if (event.button !== 0) return;
    
    isDragging.value = true;
    hasMoved.value = false;
    dragStart.value = {
        x: event.clientX,
        y: event.clientY,
        initialBottom: position.value.bottom,
        initialRight: position.value.right,
    };
    
    document.addEventListener('mousemove', onDrag);
    document.addEventListener('mouseup', stopDrag);
    
    // 防止拖拽时选中文本
    document.body.style.userSelect = 'none';
};

const onDrag = (event: MouseEvent) => {
    if (!isDragging.value) return;
    
    const deltaY = event.clientY - dragStart.value.y;
    const deltaX = event.clientX - dragStart.value.x;
    
    // 如果移动距离超过 3px，认为是拖拽而不是点击
    if (Math.abs(deltaX) > 3 || Math.abs(deltaY) > 3) {
        hasMoved.value = true;
    }
    
    // 更新位置（注意：鼠标向下移动时 bottom 应该减小）
    position.value.bottom = dragStart.value.initialBottom - deltaY;
    position.value.right = dragStart.value.initialRight - deltaX;
    
    // 获取窗口尺寸用于边界限制
    const windowHeight = window.innerHeight;
    const windowWidth = window.innerWidth;
    
    // 确保不会移出屏幕（留出至少 50px 保证按钮可见）
    if (position.value.bottom < 0) position.value.bottom = 0;
    if (position.value.right < 0) position.value.right = 0;
    if (position.value.bottom > windowHeight - 50) position.value.bottom = windowHeight - 50;
    if (position.value.right > windowWidth - 50) position.value.right = windowWidth - 50;
    
    // 如果发生了移动，阻止默认行为
    if (hasMoved.value) {
        event.preventDefault();
    }
};

const stopDrag = () => {
    isDragging.value = false;
    document.removeEventListener('mousemove', onDrag);
    document.removeEventListener('mouseup', stopDrag);
    
    // 恢复文本选择
    document.body.style.userSelect = '';
    
    // 保存位置到 localStorage
    savePosition();
};

// 组件卸载时清理事件监听
onUnmounted(() => {
    document.removeEventListener('mousemove', onDrag);
    document.removeEventListener('mouseup', stopDrag);
});

// 保存位置到 localStorage
const savePosition = () => {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(position.value));
    } catch (error) {
        console.warn('Failed to save notification fab position:', error);
    }
};

// 从 localStorage 加载位置
const loadPosition = () => {
    try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) {
            const parsed = JSON.parse(saved);
            // 验证数据有效性
            if (typeof parsed.bottom === 'number' && typeof parsed.right === 'number') {
                position.value = parsed;
            }
        }
    } catch (error) {
        console.warn('Failed to load notification fab position:', error);
    }
};

// 组件挂载时加载保存的位置
onMounted(() => {
    loadPosition();
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
