<template>
    <BaseTreeNode v-bind="$attrs">
        <template #suffix="{ data }">
            <span>{{ ` ${data.params.selectAuthCert.username}@${data.params.ip}:${data.params.port}` }}</span>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>
