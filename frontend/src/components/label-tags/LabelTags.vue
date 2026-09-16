<template>
    <div v-if="pairs.length" class="label-tags" :class="[`label-tags--${size}`]">
        <el-tooltip
            v-for="(label, idx) in pairs"
            :key="idx"
            :content="getLabelDescription(label.key, label.value)"
            :disabled="!getLabelDescription(label.key, label.value)"
            placement="top"
        >
            <el-tag
                :size="size"
                :style="getLabelTagStyle(label.key, label.value)"
                class="label-tags__item"
            >
                {{ label.key }}:{{ label.value }}
            </el-tag>
        </el-tooltip>
    </div>
    <span v-else class="label-tags__empty">-</span>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { parseLabelPairs, getLabelTagStyle, getLabelDescription } from '@/hooks/useLabelFill';

interface LabelTagsProps {
    /** 标签 JSON 字符串，如 '{"env":"prod","team":"infra"}' */
    labels?: string;
    /** el-tag 尺寸 */
    size?: 'large' | 'default' | 'small';
}

const props = withDefaults(defineProps<LabelTagsProps>(), {
    labels: '',
    size: 'small',
});

const pairs = computed(() => parseLabelPairs(props.labels));
</script>

<style lang="scss" scoped>
.label-tags {
    display: inline-flex;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;

    &--small {
        gap: 3px;
    }

    &__item {
        margin: 0;
        cursor: default;
    }

    &__empty {
        color: var(--el-text-color-placeholder);
    }
}
</style>
