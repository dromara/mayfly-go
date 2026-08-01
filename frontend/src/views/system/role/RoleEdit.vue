<template>
    <AutoFormDialog
        v-model:visible="visible"
        :title="title"
        :items="formItems"
        :data="data"
        :confirm-loading="saveBtnLoading"
        width="600px"
        @confirm="onConfirm"
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

const { isFetching: saveBtnLoading, execute: saveRoleExec } = roleApi.save.useApi(form);

const onConfirm = async (formData: AutoFormData) => {
    form.value = formData;
    await saveRoleExec();
    emit('val-change', formData);
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
