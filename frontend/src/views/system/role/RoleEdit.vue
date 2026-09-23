<template>
    <AutoFormDialog
        v-model:visible="visible"
        :title="title"
        :items="formItems"
        :data="data"
        :confirm-api="onConfirm"
        @opened="onOpened"
        width="600px"
        @submitted="emit('cancel')"
        @cancel="emit('cancel')"
    />
</template>

<script lang="ts" setup>
import { roleApi } from '../api';
import { RoleStatusEnum } from '../enums';
import { AutoFormDialog, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import type { RoleForm } from '../types';

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

// 配置化表单：渲染与校验统一由 AutoFormDialog 处理（回调形参直接是角色表单）
const formItems = defineFormItems<RoleForm>([
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
]);

// 宿主内部表单在 @opened 接管（提交即它的当前值，无需再拷一份请求源）
const { onOpened, requireForm } = useAutoFormModel<RoleForm>();

const { execute: saveRoleExec } = roleApi.save.useApi();

// confirmApi 提交动作；成功提示与关闭弹窗由组件内置逻辑处理
const onConfirm = async () => {
    const form = requireForm();
    await saveRoleExec(form);
    emit('val-change', form);
};
</script>
<style lang="scss"></style>
