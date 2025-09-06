<template>
    <div class="h-full">
        <monaco-editor
            ref="editorRef"
            :height="props.height"
            class="editor"
            language="text"
            v-model="modelValue"
            :options="{
                readOnly: true,
            }"
            :can-change-mode="false"
        />
    </div>
</template>

<script lang="ts" setup>
import { ref, useTemplateRef, watch } from 'vue';
import { useWebSocket } from '@vueuse/core';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';

const props = defineProps({
    height: {
        type: String,
        default: '100%',
    },
    wsUrl: {
        type: String,
        default: '',
    },
});

const websocketUrl = ref(props.wsUrl);

const { data } = useWebSocket(websocketUrl);

const editorRef: any = useTemplateRef('editorRef');

const modelValue = defineModel<string>('modelValue', {
    type: String,
    default: '',
});

watch(data, (value) => {
    // eslint-disable-next-line no-control-regex
    modelValue.value = modelValue.value + value.replace(/\x1B\[[0-?]*[ -/]*[@-~]/g, '');
    setTimeout(() => {
        revealLastLine();
    }, 200);
});

const reload = (wsUrl: string) => {
    modelValue.value = '';
    websocketUrl.value = wsUrl;
    revealLastLine();
};

const revealLastLine = () => {
    const editor = editorRef.value.getEditor();
    const lineCount = editor?.getModel().getLineCount();
    editor.revealLine(lineCount);
};

defineExpose({
    reload,
});
</script>
<style lang="scss" scoped>
.editor {
    font-size: 9pt;
    font-weight: 600;
}
</style>
