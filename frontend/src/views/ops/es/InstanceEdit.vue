<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="dbForm" :rules="rules" label-width="auto">
                <el-divider content-position="left">{{ t('common.basic') }}</el-divider>

                <el-form-item ref="tagSelectRef" prop="tagCodePaths" :label="t('tag.relateTag')">
                    <tag-tree-select
                        multiple
                        @change-tag="
                            (paths: any) => {
                                form.tagCodePaths = paths;
                                tagSelectRef.validate();
                            }
                        "
                        :select-tags="form.tagCodePaths"
                    />
                </el-form-item>

                <el-form-item prop="name" :label="t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>

                <el-form-item prop="version" :label="t('common.version')">
                    <el-input v-model.trim="form.version" auto-complete="off" disabled></el-input>
                </el-form-item>

                <el-form-item prop="host" label="Host" required>
                    <el-col :span="18">
                        <el-input v-model.trim="form.host" auto-complete="off"></el-input>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="5">
                        <el-input type="number" v-model.number="form.port" :placeholder="t('es.port')"></el-input>
                    </el-col>
                </el-form-item>

                <el-form-item prop="remark" :label="t('common.remark')">
                    <el-input v-model="form.remark" auto-complete="off" type="textarea"></el-input>
                </el-form-item>

                <el-divider content-position="left">{{ t('common.account') }}</el-divider>
                <div>
                    <ResourceAuthCertTableEdit
                        v-model="form.authCerts"
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.EsInstance.value"
                        :test-conn-btn-loading="testConnBtnLoading"
                        @test-conn="testConn"
                        :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.PrivateKey.value]"
                    />
                </div>

                <el-divider content-position="left">{{ t('common.other') }}</el-divider>

                <el-form-item prop="sshTunnelMachineId" :label="t('machine.sshTunnel')">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="testConn(null)" type="success" v-if="form.authCerts?.length <= 0">{{ t('ac.testConn') }}</el-button>
                <el-button @click="cancel()">{{ t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, watchEffect } from 'vue';
import { esApi } from './api';
import { ElMessage } from 'element-plus';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { useI18nFormValidate, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';
import { Rules } from '@/common/rule';

const { t } = useI18n();

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

const rules = {
    tagCodePaths: [Rules.requiredSelect('tag.relateTag')],
    name: [Rules.requiredInput('common.name')],
    type: [Rules.requiredSelect('common.type')],
    host: [Rules.requiredInput('Host:Port')],
};

const dbForm: any = ref(null);
const tagSelectRef: any = ref(null);

const DefaultForm = {
    id: null,
    code: '',
    name: null,
    host: '',
    version: '',
    port: 9200,
    remark: '',
    sshTunnelMachineId: null as any,
    authCerts: [],
    tagCodePaths: [],
};

const state = reactive({
    dialogVisible: false,
    form: DefaultForm,
    submitForm: {} as any,
});

const { dialogVisible, form, submitForm } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveInstanceExec, data: saveInstanceRes } = esApi.saveInstance.useApi(submitForm);
const { isFetching: testConnBtnLoading, execute: testConnExec, data: testConnRes } = esApi.testConn.useApi<any>(submitForm);

watchEffect(() => {
    state.dialogVisible = props.visible;
    if (!state.dialogVisible) {
        return;
    }
    const dbInst: any = props.data;
    if (dbInst) {
        state.form = { ...dbInst };
        state.form.tagCodePaths = dbInst.tags.map((t: any) => t.codePath) || [];
    } else {
        state.form = { ...DefaultForm };
        state.form.authCerts = [];
    }
});

const getReqForm = async () => {
    const reqForm: any = { ...state.form };
    reqForm.selectAuthCert = null;
    reqForm.tags = null;
    if (!state.form.sshTunnelMachineId) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const testConn = async (authCert: any) => {
    await useI18nFormValidate(dbForm);
    state.submitForm = await getReqForm();
    if (authCert) {
        state.submitForm.authCerts = [authCert];
    }
    await testConnExec();
    state.form.version = testConnRes.value.version.number;
    ElMessage.success(t('es.connSuccess'));
};

const btnOk = async () => {
    if (!state.form.version) {
        ElMessage.warning(t('es.shouldTestConn'));
        return;
    }

    await useI18nFormValidate(dbForm);
    state.submitForm = await getReqForm();
    await saveInstanceExec();
    useI18nSaveSuccessMsg();
    state.form.id = saveInstanceRes as any;
    emit('val-change', state.form);
    cancel();
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
