<template>
    <div class="compaction-part">
        <span class="compaction-part__divider" />
        <ArchiveIcon class="compaction-part__icon" :size="12" />
        <span class="compaction-part__title">{{ t('ai.chat.contextCompacted') }}</span>
        <span class="compaction-part__meta">
            {{ t('ai.chat.compactionDetail', { count, original, compressed }) }}
        </span>
        <span class="compaction-part__divider compaction-part__divider--grow" />
    </div>
</template>

<script setup lang="ts">
/**
 * CompactionPart - 上下文压缩提示节点
 * 对齐 tokhub 的 ContextCompaction item 渲染：时间线上的轻量分隔提示行，
 * 展示"上下文已压缩"及压缩前后 token 统计
 */
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { ArchiveIcon } from '@lucide/vue';

const props = defineProps<{
    originalTokens: number;
    compressedTokens: number;
    compressedMessageCount: number;
}>();

const { t } = useI18n();

const count = computed(() => props.compressedMessageCount);
const original = computed(() => props.originalTokens.toLocaleString());
const compressed = computed(() => props.compressedTokens.toLocaleString());
</script>

<style scoped>
.compaction-part {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 0;
    font-size: 12px;
    color: var(--el-text-color-placeholder);
}

.compaction-part__divider {
    height: 1px;
    width: 24px;
    background: var(--el-border-color-lighter);
    flex-shrink: 0;
}

.compaction-part__divider--grow {
    flex: 1;
    width: auto;
    max-width: 120px;
}

.compaction-part__icon {
    flex-shrink: 0;
}

.compaction-part__title {
    color: var(--el-text-color-secondary);
    flex-shrink: 0;
}

.compaction-part__meta {
    font-variant-numeric: tabular-nums;
}
</style>
