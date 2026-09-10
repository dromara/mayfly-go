<template>
    <AInputNumber v-model="value" :min="item.min" :max="item.max" :disabled="disabled" class="w-full!" v-bind="item.props">
        <template v-if="item.prefix" #prefix>{{ $t(item.prefix) }}</template>
        <template v-if="item.suffix" #suffix>{{ $t(item.suffix) }}</template>
    </AInputNumber>
</template>

<script lang="ts" setup>
import { AInputNumber } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { value, disabled } = useFieldControl(props, modelValue);
</script>
