<template>
    <AutoFormDialog
        v-model:visible="visible"
        :title="title"
        :items="formItems"
        :data="data"
        :confirm-api="onConfirm"
        width="600px"
        @submitted="emit('cancel')"
        @cancel="emit('cancel')"
    />
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import { roleApi } from '../api';
import { RoleStatusEnum } from '../enums';
import { AutoFormDialog, type AutoFormData, type AutoFormItem } from '@/components/auto-form';

defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

// 配置化表单：渲染与校验统一由 AutoFormDialog 处理
const formItems: AutoFormItem[] = [
    { prop: 'name', label: 'system.role.roleName', required: true },
    {
        prop: 'code',
        label: 'system.role.roleCode',
        required: true,
        placeholder: 'system.role.roleCodePlaceholder',
        disabled: (form) => form.id != null,
    },
    { prop: 'status', label: 'common.status', type: 'enum', enums: RoleStatusEnum, required: true, defaultValue: 1 },
    { prop: 'remark', label: 'common.remark', type: 'textarea', rows: 3 },
];

const form = ref<AutoFormData>({});

const { execute: saveRoleExec } = roleApi.save.useApi(form);

// confirmApi 提交动作（将表单写入请求源 form 后提交）；成功提示与关闭弹窗由组件内置逻辑处理
const onConfirm = async (formData: AutoFormData) => {
    form.value = formData;
    await saveRoleExec();
    emit('val-change', formData);
};
</script>
<style lang="scss"></style>
