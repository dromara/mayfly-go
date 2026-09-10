<template>
    <monaco-editor v-model="value" class="w-full" v-bind="item.props" :options="monacoOptions" />
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { useFieldControl, type FieldControlProps } from '../useFieldControl';

const props = defineProps<FieldControlProps>();

const modelValue = defineModel<any>();

const { value, disabled } = useFieldControl(props, modelValue);

/** Monaco 编辑器选项：合并调用方自定义 options，并强制注入 readOnly；
 *  调用方显式 readOnly: true 保留，但不可覆盖 disabled/readonly 状态的强制只读 */
const monacoOptions = computed(() => {
    const callerOptions = ((props.item.props as { options?: Record<string, unknown> } | undefined)?.options ?? {}) as { readOnly?: boolean };
    return { ...callerOptions, readOnly: disabled.value || callerOptions.readOnly === true };
});
</script>
