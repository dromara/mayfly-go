<template>
    <div>
        <auto-form-drawer
            ref="drawerRef"
            v-model:visible="dialogVisible"
            :title="title"
            :items="items"
            :data="editData"
            size="40%"
            :confirm-api="onConfirm"
            @opened="onOpened"
            @cancel="emit('cancel')"
        >
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
import { useSshTunnelTransform } from '@/hooks/useResourceForm';
import { computed, ref, useTemplateRef, type PropType } from 'vue';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { machineApi } from './api';
import { MachineProtocolEnum } from './enums';
import type { MachineVO, MachineForm, MachineAuthCert } from './types';

const props = defineProps({
    data: {
        type: Object as PropType<MachineVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

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

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData | null>(() => {
    const machine = props.data;
    if (!machine) {
        return { ...defaultForm, authCerts: [], tagCodePaths: [] } as unknown as AutoFormData;
    }
    return {
        ...machine,
        authCerts: machine.authCerts || [],
        extra: (machine.extra as Record<string, string>) || {},
    } as unknown as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用（提交组装基于它） */
const internalForm = ref<AutoFormData>({});

const onOpened = (form: AutoFormData) => {
    internalForm.value = form;
};

const submitForm = useSshTunnelTransform(
    computed(() => internalForm.value)
);

const { isFetching: testConnBtnLoading, execute: testConnExec } = machineApi.testConn.useApi();
const { execute: saveMachineExec } = machineApi.saveMachine.useApi();

const onTestConn = async (rawForm: AutoFormData, authCert: MachineAuthCert) => {
    await useI18nFormValidate(drawerRef);
    await testConnExec({
        ...submitForm.value,
        authCerts: [authCert],
    });
    Msg.success('machine.connSuccess');
};

// confirmApi 提交动作：前置校验失败抛错中止（组件保持抽屉打开可修正）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async (rawForm: AutoFormData) => {
    const form = rawForm as MachineForm;
    if ((form.authCerts || []).length == 0) {
        Msg.error('machine.noAcErrMsg');
        throw new Error('authCerts required');
    }
    await saveMachineExec(submitForm.value);
    emit('val-change', submitForm.value);
};
</script>
<style lang="scss"></style>
