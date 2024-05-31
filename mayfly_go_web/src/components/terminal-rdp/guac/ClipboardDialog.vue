<template>
    <div class="clipboard-dialog">
        <el-dialog
            v-model="dialogVisible"
            title="请输入需要粘贴的文本"
            :before-close="onclose"
            :close-on-click-modal="false"
            :close-on-press-escape="false"
            width="600"
        >
            <el-input v-model="state.modelValue" type="textarea" :rows="20" />

            <template #footer>
                <el-button type="primary" @click="onsubmit">确 定</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script setup lang="ts">
import { reactive, toRefs, watch } from 'vue';
import { ElMessage } from 'element-plus';

const props = defineProps({
    visible: { type: Boolean },
});

const emits = defineEmits(['submit', 'close', 'update:visible']);

const state = reactive({
    dialogVisible: false,
    modelValue: '',
});

const { dialogVisible } = toRefs(state);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
});

const onclose = () => {
    emits('update:visible', false);
    emits('close');
};

const onsubmit = () => {
    state.dialogVisible = false;
    if (state.modelValue) {
        ElMessage.success('发送剪贴板数据成功');
        emits('submit', state.modelValue);
    } else {
        ElMessage.warning('请输入需要粘贴的文本');
    }
};

const setValue = (val: string) => {
    state.modelValue = val;
};

defineExpose({ setValue });
</script>

<style lang="scss">
.clipboard-dialog {
}
</style>
