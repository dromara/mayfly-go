<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-api="onConfirm" @submitted="emit('cancel')" @opened="onOpened" @cancel="emit('cancel')">
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>

            <!-- 认证信息表格编辑 -->
            <template #authCerts="{ form }">
                <ResourceAuthCertTableEdit
                    v-model="form.authCerts"
                    :resource-code="form.code"
                    :resource-type="TagResourceTypeEnum.Milvus.value"
                    :test-conn-btn-loading="testConnBtnLoading"
                    @test-conn="testConn"
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
import { Msg } from '@/hooks/useI18n';
import TagTreeSelect from '@/views/ops/component/TagTreeSelect.vue';
import { computed, ref, useTemplateRef, type PropType } from 'vue';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import { milvusApi } from './api';
import type { Milvus } from './types';
import type { MachineAuthCert } from '@/views/ops/machine/types';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { AuthCertCiphertextTypeEnum } from '@/views/ops/tag/enums';
import ResourceAuthCertTableEdit from '@/views/ops/component/ResourceAuthCertTableEdit.vue';

/** Milvus 实例编辑表单类型 (允许 null 的字段重定义) */
interface MilvusForm extends Omit<Partial<Milvus>, 'id' | 'name' | 'sshTunnelMachineId'> {
    id?: number | null;
    name?: string | null;
    sshTunnelMachineId?: number | null;
    tagCodePaths?: string[];
    authCerts?: MachineAuthCert[];
}

const props = defineProps({
    milvus: {
        type: Object as PropType<Milvus | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['val-change', 'cancel']);

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；group 分组容器 + tagCodePaths/authCerts/sshTunnel 走插槽） */
const items: AutoFormItem[] = [
    { prop: 'tagCodePaths', label: 'tag.relateTag', required: true },
    { prop: 'name', label: 'common.name', required: true, placeholder: 'common.pleaseInput' },
    { prop: 'host', label: 'milvus.host', type: 'textarea', required: true, placeholder: 'milvus.connAddress' },
    { type: 'group', label: 'common.account' },
    { prop: 'authCerts', label: 'db.acName', type: 'custom' },
    { prop: 'database', label: 'milvus.database', placeholder: 'milvus.dbNamePlaceholder' },
    { prop: 'sshTunnelMachineId', label: 'machine.sshTunnel' },
];

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => Promise<unknown> }>('drawerRef');

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData>(() => {
    const milvusData = props.milvus as Milvus | false | undefined;
    if (milvusData) {
        return { ...milvusData, authCerts: milvusData.authCerts || [] } as AutoFormData;
    }
    return { database: 'default', sshTunnelMachineId: -1, authCerts: [] } as AutoFormData;
});

/** 抽屉打开后暂存的内部表单引用（提交组装与保存后回写 id 基于它） */
const internalForm = ref<AutoFormData>({});

const onOpened = (form: AutoFormData) => {
    internalForm.value = form;
};

const submitForm = computed(() => {
    const reqForm: Record<string, unknown> = { ...internalForm.value };
    const sshTunnelMachineId = internalForm.value.sshTunnelMachineId as number | null | undefined;
    if (!sshTunnelMachineId || sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
});

const { isFetching: testConnBtnLoading, execute: testConnExec } = milvusApi.testConn.useApi(submitForm);
const { execute: saveMilvusExec, data: saveMilvusRes } = milvusApi.save.useApi(submitForm);

const testConn = async (authCert: MachineAuthCert) => {
    await drawerRef.value?.validate();
    await testConnExec({
        ...submitForm.value,
        authCerts: [authCert],
    });
    Msg.success(('milvus.connSuccess'));
};

// confirmApi 提交动作（无参，从内部表单读取提交数据）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    await saveMilvusExec(submitForm.value);
    internalForm.value.id = saveMilvusRes.value;
    emit('val-change', internalForm.value);
};
</script>

<style lang="scss" scoped></style>
