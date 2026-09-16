<template>
    <div class="label-association">
        <LabelSelect v-model="labelPairs" />
    </div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import LabelSelect from './LabelSelect.vue';
import type { LabelSelectValue } from '../types';

const props = defineProps<{
    modelValue?: string;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: string): void;
}>();

/** 解析 JSON 字符串为标签对数组 */
const parseJsonToPairs = (text: string): LabelSelectValue[] => {
    if (!text?.trim()) return [];
    try {
        const parsed = JSON.parse(text);
        if (typeof parsed !== 'object' || parsed === null) return [];
        return Object.entries(parsed).map(([key, value]) => ({
            key,
            value: String(value),
        }));
    } catch {
        return [];
    }
};

/** 标签对数组序列化为 JSON 字符串 */
const pairsToJson = (pairs: LabelSelectValue[]): string => {
    const obj: Record<string, string> = {};
    for (const pair of pairs) {
        if (pair.key.trim()) {
            obj[pair.key.trim()] = pair.value;
        }
    }
    return Object.keys(obj).length ? JSON.stringify(obj) : '';
};

// 用 ref 管理内部状态
const labelPairs = ref<LabelSelectValue[]>(parseJsonToPairs(props.modelValue ?? ''));

// 监听内部变化，同步到外部
watch(
    labelPairs,
    (newVal) => {
        emit('update:modelValue', pairsToJson(newVal));
    },
    { deep: true }
);

// 监听外部变化（backfill 等场景）
watch(
    () => props.modelValue,
    (newVal) => {
        const currentJson = pairsToJson(labelPairs.value);
        if (newVal !== currentJson) {
            labelPairs.value = parseJsonToPairs(newVal ?? '');
        }
    }
);
</script>

<style lang="scss" scoped>
.label-association {
    width: 100%;
}
</style>
