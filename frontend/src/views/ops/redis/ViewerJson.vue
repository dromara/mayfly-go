<template>
    <div class="text-formated-container">
        <monaco-editor ref="monacoEditorRef" :canChangeMode="false" v-model="state.modelValue" language="json" />
    </div>
</template>
<script lang="ts" setup>
import { ref, watch, reactive, onMounted } from 'vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';

const props = defineProps({
    content: {
        type: String,
        default: '',
    },
});

const monacoEditorRef = ref(null) as any;

const state = reactive({
    modelValue: '',
    content: null as any,
});

// 因为默认从Text viewer开始，暂时不watch（保存时会触发重新格式化）。
watch(
    () => props.content,
    (val: any) => {
        setContent(val);
    }
);

onMounted(() => {
    setContent(props.content);
});

const setContent = (val: any) => {
    state.modelValue = val;
    setTimeout(() => {
        monacoEditorRef.value.format();
    }, 100);
};

const getContent = () => {
    // 尝试压缩json
    try {
        state.content = JSON.stringify(JSON.parse(state.modelValue));
        return state.content;
    } catch (e) {
        return state.modelValue;
    }
};

defineExpose({ getContent });
</script>
<style lang="scss"></style>
