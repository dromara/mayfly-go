<template>
    <BaseTreeNode v-bind="$attrs">
        <template #suffix="{ data }">
            <span v-if="data.params.size">{{ ` ${data.params.size}` }}</span>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>
