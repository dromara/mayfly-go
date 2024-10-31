<template>
    <div class="form-dialog">
        <el-dialog @close="close" v-bind="$attrs" :title="title" v-model="dialogVisible" :width="width">
            <dynamic-form ref="df" :form-items="props.formItems" v-model="formData" />

            <template #footer>
                <span>
                    <slot name="btns">
                        <el-button @click="dialogVisible = false">取 消</el-button>
                        <el-button type="primary" @click="confirm">确 定</el-button>
                    </slot>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import DynamicForm from './DynamicForm.vue';

const emit = defineEmits(['close', 'confirm']);

const props = defineProps({
    title: { type: String },
    width: { type: [String, Number], default: '500px' },
    formItems: { type: Array },
});

const df: any = ref();

const formData: any = defineModel('modelValue');
const dialogVisible = defineModel<boolean>('visible', { default: false });

const close = () => {
    emit('close');
    // 取消动态表单的校验
    setTimeout(() => {
        formData.value = {};
        df.value.resetFields();
    }, 200);
};

const confirm = () => {
    df.value.validate((valid: any) => {
        if (!valid) {
            return false;
        }
        emit('confirm', formData.value);
    });
};
</script>
