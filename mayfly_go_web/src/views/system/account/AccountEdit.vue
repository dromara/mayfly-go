<template>
    <div class="account-dialog">
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :show-close="false" width="35%" :destroy-on-close="true">
            <el-form :model="form" ref="accountForm" :rules="rules" label-width="85px">
                <el-form-item prop="username" label="用户名:" required>
                    <el-input :disabled="edit" v-model.trim="form.username" placeholder="请输入账号用户名，密码默认与账号名一致" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item v-if="edit" prop="password" label="密码:" required>
                    <el-input type="password" v-model.trim="form.password" placeholder="请输入密码" autocomplete="new-password"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, watch, defineComponent, ref } from 'vue';
import { accountApi } from '../api';
import { ElMessage } from 'element-plus';

export default defineComponent({
    name: 'AccountEdit',
    props: {
        visible: {
            type: Boolean,
        },
        account: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const accountForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            edit: false,
            form: {
                id: null,
                username: null,
                password: null,
                repassword: null,
            },
            btnLoading: false,
            rules: {
                username: [
                    {
                        required: true,
                        message: '请输入用户名',
                        trigger: ['change', 'blur'],
                    },
                ],
                // password: [
                //     {
                //         required: true,
                //         message: '请输入密码',
                //         trigger: ['change', 'blur'],
                //     },
                // ],
            },
        });

        watch(props, (newValue) => {
            if (newValue.account) {
                state.form = { ...newValue.account };
                state.edit = true;
            } else {
                state.form = {} as any;
            }
            state.dialogVisible = newValue.visible;
        });

        const btnOk = async () => {
            accountForm.value.validate((valid: boolean) => {
                if (valid) {
                    accountApi.save.request(state.form).then(() => {
                        ElMessage.success('操作成功');
                        emit('val-change', state.form);
                        state.btnLoading = true;
                        setTimeout(() => {
                            state.btnLoading = false;
                        }, 1000);
                        //重置表单域
                        accountForm.value.resetFields();
                        state.form = {} as any;
                    });
                } else {
                    ElMessage.error('表单填写有误');
                    return false;
                }
            });
        };

        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
        };

        return {
            ...toRefs(state),
            accountForm,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
