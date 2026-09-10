<template>
    <AInput
        v-model="value"
        type="textarea"
        :rows="item.rows ?? 3"
        :placeholder="placeholder"
        :disabled="disabled"
        v-bind="item.props"
    />
</template>

<script lang="ts" setup>
import { AInput } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { value, disabled, placeholder } = useFieldControl(props, modelValue);
</script>
