<template>
    <el-input v-model="content" class="viewer-text h-full" type="textarea" resize="none" :readonly="props.readonly" />
</template>
<script lang="ts" setup>
import { watch } from 'vue';

const props = defineProps<{
    content?: string;
    readonly?: boolean;
}>();

const content = defineModel<string>('content', { default: '' });

// 宿主通过 :content 传入初始值，双向绑定后需要把外部变化同步进来
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
.viewer-text :deep(.el-textarea__inner) {
    height: 100%;
}
</style>
