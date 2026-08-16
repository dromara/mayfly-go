<template>
    <div class="form-dialog">
        <el-dialog @close="close" v-bind="$attrs" :title="title" v-model="dialogVisible" :width="width">
            <dynamic-form ref="df" :form-items="props.formItems" v-model="formData" />

            <template #footer>
                <span>
                    <slot name="btns">
                        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                        <el-button type="primary" @click="confirm">{{ $t('common.confirm') }}</el-button>
                    </slot>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import DynamicForm from './DynamicForm.vue';
import type { FormItem } from './DynamicForm.vue';

const emit = defineEmits(['close', 'confirm']);

const props = defineProps<{
    title?: string;
    width?: string | number;
    formItems: FormItem[];
}>();

const df = ref<InstanceType<typeof DynamicForm>>();

const formData = defineModel<Record<string, unknown>>('modelValue');
const dialogVisible = defineModel<boolean>('visible', { default: false });

const close = () => {
    emit('close');
    // 取消动态表单的校验
    setTimeout(() => {
        formData.value = {};
        df.value?.resetFields();
    }, 200);
};

const confirm = async () => {
    try {
        await df.value?.validate();
        emit('confirm', formData.value);
    } catch {
        // 校验未通过
    }
};
</script>
