<template>
    <TreeNodeRow :data="data" :show-actions="showActions">
        <template #prefix>
            <SvgIcon name="Coin" color="#67c23a" />
        </template>
        <template #suffix="{ data }">
            <span>{{ data.params.keys }}</span>
        </template>
    </TreeNodeRow>
</template>

<script lang="ts" setup>
import TreeNodeRow from '@/views/ops/resource/tree/TreeNodeRow.vue';
import type { TreeNode } from '@/views/ops/resource/tree/types';

defineProps<{
    data: TreeNode;
    showActions?: boolean;
}>();
</script>

<style lang="scss"></style>
