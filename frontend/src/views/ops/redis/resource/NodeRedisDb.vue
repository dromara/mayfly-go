<template>
    <BaseTreeNode v-bind="$attrs">
        <template #prefix>
            <SvgIcon name="Coin" color="#67c23a" />
        </template>
        <template #suffix="{ node, data }">
            <span>{{ data.params.keys }}</span>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>
