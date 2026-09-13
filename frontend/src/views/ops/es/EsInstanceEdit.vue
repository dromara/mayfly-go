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
            @submitted="emit('cancel')"
            @cancel="emit('cancel')"
        >
            <template #footer>
                <el-button @click="onTestConn(null)" type="success" v-if="(internalForm.authCerts?.length ?? 0) <= 0">{{ $t('ac.testConn') }}</el-button>
                <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="drawerRef?.submitting" @click="drawerRef?.submit()">{{ $t('common.confirm') }}</el-button>
            </template>

            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- 认证信息表格编辑 -->
            <template #authCerts="{ form }">
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
            <template #sshTunnelMachineId="{ form }">
                <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { useSshTunnelTransform } from '@/hooks/useResourceForm';
import { computed, ref, useTemplateRef, type PropType } from 'vue';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { AuthCertCiphertextTypeEnum } from '../tag/enums';
import { esApi } from './api';
import type { EsInstance } from './types';
import type { MachineAuthCert } from '@/views/ops/machine/types';

/** ES 实例编辑表单类型 (允许 null 的字段重定义) */
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

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown>; submitting: boolean; submit: () => Promise<void> }>('drawerRef');

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

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData>(() => {
    const dbInst = props.data as EsInstance | false | undefined;
    if (dbInst) {
        return { ...dbInst, authCerts: dbInst.authCerts || [] } as AutoFormData;
    }
    return { ...DefaultForm, authCerts: [], tagCodePaths: [] } as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用 */
const internalForm = ref<AutoFormData>({});

const onOpened = (form: AutoFormData) => {
    internalForm.value = form;
};

const submitForm = useSshTunnelTransform(
    computed(() => internalForm.value)
);

const { isFetching: saveBtnLoading, execute: saveInstanceExec, data: saveInstanceRes } = esApi.saveInstance.useApi();
const { isFetching: testConnBtnLoading, execute: testConnExec, data: testConnRes } = esApi.testConn.useApi();

const onTestConn = async (authCert: MachineAuthCert | null) => {
    await useI18nFormValidate(drawerRef);
    const form = submitForm.value;
    if (authCert) {
        form.authCerts = [authCert];
    }
    await testConnExec(form);
    internalForm.value.version = testConnRes.value?.version?.number;
    Msg.success('es.connSuccess');
};

// confirmApi 提交动作
const onConfirm = async () => {
    if (!internalForm.value.version) {
        Msg.warning('es.shouldTestConn');
        return;
    }
    await saveInstanceExec(submitForm.value);
    internalForm.value.id = saveInstanceRes.value;
    emit('val-change', internalForm.value);
};
</script>
<style lang="scss"></style>
