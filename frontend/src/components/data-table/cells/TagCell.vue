<template>
    <EnumTag :size="size" :enums="item.typeParam ?? {}" :value="tagValue" />
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import type { CellRendererProps } from '../cellRenderers';

/**
 * 枚举 tag 列的单元格渲染器：把统一的 CellRendererProps 适配到 EnumTag 的入参
 * （枚举源取自 item.typeParam，展示值即 value）。由 cellRenderers 按 type='tag' 选中。
 */
const props = defineProps<CellRendererProps>();

// 在脚本里收敛 value 类型：模板里写 `as A | B` 会被 vue 模板解析当成过滤器（vue/no-deprecated-filter）
const tagValue = computed(() => props.value as string | number | boolean | null | undefined);
</script>
