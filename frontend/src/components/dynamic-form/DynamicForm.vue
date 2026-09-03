<template>
    <div class="dynamic-form">
        <el-form v-bind="$attrs" ref="formRef" :model="modelValue" label-width="auto">
            <el-form-item v-for="item in props.formItems" :key="item.name" :prop="item.model" :label="$t(item.name)" :required="item.required ?? true">
                <el-input v-if="!item.options" v-model="modelValue[item.model]" :placeholder="item.placeholder ? $t(item.placeholder) : ''" autocomplete="off" clearable></el-input>

                <el-select
                    v-else
                    v-model="modelValue[item.model]"
                    :placeholder="item.placeholder ? $t(item.placeholder) : ''"
                    filterable
                    autocomplete="off"
                    clearable
                    style="width: 100%"
                >
                    <el-option v-for="option in item.options.split(',')" :key="option" :label="option" :value="option" />
                </el-select>
            </el-form-item>
        </el-form>
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import type { FormInstance } from 'element-plus';

export interface FormItem {
    model: string;
    name: string;
    placeholder: string;
    options?: string;
    required?: boolean;
    [key: string]: unknown;
}

const props = defineProps<{
    formItems: FormItem[];
}>();

const formRef = ref<FormInstance>();

const modelValue = defineModel<Record<string, unknown>>({ default: () => ({}) });

// validate 支持两种风格：无参调用时返回Promise（校验失败reject）；传入回调时结果经由回调返回，Promise不reject
const validate = async (callback?: (valid: boolean, invalidFields?: unknown) => void) => {
    return await formRef.value?.validate(callback);
};

const resetFields = () => {
    formRef.value?.resetFields();
};

defineExpose({
    validate,
    resetFields,
});
</script>
