<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="machineForm" :rules="rules" label-width="auto">
                <el-divider content-position="left">{{ $t('common.basic') }}</el-divider>
                <el-form-item ref="tagSelectRef" prop="tagCodePaths" :label="$t('tag.relateTag')">
                    <tag-tree-select
                        multiple
                        @change-tag="
                            (paths) => {
                                form.tagCodePaths = paths;
                                tagSelectRef.validate();
                            }
                        "
                        :select-tags="form.tagCodePaths"
                        style="width: 100%"
                    />
                </el-form-item>
                <el-form-item prop="name" :label="$t('common.name')" required>
                    <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="protocol" :label="$t('machine.protocol')" required>
                    <el-radio-group v-model="form.protocol" @change="handleChangeProtocol">
                        <el-radio v-for="item in MachineProtocolEnum" :key="item.value" :label="item.label" :value="item.value"></el-radio>
                    </el-radio-group>
                </el-form-item>
                <el-form-item prop="ip" label="ip" required>
                    <el-col :span="18">
                        <el-input v-model.trim="form.ip" auto-complete="off"> </el-input>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="5">
                        <el-input type="number" v-model.number="form.port" :placeholder="$t('machine.port')"></el-input>
                    </el-col>
                </el-form-item>

                <el-form-item prop="remark" :label="$t('common.remark')">
                    <el-input type="textarea" v-model="form.remark"></el-input>
                </el-form-item>

                <el-divider content-position="left">{{ $t('common.account') }}</el-divider>
                <div>
                    <ResourceAuthCertTableEdit
                        v-model="form.authCerts"
                        :resource-code="form.code"
                        :resource-type="TagResourceTypeEnum.Machine.value"
                        :test-conn-btn-loading="testConnBtnLoading"
                        @test-conn="testConn"
                    />
                </div>

                <el-divider content-position="left">{{ $t('common.other') }}</el-divider>
                <el-form-item prop="enableRecorder" :label="$t('machine.sshTunnel')">
                    <el-checkbox v-model="form.enableRecorder" :true-value="1" :false-value="-1"></el-checkbox>
                </el-form-item>

                <el-form-item prop="sshTunnelMachineId" :label="$t('machine.sshTunnel')">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, watchEffect } from 'vue';
import { machineApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import ResourceAuthCertTableEdit from '../component/ResourceAuthCertTableEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import { MachineProtocolEnum } from './enums';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { useI18nFormValidate, useI18nPleaseInput, useI18nPleaseSelect, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    visible: {
        type: Boolean,
    },
    machine: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const rules = {
    tagCodePaths: [
        {
            required: true,
            message: useI18nPleaseSelect('tag.relateTag'),
            trigger: ['change'],
        },
    ],
    name: [
        {
            required: true,
            message: useI18nPleaseInput('common.name'),
            trigger: ['change', 'blur'],
        },
    ],
    protocol: [
        {
            required: true,
            message: useI18nPleaseSelect('machine.protocol'),
            trigger: ['change', 'blur'],
        },
    ],
    ip: [
        {
            required: true,
            message: useI18nPleaseInput('machine.ipAndPort'),
            trigger: ['blur'],
        },
    ],
};

const machineForm: any = ref(null);
const tagSelectRef: any = ref(null);

const defaultForm = {
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
    sshTunnelMachineId: null as any,
    enableRecorder: -1,
};

const state = reactive({
    sshTunnelMachineList: [] as any,
    form: defaultForm,
    submitForm: {} as any,
    pwd: '',
});

const { form, submitForm } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = machineApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveMachineExec } = machineApi.saveMachine.useApi(submitForm);

watchEffect(() => {
    if (!dialogVisible.value) {
        return;
    }
    const machine: any = props.machine;
    if (machine) {
        state.form = { ...machine };
        state.form.tagCodePaths = machine.tags.map((t: any) => t.codePath);
        state.form.authCerts = machine.authCerts || [];
    } else {
        state.form = { ...defaultForm };
        state.form.authCerts = [];
    }
});

const testConn = async (authCert: any) => {
    await useI18nFormValidate(machineForm);

    state.submitForm = getReqForm();
    state.submitForm.authCerts = [authCert];
    await testConnExec();
    ElMessage.success(t('machine.connSuccess'));
};

const btnOk = async () => {
    await useI18nFormValidate(machineForm);

    if (state.form.authCerts.length == 0) {
        ElMessage.error(t('machine.noAcErrMsg'));
        return false;
    }

    state.submitForm = getReqForm();
    await saveMachineExec();
    useI18nSaveSuccessMsg();
    emit('val-change', submitForm);
    cancel();
};

const getReqForm = () => {
    const reqForm: any = { ...state.form };
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const handleChangeProtocol = (val: any) => {
    if (val == MachineProtocolEnum.Ssh.value) {
        state.form.port = 22;
    } else if (val == MachineProtocolEnum.Rdp.value) {
        state.form.port = 3389;
    } else {
        state.form.port = 5901;
    }
};

const cancel = () => {
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
