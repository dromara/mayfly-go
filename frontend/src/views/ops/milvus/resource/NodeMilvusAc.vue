<template>
    <BaseTreeNode v-bind="$attrs">
        <template #suffix="{ data }">
            <span>{{ ` ${(data.params.selectAuthCert as { username?: string } | undefined)?.username}@${data.params.host}` }}</span>
        </template>
    </BaseTreeNode>
</template>

<script lang="ts" setup>
import BaseTreeNode from '@/views/ops/resource/BaseTreeNode.vue';
</script>

<style lang="scss"></style>