<template>
    <div class="terminal-tab-label flex items-center gap-1.5">
        <!-- 连接状态指示器 -->
        <span
            class="w-2 h-2 rounded-full shrink-0 transition-all duration-300"
            :class="statusClasses"
        ></span>
        <!-- 终端图标 -->
        <SvgIcon v-if="icon" :name="icon.name" :color="icon.color" class="text-xs shrink-0" />
        <!-- 终端名称（从外层 tab.name 获取） -->
        <span class="max-w-[120px] overflow-hidden text-ellipsis" :title="tabName">{{ tabName }}</span>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import SvgIcon from '@/components/svg-icon/index.vue';

const props = withDefaults(
    defineProps<{
        tabName?: string;  // 从外层 tab.name 传入
        icon?: { name: string; color?: string };
        status?: 'connected' | 'disconnected' | 'connecting' | 'error' | string;
    }>(),
    {
        tabName: '终端',
        status: 'disconnected',
    }
);

// 计算状态样式
const statusClasses = computed(() => {
    switch (props.status) {
        case 'connected':
            return 'bg-green-500';
        case 'connecting':
            return 'bg-yellow-500 animate-pulse';
        case 'error':
            return 'bg-red-500 animate-pulse';
        case 'disconnected':
        default:
            return 'bg-red-500 animate-pulse';
    }
});
</script>
