<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="600px"
            :confirm-api="onConfirm"
            @submitted="emit('cancel')"
            @cancel="emit('cancel')"
        >
            <!-- 密码字段：带一键随机生成按钮（自定义插槽） -->
            <template #password="{ form: f }">
                <el-input type="password" :model-value="f.password" autocomplete="new-password" show-password @update:model-value="(v: string) => (f.password = v)">
                    <template #append>
                        <el-button @click="f.password = randomPassword(10)">{{ $t('system.account.random') }}</el-button>
                    </template>
                </el-input>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { randomPassword } from '@/common/utils/string';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { computed, useTemplateRef, type PropType } from 'vue';
import { accountApi } from '../api';
import type { Account } from '../types';

const props = defineProps({
    data: {
        type: Object as PropType<Account | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'val-change']);

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

const isEdit = computed(() => !!props.data);

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；password 走自定义插槽，extra.* 为嵌套路径字段） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'system.account.name', required: true },
    { prop: 'username', label: 'common.username', placeholder: 'system.account.usernamePlacholder', disabled: (form) => !!form.id, required: true, rules: [Rules.accountUsername] },
    { prop: 'mobile', label: 'common.mobile' },
    { prop: 'email', label: 'common.email' },
    { prop: 'password', label: 'common.password', required: true, slot: 'password' },
    { prop: 'extra.qywxUserId', label: 'system.account.qywxUserId' },
    { prop: 'extra.feishuUserId', label: 'system.account.feishuUserId' },
];

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData>(() => {
    const account = props.data;
    if (account) {
        return {
            ...account,
            extra: account.extra || { qywxUserId: '', feishuUserId: '' },
        } as AutoFormData;
    }
    return {
        id: null,
        name: null,
        username: null,
        mobile: null,
        email: null,
        password: '',
        extra: { qywxUserId: '', feishuUserId: '' },
    } as AutoFormData;
});

const { execute: saveAccountExec } = accountApi.save.useApi();

// confirmApi 提交动作；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async (form: AutoFormData) => {
    await saveAccountExec(form);
    emit('val-change', form);
};
</script>
<style lang="scss"></style>
