<template>
    <div>
        <el-drawer :title="title" v-model="visible" :show-close="false" :before-close="onCancel" size="1000px" :destroy-on-close="true">
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form ref="configFormRef" :model="form" :rules="rules" label-width="auto">
                <el-form-item prop="name" :label="$t('system.sysconf.confItem')" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>
                <el-form-item prop="key" :label="$t('system.sysconf.confKey')" required>
                    <el-input :disabled="form.id != null" v-model="form.key"></el-input>
                </el-form-item>
                <el-form-item prop="permission" :label="$t('system.sysconf.permission')">
                    <el-select
                        remote
                        :remote-method="getAccount"
                        v-model="state.permissionAccount"
                        filterable
                        multiple
                        :placeholder="$t('system.sysconf.permissionPlaceholder')"
                    >
                        <el-option v-for="item in state.accounts" :key="item.id" :label="`${item.username} [${item.name}]`" :value="item.username"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item :label="$t('system.sysconf.confItem')" class="!w-full">
                    <dynamic-form-edit v-model="params" />
                </el-form-item>

                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, useTemplateRef } from 'vue';
import { configApi, accountApi } from '../api';
import { DynamicFormEdit } from '@/components/dynamic-form';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { useI18nFormValidate } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';

const rules = {
    name: [Rules.requiredInput('system.sysconf.confItem')],
    key: [Rules.requiredInput('system.sysconf.confKey')],
};

const props = defineProps({
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

const configFormRef: any = useTemplateRef('configFormRef');

const state = reactive({
    params: [] as any,
    accounts: [] as any,
    permissionAccount: [] as any,
    form: {
        id: null,
        name: '',
        key: '',
        params: '',
        value: '',
        remark: '',
        permission: '',
    },
});

const { params, form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveConfigExec } = configApi.save.useApi(form);

watch(visible, () => {
    if (!visible.value) {
        return;
    }

    if (props.data) {
        state.form = { ...(props.data as any) };
        if (state.form.params) {
            state.params = JSON.parse(state.form.params);
        } else {
            state.params = [];
        }
    } else {
        state.form = { permission: 'all' } as any;
        state.params = [];
    }

    if (state.form.permission != 'all') {
        const accounts = state.form.permission.split(',');
        state.permissionAccount = accounts.slice(0, accounts.length - 1);
    } else {
        state.permissionAccount = [];
    }
});

const onCancel = () => {
    visible.value = false;
    // 若父组件有取消事件，则调用
    emit('cancel');
    state.permissionAccount = [];
};

const getAccount = (username: any) => {
    if (username) {
        accountApi.list.request({ username }).then((res) => {
            state.accounts = res.list;
        });
    }
};

const onConfirm = async () => {
    await useI18nFormValidate(configFormRef);
    if (state.params) {
        state.form.params = JSON.stringify(state.params);
    }
    if (state.permissionAccount.length > 0) {
        state.form.permission = state.permissionAccount.join(',') + ',';
    } else {
        state.form.permission = 'all';
    }

    await saveConfigExec();
    emit('val-change', state.form);
    onCancel();
};
</script>
<style lang="scss"></style>
