<template>
    <div class="auth-cert-manage">
        <el-table :data="authCerts" :max-height="180" stripe size="small">
            <el-table-column min-wdith="120px">
                <template #header>
                    <el-button v-auth="'authcert:save'" class="ml0" type="primary" circle size="small" icon="Plus" @click="edit(null)"> </el-button>
                </template>
                <template #default="scope">
                    <el-button v-auth="'authcert:save'" @click="edit(scope.row, scope.$index)" type="primary" icon="edit" link></el-button>
                    <el-button class="ml-0.5!" v-auth="'authcert:del'" type="danger" @click="deleteRow(scope.$index)" icon="delete" link></el-button>

                    <el-button
                        :title="$t('ac.testConn')"
                        :loading="props.testConnBtnLoading && scope.$index == state.idx"
                        :disabled="props.testConnBtnLoading"
                        class="ml-0.5!"
                        type="success"
                        @click="testConn(scope.row, scope.$index)"
                        icon="Link"
                        link
                    ></el-button>
                </template>
            </el-table-column>

            <el-table-column prop="username" :label="$t('common.username')" min-width="120px" show-overflow-tooltip> </el-table-column>
            <el-table-column prop="ciphertextType" :label="$t('ac.ciphertextType')" width="100px">
                <template #default="scope">
                    <EnumTag :value="scope.row.ciphertextType" :enums="AuthCertCiphertextTypeEnum" />
                </template>
            </el-table-column>
            <el-table-column prop="type" :label="$t('ac.credentialType')" width="100px">
                <template #default="scope">
                    <EnumTag :value="scope.row.type" :enums="AuthCertTypeEnum" />
                </template>
            </el-table-column>
            <el-table-column prop="remark" :label="$t('common.remark')" show-overflow-tooltip width="120px"> </el-table-column>
        </el-table>

        <ResourceAuthCertEdit
            v-model:visible="state.dvisible"
            :auth-cert="state.form"
            @confirm="btnOk"
            @cancel="cancelEdit"
            :disable-type="[AuthCertTypeEnum.Public.value]"
            :disable-ciphertext-type="props.disableCiphertextType"
        />
    </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive } from 'vue';
import { AuthCertTypeEnum, AuthCertCiphertextTypeEnum } from '../tag/enums';
import { resourceAuthCertApi } from '../tag/api';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import ResourceAuthCertEdit from './ResourceAuthCertEdit.vue';
import { ElMessage } from 'element-plus';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    resourceType: { type: Number },
    resourceCode: { type: String },
    disableCiphertextType: {
        type: Array,
    },
    testConnBtnLoading: { type: Boolean },
});

const authCerts = defineModel<any>('modelValue', { required: true, default: [] });
const emit = defineEmits(['testConn']);

const state = reactive({
    dvisible: false,
    params: [] as any,
    form: {},
    idx: -1,
});

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

const testConn = async (row: any, idx: number) => {
    state.idx = idx;
    emit('testConn', row);
};

const edit = (form: any, idx = -1) => {
    state.idx = idx;
    if (form) {
        state.form = form;
    } else {
        state.form = { ciphertextType: AuthCertCiphertextTypeEnum.Password.value, type: AuthCertTypeEnum.Private.value, extra: {} };
    }
    state.dvisible = true;
};

const deleteRow = (idx: any) => {
    authCerts.value.splice(idx, 1);
};

const cancelEdit = () => {
    state.dvisible = false;
};

const btnOk = async (authCert: any) => {
    const isEdit = authCert.id;

    if (isEdit || state.idx >= 0) {
        authCerts.value[state.idx] = authCert;
        cancelEdit();
        return;
    }

    if (authCerts.value?.filter((x: any) => x.username == authCert.username).length > 0) {
        ElMessage.error(t('ac.usernameExist'));
        return;
    }

    authCerts.value.push(authCert);
    cancelEdit();
};
</script>
<style lang="scss"></style>
