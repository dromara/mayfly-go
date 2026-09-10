<template>
    <AInputTag v-model="value" :placeholder="placeholder" :disabled="disabled" v-bind="item.props" />
</template>

<script lang="ts" setup>
import { AInputTag } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { value, disabled, placeholder } = useFieldControl(props, modelValue);
</script>
