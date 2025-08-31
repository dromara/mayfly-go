<template>
    <BaseTreeNode v-bind="$attrs">
        <template #suffix="{ data }">
            <span>{{ formatByteSize(data.params.size) }}</span>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import { formatByteSize } from '@/common/utils/format';
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>
