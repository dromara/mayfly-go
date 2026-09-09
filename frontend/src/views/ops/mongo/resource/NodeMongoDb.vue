<template>
    <TreeNodeRow :data="data" :show-actions="showActions">
        <template #suffix="{ data }">
            <span>{{ formatByteSize(Number(data.params.size)) }}</span>
        </template>
    </TreeNodeRow>
</template>

<script lang="ts" setup>
import { formatByteSize } from '@/common/utils/format';
import TreeNodeRow from '@/views/ops/resource/tree/TreeNodeRow.vue';
import type { TreeNode } from '@/views/ops/resource/tree/types';

defineProps<{
    data: TreeNode;
    showActions?: boolean;
}>();
</script>

<style lang="scss"></style>
