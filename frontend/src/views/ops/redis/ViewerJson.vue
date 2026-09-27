<template>
    <div class="text-formated-container h-full">
        <MonacoEditor v-model="content" :can-change-mode="false" :options="{ readOnly: props.readonly }" language="json" />
    </div>
</template>
<script lang="ts" setup>
import { defineAsyncComponent, watch } from 'vue';
// monaco 体积大，按「组件内异步边界」规范延迟加载，避免把它拉进 redis 模块首屏 chunk
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const props = defineProps<{
    content?: string;
    readonly?: boolean;
}>();

const content = defineModel<string>('content', { default: '' });

watch(
    () => props.content,
    (val) => {
        if (val !== undefined && val !== content.value) {
            content.value = val;
        }
    }
);

const getContent = () => content.value;

defineExpose({ getContent });
</script>
<style lang="scss" scoped>
.text-formated-container :deep(.monaco-editor-content) {
    height: 100% !important;
}
</style>
