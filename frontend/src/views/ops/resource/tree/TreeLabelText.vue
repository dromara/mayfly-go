<template>
    <span v-if="segments" class="tree-label-text">
        <span v-for="(segment, index) in segments" :key="index" :class="{ 'tree-label-hit': segment.hit }">{{ segment.text }}</span>
    </span>
    <span v-else>{{ text }}</span>
</template>

<script lang="ts" setup>
/**
 * 树节点标签文本（查找态下按命中片段标色）
 *
 * 单独成组件的原因：默认行与禁用行（el-link）两条分支都要渲染同一份片段，内联会写两遍切分逻辑。
 * 贡献者自带 labelRenderer 的行（如标签路径分段）不经过本组件，其高亮由该渲染器自行决定。
 */
import { computed } from 'vue';

import { labelHitSegments } from './filter';

const props = defineProps<{
    /** 节点展示文本（已 i18n，与过滤谓词同一份字符串） */
    text: string;
    /** 生效中的查找词，空串表示不在查找态 */
    query?: string;
}>();

const segments = computed(() => labelHitSegments(props.query ?? '', props.text));
</script>

<style scoped>
/* 命中片段：与本页查找入口同一套主色（激活页签即 primary-light-9 底 + primary 字），明暗主题均由变量驱动 */
.tree-label-hit {
    color: var(--el-color-primary);
    background-color: var(--el-color-primary-light-9);
    border-radius: 2px;
    font-weight: 600;
}
</style>
