<template>
    <BaseTreeNode v-bind="$attrs">
        <template #suffix="{ data }">
            <span>{{ ` ${acParams(data).selectAuthCert?.username}@${acParams(data).ip}:${acParams(data).port}` }}</span>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
import type { MachineNodeParams } from './index';

/** 将树节点 params 还原为机器节点参数 */
const acParams = (data: { params?: unknown }): MachineNodeParams => (data.params ?? {}) as MachineNodeParams;
</script>

<style lang="scss"></style>
