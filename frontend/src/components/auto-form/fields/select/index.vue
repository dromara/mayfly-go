<template>
    <ASelect
        v-model="value"
        :placeholder="placeholder"
        :multiple="item.multiple"
        :loading="state.loading"
        :disabled="disabled"
        filterable
        clearable
        class="w-full"
        v-bind="item.props"
    >
        <AOption
            v-for="option in resolvedOptions"
            :key="`${option.value}`"
            :label="$t(option.label)"
            :value="option.value"
            :disabled="option.disabled || isOptionDisabled(option.value)"
        />
    </ASelect>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { ASelect, AOption } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';
import type { AutoFormSelectOption } from '../../types';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { type, value, disabled, placeholder, enumValues, selectOptions, isOptionDisabled, state, watchOptionsLoader } = useFieldControl(props, modelValue);

/** 统一选项来源：select 类型走异步加载结果，enum 类型走 enums 解析 */
const resolvedOptions = computed<AutoFormSelectOption[]>(() => (type.value === 'select' ? selectOptions.value : enumValues.value));

// 启动异步选项加载（select 类型支持异步 options 函数，enum 类型无 options 时 watchEffect 自动跳过）
watchOptionsLoader();
</script>
