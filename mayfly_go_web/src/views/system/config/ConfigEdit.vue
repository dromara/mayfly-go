<template>
    <div>
        <el-dialog :title="title" v-model="dvisible" :show-close="false" :before-close="cancel" width="900px" :destroy-on-close="true">
            <el-form ref="configForm" :model="form" label-width="auto">
                <el-form-item prop="name" label="配置项" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>
                <el-form-item prop="key" label="配置key" required>
                    <el-input :disabled="form.id != null" v-model="form.key"></el-input>
                </el-form-item>
                <el-form-item prop="permission" label="权限">
                    <el-select
                        style="width: 100%"
                        remote
                        :remote-method="getAccount"
                        v-model="state.permissionAccount"
                        filterable
                        multiple
                        placeholder="请输入账号模糊搜索并选择"
                    >
                        <el-option v-for="item in state.accounts" :key="item.id" :label="`${item.username} [${item.name}]`" :value="item.username"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item label="配置项" class="w100">
                    <dynamic-form-edit v-model="params" />
                </el-form-item>

                <el-form-item label="备注">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { configApi, accountApi } from '../api';
import { DynamicFormEdit } from '@/components/dynamic-form';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const configForm: any = ref(null);

const state = reactive({
    dvisible: false,
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

const { dvisible, params, form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveConfigExec } = configApi.save.useApi(form);

watch(props, (newValue: any) => {
    state.dvisible = newValue.visible;
    if (!state.dvisible) {
        return;
    }

    if (newValue.data) {
        state.form = { ...newValue.data };
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

const cancel = () => {
    // 更新父组件visible prop对应的值为false
    emit('update:visible', false);
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

const btnOk = async () => {
    configForm.value.validate(async (valid: boolean) => {
        if (valid) {
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
            cancel();
        }
    });
};
</script>
<style lang="scss"></style>
