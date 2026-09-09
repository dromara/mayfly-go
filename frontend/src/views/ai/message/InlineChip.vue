<template>
    <span
        class="inline-chip"
        :style="{ background: reg?.color.bg, color: reg?.color.text, borderColor: reg?.color.border }"
        :title="text"
    >
        <span class="chat-chip__icon" :data-icon="icon" />
        <span class="inline-chip__name">{{ text }}</span>
    </span>
</template>

<script setup lang="ts">
/**
 * InlineChip - 消息气泡内的芯片回显标签
 * InlineChip：图标 + 名称，图标与配色查 chipRegistry（与编辑器 chipNode 同源），
 * 图标复用 ChatInput 全局 .chat-chip__icon mask 样式（zap/server/database）
 */
import { computed } from 'vue';
import type { ContentSegment } from '../protocol/types';
import { getChipType } from '../input/chipRegistry';
import '../input/chipTypes';

const props = defineProps<{ segment: ContentSegment }>();

const reg = computed(() => getChipType(props.segment.type));

/** resource 按子类型细分图标（机器 server / 数据库 database），其余取注册图标 */
const icon = computed(() => {
    if (props.segment.type === 'resource') {
        return props.segment.extra?.resourceType === 'db' ? 'database' : 'server';
    }
    return reg.value?.icon || 'zap';
});

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
