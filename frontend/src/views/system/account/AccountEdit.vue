<template>
    <div class="account-dialog">
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :show-close="false" width="600px" :destroy-on-close="true">
            <el-form :model="form" ref="accountForm" :rules="rules" label-width="auto">
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
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref, watchEffect } from 'vue';
import { accountApi } from '../api';
import { randomPassword } from '@/common/utils/string';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    account: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const accountForm: any = ref(null);

const rules = {
    name: [Rules.requiredInput('system.account.name')],
    username: [Rules.requiredInput('common.username'), Rules.accountUsername],
    password: [Rules.requiredInput('common.password')],
};

const state = reactive({
    dialogVisible: false,
    edit: false,
    form: {
        id: null,
        name: null,
        username: null,
        password: '',
        repassword: null,
    },
});

const { dialogVisible, edit, form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveAccountExec } = accountApi.save.useApi(form);

watch(props, (newValue: any) => {
    if (newValue.account) {
        state.form = { ...newValue.account };
        state.edit = true;
    } else {
        state.edit = false;
        state.form = {} as any;
    }
    state.dialogVisible = newValue.visible;
});

watchEffect(() => {
    const account: any = props.account;
    if (account) {
        state.form = { ...account };
        state.edit = true;
    } else {
        state.edit = false;
        state.form = {} as any;
    }
    state.dialogVisible = props.visible;
});

const btnOk = async () => {
    await useI18nFormValidate(accountForm);
    await saveAccountExec();
    useI18nSaveSuccessMsg();
    emit('val-change', state.form);
    //重置表单域
    accountForm.value.resetFields();
    state.form = {} as any;
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
