<template>
    <div class="dynamic-form">
        <el-form v-bind="$attrs" ref="formRef" :model="formData" label-width="auto">
            <el-form-item v-for="item in formItems as any" :key="item.name" :prop="item.model" :label="item.name" required>
                <el-input v-if="!item.options" v-model="formData[item.model]" :placeholder="item.placeholder" autocomplete="off" clearable></el-input>

                <el-select v-else v-model="formData[item.model]" :placeholder="item.placeholder" filterable autocomplete="off" clearable style="width: 100%">
                    <el-option v-for="option in item.options.split(',')" :key="option" :label="option" :value="option" />
                </el-select>
            </el-form-item>
        </el-form>
    </div>
</template>

<script lang="ts" setup>
import { useVModel } from '@vueuse/core';
import { ref } from 'vue';

const props = defineProps({
    formItems: { type: Array },
    modelValue: { type: Object },
});

const emit = defineEmits(['update:modelValue']);

const formRef: any = ref();

const formData: any = useVModel(props, 'modelValue', emit);

const validate = async (func: any) => {
    await formRef.value.validate(func);
};

const resetFields = () => {
    formRef.value.resetFields();
};

defineExpose({
    validate,
    resetFields,
});
</script>
<style lang="scss"></style>
