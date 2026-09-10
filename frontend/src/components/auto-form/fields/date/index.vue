<template>
    <!-- 日期 / 日期时间 -->
    <ADatePicker
        v-if="type == 'date' || type == 'datetime'"
        v-model="value"
        :type="type"
        :placeholder="placeholder"
        :disabled="disabled"
        class="w-full"
        v-bind="item.props"
    />
    <!-- 时间 -->
    <ATimePicker v-else v-model="value" :placeholder="placeholder" :disabled="disabled" class="w-full" v-bind="item.props" />
</template>

<script lang="ts" setup>
import { ADatePicker, ATimePicker } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { type, value, disabled, placeholder } = useFieldControl(props, modelValue);
</script>
