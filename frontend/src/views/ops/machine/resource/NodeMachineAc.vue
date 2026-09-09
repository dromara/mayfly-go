<template>
    <TreeNodeRow :data="data" :show-actions="showActions">
        <template #suffix="{ data }">
            <span>{{ ` ${acParams(data).selectAuthCert?.username}@${acParams(data).ip}:${acParams(data).port}` }}</span>
        </template>
    </TreeNodeRow>
</template>

<script lang="ts" setup>
import TreeNodeRow from '@/views/ops/resource/tree/TreeNodeRow.vue';
import type { TreeNode } from '@/views/ops/resource/tree/types';
import type { MachineNodeParams } from './index';

/** 将树节点 params 还原为机器节点参数 */
const acParams = (data: { params?: unknown }): MachineNodeParams => (data.params ?? {}) as MachineNodeParams;

defineProps<{
    data: TreeNode;
    showActions?: boolean;
}>();
</script>

<style lang="scss"></style>
