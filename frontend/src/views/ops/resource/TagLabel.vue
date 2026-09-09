<template>
    <!-- 标签路径 label：'/' 分段高亮渲染（经贡献者 labelRenderer 挂载，kind 知识归资源侧） -->
    <span>
        <template v-for="(v, i) in segments" :key="i">
            <el-text v-if="i !== 0" tag="b" type="primary" size="large" class="mx-[1.5px]!">/</el-text>
            <el-text>{{ v }}</el-text>
        </template>
    </span>
</template>

<script lang="ts" setup>
import { computed } from 'vue';

import type { TreeNode } from '@/views/ops/resource/tree/types';

const props = defineProps<{
    node: TreeNode;
}>();

const segments = computed(() => props.node.label.split('/'));
</script>
