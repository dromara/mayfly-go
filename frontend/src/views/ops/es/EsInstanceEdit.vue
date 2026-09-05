<template>
    <div>
        <el-drawer
            :append-to-body="false"
            :title="title"
            v-model="dialogVisible"
            :before-close="onCancel"
            :destroy-on-close="true"
            :close-on-click-modal="false"
            size="40%"
        >
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <auto-form ref="dbFormRef" v-model="form" :items="items" label-width="auto">
                <!-- 关联标签 -->
                <template #tagCodePaths>
                    <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
                </template>

                <!-- 认证信息表格编辑 -->
                <template #authCerts>
                    <ResourceAuthCertTableEdit
                        v-model="form.authCerts"
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.EsInstance.value"
                        :test-conn-btn-loading="testConnBtnLoading"
                        @test-conn="onTestConn"
                        :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.PrivateKey.value]"
                    />
                </template>

                <!-- SSH 隧道 -->
                <template #sshTunnelMachineId>
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </template>
            </auto-form>

            <template #footer>
                <el-button @click="onTestConn(null)" type="success" v-if="(form.authCerts?.length ?? 0) <= 0">{{ $t('ac.testConn') }}</el-button>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, toRefs, useTemplateRef, watchEffect, type PropType } from 'vue';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import { esApi } from './api';
import type { EsInstance } from './types';
import type { MachineAuthCert } from '@/views/ops/machine/types';

/** ES 实例编辑表单类型 */
interface EsInstanceForm extends Omit<Partial<EsInstance>, 'id' | 'name' | 'sshTunnelMachineId'> {
    id?: number | null;
    name?: string | null;
    protocol: string;
    sshTunnelMachineId?: number | null;
    tagCodePaths?: string[];
}


const props = defineProps({
    data: {
        type: Object as PropType<EsInstance | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；group 分组容器 + tagCodePaths/authCerts/sshTunnel 走插槽） */
const items: AutoFormItem[] = [
    { type: 'group', label: 'common.basic' },
    { prop: 'tagCodePaths', label: 'tag.relateTag' },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'version', label: 'common.version', disabled: true },
    { prop: 'protocol', label: 'es.protocol', type: 'select', options: [{ value: 'http', label: 'http' }, { value: 'https', label: 'https' }], placeholder: 'http' },
    { prop: 'host', label: 'Host', required: true, span: 18 },
    { prop: 'port', label: 'Port', type: 'number', span: 6, placeholder: 'es.port' },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { type: 'group', label: 'common.account' },
    { prop: 'authCerts', label: 'db.acName', type: 'custom' },
    { type: 'group', label: 'common.other' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
];

const dbFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('dbFormRef');

const DefaultForm: EsInstanceForm = {
    id: null,
    code: '',
    name: null,
    protocol: 'http',
    host: '',
    version: '',
    port: 9200,
    remark: '',
    sshTunnelMachineId: null as number | null,
    authCerts: [],
    tagCodePaths: [],
};

const state = reactive({
    form: DefaultForm,
});

const { form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveInstanceExec, data: saveInstanceRes } = esApi.saveInstance.useApi();
const { isFetching: testConnBtnLoading, execute: testConnExec, data: testConnRes } = esApi.testConn.useApi();

watchEffect(() => {
    if (!dialogVisible.value) {
        return;
    }
    const dbInst = props.data as EsInstance | false | undefined;
    if (dbInst) {
        state.form = { ...dbInst };
    } else {
        state.form = { ...DefaultForm };
        state.form.authCerts = [];
    }
});

const getReqForm = () => {
    const reqForm: Record<string, unknown> = { ...state.form };
    reqForm.selectAuthCert = null;
    reqForm.tags = null;
    if (!state.form.sshTunnelMachineId) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async (authCert: MachineAuthCert | null) => {
    await useI18nFormValidate(dbFormRef);
    const submitForm = getReqForm();
    if (authCert) {
        submitForm.authCerts = [authCert];
    }
    await testConnExec(submitForm);
    state.form.version = testConnRes.value?.version?.number;
    Msg.success('es.connSuccess');
};

const onConfirm = async () => {
    if (!state.form.version) {
        Msg.warning('es.shouldTestConn');
        return;
    }

    await useI18nFormValidate(dbFormRef);
    await saveInstanceExec(getReqForm());
    Msg.saveSuccess();
    state.form.id = saveInstanceRes.value;
    emit('val-change', state.form);
    onCancel();
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
