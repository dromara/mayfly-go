<template>
    <AInput
        v-model="value"
        :type="type == 'password' ? 'password' : 'text'"
        :show-password="type == 'password'"
        :placeholder="placeholder"
        :disabled="disabled"
        autocomplete="off"
        clearable
        v-bind="item.props"
    >
        <template v-if="item.prefix" #prefix>{{ $t(item.prefix) }}</template>
        <template v-if="item.suffix" #suffix>{{ $t(item.suffix) }}</template>
    </AInput>
</template>

<script lang="ts" setup>
import { AInput } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { type, value, disabled, placeholder } = useFieldControl(props, modelValue);
</script>
