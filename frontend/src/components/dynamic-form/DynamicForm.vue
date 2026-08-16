<template>
    <div class="dynamic-form">
        <el-form v-bind="$attrs" ref="formRef" :model="modelValue" label-width="auto">
            <el-form-item v-for="item in props.formItems" :key="item.name" :prop="item.model" :label="$t(item.name)" :required="item.required ?? true">
                <el-input v-if="!item.options" v-model="modelValue[item.model]" :placeholder="$t(item.placeholder)" autocomplete="off" clearable></el-input>

                <el-select
                    v-else
                    v-model="modelValue[item.model]"
                    :placeholder="$t(item.placeholder)"
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

const validate = async () => {
    return await formRef.value?.validate();
};

const resetFields = () => {
    formRef.value?.resetFields();
};

defineExpose({
    validate,
    resetFields,
});
</script>
