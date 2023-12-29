<template>
    <div class="dynamic-form">
        <el-form v-bind="$attrs" ref="formRef" :model="modelValue" label-width="auto">
            <el-form-item v-for="item in props.formItems as any" :key="item.name" :prop="item.model" :label="item.name" :required="item.required ?? true">
                <el-input v-if="!item.options" v-model="modelValue[item.model]" :placeholder="item.placeholder" autocomplete="off" clearable></el-input>

                <el-select v-else v-model="modelValue[item.model]" :placeholder="item.placeholder" filterable autocomplete="off" clearable style="width: 100%">
                    <el-option v-for="option in item.options.split(',')" :key="option" :label="option" :value="option" />
                </el-select>
            </el-form-item>
        </el-form>
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';

const props = defineProps({
    formItems: { type: Array },
});

const formRef: any = ref();

const modelValue: any = defineModel();

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
