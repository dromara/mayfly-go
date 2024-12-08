<template>
    <div>
        <el-input type="textarea" v-model="modelValue" />
    </div>
</template>
<script lang="ts" setup>
import { reactive, watch, toRefs, onMounted } from 'vue';

const props = defineProps({
    content: {
        type: String,
    },
});

const state = reactive({
    modelValue: '',
});

const { modelValue } = toRefs(state);

watch(
    () => props.content,
    (val: any) => {
        state.modelValue = val;
    }
);

onMounted(() => {
    state.modelValue = props.content as any;
});

const getContent = () => {
    return state.modelValue;
};

defineExpose({ getContent });
</script>
<style lang="scss"></style>
