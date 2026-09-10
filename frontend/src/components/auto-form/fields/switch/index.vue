<template>
    <ASwitch v-model="value" :disabled="disabled" v-bind="item.props" />
</template>

<script lang="ts" setup>
import { ASwitch } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { value, disabled } = useFieldControl(props, modelValue);
</script>
