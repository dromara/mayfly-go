<template>
    <div>
        <el-drawer :append-to-body="false" :title="title" v-model="dialogVisible" :before-close="onCancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="onCancel" />
            </template>

            <el-form :model="form" ref="milvusFormRef" :rules="rules" label-width="auto">
                <el-form-item prop="tagCodePaths" :label="$t('tag.relateTag')" required>
                    <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
                </el-form-item>
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" :placeholder="$t('common.pleaseInput')" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="host" :label="$t('milvus.host')" required>
                    <el-input v-model.trim="form.host" :placeholder="$t('milvus.connAddress')" auto-complete="off" type="textarea"></el-input>
                </el-form-item>

                <el-divider content-position="left">{{ $t('common.account') }}</el-divider>
                <div>
                    <ResourceAuthCertTableEdit
                        v-model="form.authCerts"
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.Milvus.value"
                        :test-conn-btn-loading="testConnBtnLoading"
                        @test-conn="testConn"
                        :disable-ciphertext-type="[AuthCertCiphertextTypeEnum.PrivateKey.value]"
                    />
                </div>
                <el-divider content-position="left" />

                <el-form-item prop="database" :label="$t('milvus.database')">
                    <el-input v-model.trim="form.database" :placeholder="$t('milvus.dbNamePlaceholder')"></el-input>
                </el-form-item>
                <el-form-item prop="sshTunnelMachineId" :label="$t('machine.sshTunnel')">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { Msg } from '@/hooks/useI18n';
import TagTreeSelect from '@/views/ops/component/TagTreeSelect.vue';
import { computed, reactive, toRefs, useTemplateRef, watch, type PropType } from 'vue';
import type { FormInstance } from 'element-plus';
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

const rules = {
    code: [Rules.requiredInput('milvus.code')],
    name: [Rules.requiredInput('common.name')],
    host: [Rules.requiredInput('milvus.host')],
};

const milvusFormRef = useTemplateRef<FormInstance>('milvusFormRef');

const state = reactive({
    form: {
        id: null,
        code: '',
        name: null,
        host: '',
        database: 'default',
        sshTunnelMachineId: null as number | null,
        tagCodePaths: [],
        authCerts: [] as MachineAuthCert[],
    } as MilvusForm,
});

const { form } = toRefs(state);

const submitForm = computed(() => {
    const reqForm: Record<string, unknown> = { ...state.form };
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
});

const { isFetching: testConnBtnLoading, execute: testConnExec } = milvusApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveMilvusExec, data: saveMilvusRes } = milvusApi.save.useApi(submitForm);

watch(dialogVisible, () => {
    if (!dialogVisible.value) {
        return;
    }

    const milvusData = props.milvus as Milvus | false | undefined;
    if (milvusData) {
        state.form = { ...milvusData, authCerts: milvusData.authCerts || [] } as MilvusForm;
    } else {
        state.form = { database: 'default', sshTunnelMachineId: -1, authCerts: [] } as MilvusForm;
    }
});

const testConn = async (authCert: MachineAuthCert) => {
    await milvusFormRef.value?.validate();
    await testConnExec({
        ...submitForm.value,
        authCerts: [authCert],
    });
    Msg.success(('milvus.connSuccess'));
};

const onConfirm = async () => {
    await milvusFormRef.value?.validate();
    await saveMilvusExec(submitForm.value);
    Msg.success(('milvus.savedSuccess'));
    state.form.id = saveMilvusRes.value;
    emit('val-change', state.form);
    onCancel();
};

const onCancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>

<style lang="scss" scoped></style>
