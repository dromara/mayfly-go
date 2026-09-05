<template>
    <div>
        <el-dialog :title="title" v-model="visible" :before-close="onCancel" :show-close="false" width="600px" :destroy-on-close="true">
            <AutoForm ref="accountFormRef" v-model="form" :items="items" label-width="auto">
                <!-- 密码字段：带一键随机生成按钮（自定义插槽） -->
                <template #password="{ form: f }">
                    <el-input type="password" :model-value="f.password" autocomplete="new-password" show-password @update:model-value="(v: string) => (f.password = v)">
                        <template #append>
                            <el-button @click="f.password = randomPassword(10)">{{ $t('system.account.random') }}</el-button>
                        </template>
                    </el-input>
                </template>
            </AutoForm>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { randomPassword } from '@/common/utils/string';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { computed, reactive, toRefs, useTemplateRef, watch, type PropType } from 'vue';
import { accountApi } from '../api';
import type { Account } from '../types';

const props = defineProps({
    account: {
        type: Object as PropType<Account | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const visible = defineModel<boolean>('visible', { default: false });

const accountFormRef = useTemplateRef('accountFormRef');

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；extra.* 为嵌套路径字段） */
const items = computed<AutoFormItem[]>(() => [
    { prop: 'name', label: 'system.account.name', required: true },
    { prop: 'username', label: 'common.username', placeholder: 'system.account.usernamePlacholder', disabled: edit.value, required: true, rules: [Rules.accountUsername] },
    { prop: 'mobile', label: 'common.mobile' },
    { prop: 'email', label: 'common.email' },
    { prop: 'password', label: 'common.password', required: true, slot: 'password' },
    { prop: 'extra.qywxUserId', label: 'system.account.qywxUserId' },
    { prop: 'extra.feishuUserId', label: 'system.account.feishuUserId' },
]);

const defaultForm = (): Record<string, any> => {
    return {
        id: null,
        name: null,
        username: null,
        mobile: null,
        email: null,
        password: '',
        repassword: null,
        extra: {
            qywxUserId: '',
            feishuUserId: '',
        },
    };
};

const state = reactive({
    edit: false,
    form: defaultForm(),
});

const { edit, form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveAccountExec } = accountApi.save.useApi(form);

watch(props, (newValue) => {
    if (newValue.account) {
        state.form = { ...newValue.account };
        if (!state.form.extra) {
            state.form.extra = { qywxUserId: '', feishuUserId: '' };
        }
        state.edit = true;
    } else {
        state.edit = false;
        state.form = defaultForm();
    }
});

const onConfirm = async () => {
    await useI18nFormValidate(accountFormRef);
    await saveAccountExec();
    Msg.saveSuccess();
    emit('val-change', state.form);
    //重置表单域
    accountFormRef.value?.resetFields();
};

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
