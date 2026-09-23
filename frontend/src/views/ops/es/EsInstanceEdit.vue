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
                <el-button @click="onTestConn(null)" type="success" v-if="(internalForm?.authCerts?.length ?? 0) <= 0">{{ $t('ac.testConn') }}</el-button>
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
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { useSshTunnelTransform } from '@/hooks/useResourceForm';
import { computed, useTemplateRef, type PropType } from 'vue';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
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

/** 表单声明（defineFormItems<EsInstanceForm>，渲染 + 校验唯一数据源；group 分组容器 + tagCodePaths/authCerts/sshTunnel 走插槽） */
const items = defineFormItems<EsInstanceForm>([
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
]);

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
    sshTunnelMachineId: null,
    authCerts: [],
    tagCodePaths: [],
};

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<EsInstanceForm>(() => {
    return props.data ? { ...DefaultForm, ...props.data, authCerts: props.data.authCerts || [] } : { ...DefaultForm, authCerts: [], tagCodePaths: [] };
});

// 宿主内部表单在 @opened 接管（测试连接回写 version 与提交均基于它；form 供模板按凭证数决定测试入口是否展示）
const { form: internalForm, onOpened, requireForm } = useAutoFormModel<EsInstanceForm>();

const submitForm = useSshTunnelTransform(computed(requireForm));

const { isFetching: saveBtnLoading, execute: saveInstanceExec, data: saveInstanceRes } = esApi.saveInstance.useApi();
const { isFetching: testConnBtnLoading, execute: testConnExec, data: testConnRes } = esApi.testConn.useApi();

const onTestConn = async (authCert: MachineAuthCert | null) => {
    await useI18nFormValidate(drawerRef);
    const form = submitForm.value;
    if (authCert) {
        form.authCerts = [authCert];
    }
    await testConnExec(form);
    requireForm().version = testConnRes.value?.version?.number;
    Msg.success('es.connSuccess');
};

// confirmApi 提交动作
const onConfirm = async () => {
    const form = requireForm();
    if (!form.version) {
        Msg.warning('es.shouldTestConn');
        // 宿主以「抛错」为取消语义：只 return 会被当成提交成功而弹「保存成功」并关掉抽屉
        throw new Error('es version not verified');
    }
    await saveInstanceExec(submitForm.value);
    form.id = saveInstanceRes.value;
    emit('val-change', form);
};
</script>
<style lang="scss"></style>
