<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-loading="saveBtnLoading" @confirm="onConfirm" @cancel="emit('cancel')">
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- 认证信息表格编辑 -->
            <template #authCerts="{ form }">
                <ResourceAuthCertTableEdit
                    v-model="form.authCerts"
                    :resource-code="form.code"
                    :resource-type="TagResourceTypeEnum.Machine.value"
                    :test-conn-btn-loading="testConnBtnLoading"
                    @test-conn="onTestConn(form, $event)"
                />
            </template>

            <!-- SSH 隧道 -->
            <template #sshTunnelMachineId="{ form }">
                <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { computed, useTemplateRef, type PropType } from 'vue';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { machineApi } from './api';
import { MachineProtocolEnum } from './enums';
import type { MachineVO, MachineForm, MachineAuthCert } from './types';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    machine: {
        type: Object as PropType<MachineVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；group 三段分组，tagCodePaths/authCerts/sshTunnel 走插槽） */
const items: AutoFormItem[] = [
    { type: 'group', label: 'common.basic' },
    { prop: 'tagCodePaths', label: 'tag.relateTag' },
    { prop: 'name', label: 'common.name', required: true },
    {
        prop: 'protocol',
        label: 'machine.protocol',
        type: 'radio',
        enums: MachineProtocolEnum,
        required: true,
        // 切换协议时联动默认端口
        onChange: (val: unknown, form: AutoFormData) => {
            const v = val as number;
            (form as MachineForm).port = v == MachineProtocolEnum.Ssh.value ? 22 : v == MachineProtocolEnum.Rdp.value ? 3389 : 5901;
        },
    },
    { prop: 'ip', label: 'ip', required: true, rules: [Rules.requiredInput('machine.ipAndPort')], span: 17 },
    { prop: 'port', label: 'machine.port', type: 'number', span: 7 },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { type: 'group', label: 'common.account' },
    { prop: 'authCerts', type: 'custom' },
    { type: 'group', label: 'common.other' },
    { prop: 'enableRecorder', label: 'machine.terminalPlayback', type: 'switch', props: { activeValue: 1, inactiveValue: -1 } },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
    { prop: 'extra.ciphers', label: 'machine.ciphers', placeholder: 'machine.multiValuePlaceholder' },
    { prop: 'extra.keyExchanges', label: 'machine.keyExchanges', placeholder: 'machine.multiValuePlaceholder' },
];

const defaultForm: MachineForm = {
    id: null,
    code: '',
    tagPath: '',
    ip: null,
    port: 22,
    protocol: MachineProtocolEnum.Ssh.value,
    name: null,
    authCerts: [],
    tagCodePaths: [],
    remark: '',
    sshTunnelMachineId: null as number | null,
    enableRecorder: -1,
    extra: { ciphers: '', keyExchanges: '' },
};

/** 传给 AutoFormDrawer 的回填数据：新增时应用 defaultForm；编辑时兜底 authCerts/extra 为空（深拷贝由组件内部完成） */
const editData = computed<AutoFormData | null>(() => {
    const machine = props.machine;
    if (!machine) {
        return { ...defaultForm, authCerts: [], tagCodePaths: [] } as unknown as AutoFormData;
    }
    return {
        ...machine,
        authCerts: machine.authCerts || [],
        extra: (machine.extra as Record<string, string>) || {},
    } as unknown as AutoFormData;
});

const { isFetching: testConnBtnLoading, execute: testConnExec } = machineApi.testConn.useApi();
const { isFetching: saveBtnLoading, execute: saveMachineExec } = machineApi.saveMachine.useApi();

const getReqForm = (form: MachineForm) => {
    const reqForm = { ...form } as MachineForm & Record<string, unknown>;
    if (!form.sshTunnelMachineId || form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const onTestConn = async (rawForm: AutoFormData, authCert: MachineAuthCert) => {
    await useI18nFormValidate(drawerRef);

    const submitForm = getReqForm(rawForm as MachineForm);
    submitForm.authCerts = [authCert];
    await testConnExec(submitForm);
    Msg.success('machine.connSuccess');
};

const onConfirm = async (rawForm: AutoFormData) => {
    const form = rawForm as MachineForm;
    if ((form.authCerts || []).length == 0) {
        Msg.error('machine.noAcErrMsg');
        return;
    }

    const submitForm = getReqForm(form);
    await saveMachineExec(submitForm);
    Msg.saveSuccess();
    emit('val-change', submitForm);
    dialogVisible.value = false;
};
</script>
<style lang="scss"></style>
