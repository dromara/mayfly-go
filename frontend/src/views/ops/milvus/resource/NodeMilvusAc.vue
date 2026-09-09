<template>
    <TreeNodeRow :data="data" :show-actions="showActions">
        <template #suffix="{ data }">
            <span>{{ ` ${(data.params.selectAuthCert as { username?: string } | undefined)?.username}@${data.params.host}` }}</span>
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