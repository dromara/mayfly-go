<template>
    <span class="inline-chip" :style="{ background: reg?.color.bg, color: reg?.color.text, borderColor: reg?.color.border }" :title="text">
        <span class="chat-chip__icon" :data-icon="icon" />
        <span class="inline-chip__name">{{ text }}</span>
    </span>
</template>

<script setup lang="ts">
/**
 * InlineChip - 消息气泡内的芯片回显标签
 * InlineChip：图标 + 名称，图标与配色均与编辑器 chipNode 同源（都走 chipRegistry/chipIcon），
 * 图标 mask 样式在 input/chipStyles.scss 的全局 .chat-chip__icon
 */
import { computed } from 'vue';
import type { ContentSegment } from '../protocol/types';
import { chipIcon, getChipType } from '../input/chipRegistry';
import '../input/chipTypes';
// 图标 mask 样式必须在此显式导入：本组件也产出 .chat-chip__icon DOM。
// 只靠 chipNode 的副作用导入，在纯只读轨迹路径（无编辑器）下图标是 0 尺寸空白块
import '../input/chipStyles.scss';

const props = defineProps<{ segment: ContentSegment }>();

const reg = computed(() => getChipType(props.segment.type));

/** 图标与编辑器同源：子类型判定已收进注册表，这里不再重复写一遍 resourceType 分支 */
const icon = computed(() => chipIcon(props.segment.type, { label: props.segment.text, data: props.segment.extra ?? {} }));

const text = computed(() => props.segment.text);
</script>

<style scoped>
.inline-chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 1px 8px;
    border: 1px solid;
    border-radius: 12px;
    font-size: 12px;
    line-height: 18px;
    vertical-align: baseline;
    cursor: default;
    user-select: none;
    white-space: nowrap;
}

.inline-chip__name {
    max-width: 160px;
    overflow: hidden;
    text-overflow: ellipsis;
}
</style>
