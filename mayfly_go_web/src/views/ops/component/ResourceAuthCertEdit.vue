<template>
    <div class="auth-cert-manage">
        <el-table :data="authCerts" max-height="180" stripe style="width: 100%" size="small">
            <el-table-column min-wdith="120px">
                <template #header>
                    <el-button class="ml0" type="primary" circle size="small" icon="Plus" @click="edit(null)"> </el-button>
                </template>
                <template #default="scope">
                    <el-link @click="edit(scope.row)" type="primary" icon="edit"></el-link>
                    <el-link class="ml5" v-auth="'machine:file:del'" type="danger" @click="deleteRow(scope.$index)" icon="delete"></el-link>
                </template>
            </el-table-column>

            <el-table-column prop="name" label="名称" min-width="100px"> </el-table-column>
            <el-table-column prop="username" label="用户名" min-width="120px" show-overflow-tooltip> </el-table-column>
            <el-table-column prop="ciphertextType" label="密文类型" width="100px">
                <template #default="scope">
                    <EnumTag :value="scope.row.ciphertextType" :enums="AuthCertCiphertextTypeEnum" />
                </template>
            </el-table-column>
            <el-table-column prop="type" label="凭证类型" width="100px">
                <template #default="scope">
                    <EnumTag :value="scope.row.type" :enums="AuthCertTypeEnum" />
                </template>
            </el-table-column>
        </el-table>

        <el-dialog title="凭证保存" v-model="state.dvisible" :show-close="false" width="500px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form ref="acForm" :model="state.form" label-width="auto">
                <el-form-item prop="type" label="凭证类型" required>
                    <el-select style="width: 100%" v-model="form.type" placeholder="请选择凭证类型">
                        <el-option v-for="item in AuthCertTypeEnum" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="ciphertextType" label="密文类型" required>
                    <el-select style="width: 100%" v-model="form.ciphertextType" placeholder="请选择密文类型">
                        <el-option v-for="item in AuthCertCiphertextTypeEnum" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="name" label="名称" required>
                    <el-input :disabled="form.id" v-model="form.name"></el-input>
                </el-form-item>
                <el-form-item prop="username" label="用户名">
                    <el-input v-model="form.username"></el-input>
                </el-form-item>
                <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.Password.value" prop="ciphertext" label="密码">
                    <el-input type="password" show-password clearable v-model.trim="form.ciphertext" placeholder="请输入密码" autocomplete="new-password">
                    </el-input>
                </el-form-item>
                <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value" prop="ciphertext" label="秘钥">
                    <el-input type="textarea" :rows="5" v-model="form.ciphertext" placeholder="请将私钥文件内容拷贝至此"> </el-input>
                </el-form-item>
                <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value" prop="passphrase" label="秘钥密码">
                    <el-input type="password" v-model="form.extra.passphrase"> </el-input>
                </el-form-item>
                <el-form-item label="备注">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelEdit">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, toRefs } from 'vue';
import { AuthCertTypeEnum, AuthCertCiphertextTypeEnum } from '../tag/enums';
import { resourceAuthCertApi } from '../tag/api';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { ElMessage } from 'element-plus';

const props = defineProps({
    resourceType: { type: Number },
    resourceCode: { type: String },
});

const authCerts = defineModel<any>('modelValue', { required: true, default: [] });

const acForm: any = ref(null);

const DefaultForm = {
    id: null,
    name: '',
    username: '',
    ciphertextType: AuthCertCiphertextTypeEnum.Password.value,
    type: AuthCertTypeEnum.Private.value,
    ciphertext: '',
    extra: {} as any,
    remark: '',
};
const state = reactive({
    dvisible: false,
    params: [] as any,
    form: { ...DefaultForm },
    btnLoading: false,
    edit: false,
});

const { form, btnLoading } = toRefs(state);

onMounted(() => {
    getAuthCerts();
});

const getAuthCerts = async () => {
    if (!props.resourceCode || !props.resourceType) {
        return;
    }
    const res = await resourceAuthCertApi.listByQuery.request({
        resourceCode: props.resourceCode,
        resourceType: props.resourceType,
        pageNum: 1,
        pageSize: 100,
    });
    authCerts.value = res.list?.reverse() || [];
};

const edit = (form: any) => {
    if (form) {
        state.form = form;
        state.edit = true;
    }
    state.dvisible = true;
};

const deleteRow = (idx: any) => {
    authCerts.value.splice(idx, 1);
};

const cancelEdit = () => {
    state.dvisible = false;
    setTimeout(() => {
        state.form = { ...DefaultForm };
    }, 300);
};

const btnOk = async () => {
    acForm.value.validate(async (valid: boolean) => {
        if (valid) {
            const isEdit = state.form.id;
            if (isEdit || state.edit) {
                cancelEdit();
                return;
            }

            if (authCerts.value?.filter((x: any) => x.username == state.form.username || x.name == state.form.name).length > 0) {
                ElMessage.error('该名称或用户名已存在于该账号列表中');
                return;
            }
            const res = await resourceAuthCertApi.listByQuery.request({
                name: state.form.name,
                pageNum: 1,
                pageSize: 100,
            });
            if (res.total) {
                ElMessage.error('该授权凭证名称已存在');
                return;
            }

            authCerts.value.push(state.form);
            cancelEdit();
        }
    });
};
</script>
<style lang="scss"></style>
