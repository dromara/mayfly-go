<template>
    <Splitpanes class="default-theme" @resize="handleResize">
        <Pane :size="leftPaneSize" max-size="30">
            <slot name="left"></slot>
        </Pane>

        <Pane>
            <slot name="right"></slot>
        </Pane>
    </Splitpanes>
</template>

<script lang="ts" setup>
import { Pane, Splitpanes } from 'splitpanes';
import { useWindowSize } from '@vueuse/core';
import { computed } from 'vue';

const emit = defineEmits(['resize']);

const { width } = useWindowSize();

console.log(width);

const leftPaneSize = computed(() => (width.value >= 1600 ? 20 : 25));

// 处理 resize 事件
const handleResize = (event: any) => {
    emit('resize', event);
};
</script>

<style lang="scss"></style>
