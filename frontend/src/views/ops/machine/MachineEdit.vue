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
                    @test-conn="onTestConn($event)"
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
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { useSshTunnelTransform } from '@/hooks/useResourceForm';
import { computed, useTemplateRef, type PropType } from 'vue';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { machineApi } from './api';
import { getProtocolDefaultPort, MACHINE_DEFAULT_PORT, MachineProtocolEnum } from './enums';
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

/** 表单声明（defineFormItems<MachineForm>：渲染 + 校验唯一数据源，回调形参直接是业务表单；group 三段分组，tagCodePaths/authCerts/sshTunnel 走插槽） */
const items = defineFormItems<MachineForm>([
    { type: 'group', label: 'common.basic' },
    { prop: 'tagCodePaths', label: 'tag.relateTag' },
    { prop: 'name', label: 'common.name', required: true },
    {
        prop: 'protocol',
        label: 'machine.protocol',
        type: 'radio',
        enums: MachineProtocolEnum,
        required: true,
        // 切换协议时联动该协议的默认端口（端口值与协议枚举同源，见 enums.ts）
        onChange: (value, form) => {
            const port = getProtocolDefaultPort(value);
            if (port != null) {
                form.port = port;
            }
        },
    },
    { prop: 'ip', label: 'ip', required: true, rules: [Rules.requiredInput('machine.ipAndPort')], span: 17 },
    { prop: 'port', label: 'machine.port', type: 'number', span: 7 },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    { type: 'group', label: 'common.account' },
    // 无凭证则机器无法连接，后端也会拒绝保存；作为字段规则声明，不占提交环节
    { prop: 'authCerts', type: 'custom', validate: (value) => (Array.isArray(value) && value.length > 0 ? true : 'machine.noAcErrMsg') },
    { type: 'group', label: 'common.other' },
    { prop: 'enableRecorder', label: 'machine.terminalPlayback', type: 'switch', props: { activeValue: 1, inactiveValue: -1 } },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
    { prop: 'extra.ciphers', label: 'machine.ciphers', placeholder: 'machine.multiValuePlaceholder' },
    { prop: 'extra.keyExchanges', label: 'machine.keyExchanges', placeholder: 'machine.multiValuePlaceholder' },
]);

const defaultForm: MachineForm = {
    id: null,
    code: '',
    tagPath: '',
    ip: null,
    port: MACHINE_DEFAULT_PORT,
    protocol: MachineProtocolEnum.Ssh.value,
    name: null,
    authCerts: [],
    tagCodePaths: [],
    remark: '',
    sshTunnelMachineId: null,
    enableRecorder: -1,
    extra: { ciphers: '', keyExchanges: '' },
};

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<MachineForm>(() => {
    const machine = props.data;
    if (!machine) {
        return { ...defaultForm, authCerts: [], tagCodePaths: [] };
    }
    // 以默认值为底再覆盖行数据：tagCodePaths 等表单字段由插槽控件按 code 自行 hydrate，行数据不携带
    return {
        ...defaultForm,
        ...machine,
        authCerts: machine.authCerts || [],
        extra: machine.extra || {},
    };
});

// 宿主内部表单在 @opened 接管（抛出的即宿主持有的同一对象）；提交载荷在其基础上归一 SSH 隧道字段
const { onOpened, requireForm } = useAutoFormModel<MachineForm>();

const submitForm = useSshTunnelTransform(computed(requireForm));

const { isFetching: testConnBtnLoading, execute: testConnExec } = machineApi.testConn.useApi();
const { execute: saveMachineExec } = machineApi.saveMachine.useApi();

const onTestConn = async (authCert: MachineAuthCert) => {
    await useI18nFormValidate(drawerRef);
    await testConnExec({
        ...submitForm.value,
        authCerts: [authCert],
    });
    Msg.success('machine.connSuccess');
};

// confirmApi 提交动作：凭证非空已由 authCerts 字段的 validate 声明，校验不通过时宿主直接中止提交并保留抽屉
const onConfirm = async () => {
    await saveMachineExec(submitForm.value);
    emit('val-change', submitForm.value);
};
</script>
<style lang="scss"></style>
