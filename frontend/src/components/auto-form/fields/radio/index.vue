<template>
    <ARadioGroup v-model="value" :disabled="disabled" v-bind="item.props">
        <ARadio v-for="opt in radioOptions" :key="`${opt.value}`" :value="opt.value" :disabled="opt.disabled || isOptionDisabled(opt.value)">
            {{ $t(opt.label) }}
        </ARadio>
    </ARadioGroup>
</template>

<script lang="ts" setup>
import { ARadioGroup, ARadio } from '../../ui/adapter';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { value, disabled, radioOptions, isOptionDisabled } = useFieldControl(props, modelValue);
</script>
