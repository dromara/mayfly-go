<template>
    <div>
        <el-dialog :title="title" v-model="visible" :before-close="onCancel" :show-close="false" width="600px" :destroy-on-close="true">
            <el-form :model="form" ref="accountFormRef" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('system.account.name')">
                    <el-input v-model.trim="form.name" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-form-item prop="username" :label="$t('common.username')">
                    <el-input
                        :disabled="edit"
                        v-model.trim="form.username"
                        :placeholder="$t('system.account.usernamePlacholder')"
                        auto-complete="off"
                        clearable
                    ></el-input>
                </el-form-item>

                <el-form-item prop="mobile" :label="$t('common.mobile')">
                    <el-input v-model.trim="form.mobile" clearable></el-input>
                </el-form-item>

                <el-form-item prop="email" :label="$t('common.email')">
                    <el-input v-model.trim="form.email" auto-complete="off" clearable></el-input>
                </el-form-item>

                <el-form-item :required="!edit" prop="password" :label="$t('common.password')">
                    <el-input type="password" v-model.trim="form.password" autocomplete="new-password" show-password>
                        <template #append>
                            <el-button
                                @click="
                                    {
                                        form.password = randomPassword(10);
                                    }
                                "
                            >
                                {{ $t('system.account.random') }}
                            </el-button>
                        </template>
                    </el-input>
                </el-form-item>

                <el-form-item :label="$t('system.account.qywxUserId')">
                    <el-input v-model.trim="form.extra.qywxUserId" clearable></el-input>
                </el-form-item>
                <el-form-item :label="$t('system.account.feishuUserId')">
                    <el-input v-model.trim="form.extra.feishuUserId" clearable></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, useTemplateRef } from 'vue';
import { accountApi } from '../api';
import { randomPassword } from '@/common/utils/string';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const props = defineProps({
    account: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const visible = defineModel<boolean>('visible', { default: false });

const accountFormRef: any = useTemplateRef('accountFormRef');

const rules = {
    name: [Rules.requiredInput('system.account.name')],
    username: [Rules.requiredInput('common.username'), Rules.accountUsername],
    password: [Rules.requiredInput('common.password')],
};

const defaultForm = () => {
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

watch(props, (newValue: any) => {
    if (newValue.account) {
        state.form = { ...newValue.account };
        if (!state.form.extra) {
            state.form.extra = {} as any;
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
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    //重置表单域
    accountFormRef.value.resetFields();
};

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
