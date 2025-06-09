<template>
    <el-splitter @resize="handleResize">
        <el-splitter-panel :size="leftPaneSize + '%'" max="30%">
            <slot name="left"></slot>
        </el-splitter-panel>

        <el-splitter-panel>
            <slot name="right"></slot>
        </el-splitter-panel>
    </el-splitter>
</template>

<script lang="ts" setup>
import { useWindowSize } from '@vueuse/core';
import { computed } from 'vue';

const emit = defineEmits(['resize']);

const { width } = useWindowSize();

const leftPaneSize = computed(() => (width.value >= 1600 ? 20 : 24));

// 处理 resize 事件
const handleResize = (event: any) => {
    emit('resize', event);
};
</script>

<style lang="scss"></style>
