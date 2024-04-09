<template>
    <div>
        <el-drawer :title="title" v-model="dialogVisible" :before-close="cancel" :destroy-on-close="true" :close-on-click-modal="false" size="40%">
            <template #header>
                <DrawerHeader :header="title" :back="cancel" />
            </template>

            <el-form :model="form" ref="machineForm" :rules="rules" label-width="auto">
                <el-divider content-position="left">基本</el-divider>
                <el-form-item ref="tagSelectRef" prop="tagId" label="标签">
                    <tag-tree-select
                        multiple
                        @change-tag="
                            (tagIds) => {
                                form.tagId = tagIds;
                                tagSelectRef.validate();
                            }
                        "
                        :tag-path="form.tagPath"
                        :select-tags="form.tagId"
                        style="width: 100%"
                    />
                </el-form-item>
                <el-form-item prop="code" label="编号" required>
                    <el-input :disabled="form.id" v-model.trim="form.code" placeholder="请输入编号 (数字字母下划线), 不可修改" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="name" label="名称" required>
                    <el-input v-model.trim="form.name" placeholder="请输入机器别名" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="protocol" label="协议" required>
                    <el-radio-group v-model="form.protocol" @change="handleChangeProtocol">
                        <el-radio v-for="item in MachineProtocolEnum" :key="item.value" :label="item.label" :value="item.value"></el-radio>
                    </el-radio-group>
                </el-form-item>
                <el-form-item prop="ip" label="ip" required>
                    <el-col :span="18">
                        <el-input v-model.trim="form.ip" placeholder="主机ip" auto-complete="off"> </el-input>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="5">
                        <el-input type="number" v-model.number="form.port" placeholder="端口"></el-input>
                    </el-col>
                </el-form-item>

                <el-form-item prop="remark" label="备注">
                    <el-input type="textarea" v-model="form.remark"></el-input>
                </el-form-item>

                <el-divider content-position="left">账号</el-divider>
                <div>
                    <ResourceAuthCertEdit v-model="form.authCerts" :resource-code="form.code" :resource-type="TagResourceTypeEnum.Machine.value" />
                </div>

                <!-- <el-tab-pane label="其他配置" name="other"> -->
                <el-divider content-position="left">其他</el-divider>
                <el-form-item prop="enableRecorder" label="终端回放">
                    <el-checkbox v-model="form.enableRecorder" :true-value="1" :false-value="-1"></el-checkbox>
                </el-form-item>

                <el-form-item prop="sshTunnelMachineId" label="SSH隧道">
                    <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                </el-form-item>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="testConn" :loading="testConnBtnLoading" type="success">测试连接</el-button>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, watch } from 'vue';
import { machineApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import ResourceAuthCertEdit from '../component/ResourceAuthCertEdit.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import { MachineProtocolEnum } from './enums';
import DrawerHeader from '@/components/drawer-header/DrawerHeader.vue';
import { ResourceCodePattern } from '@/common/pattern';
import { TagResourceTypeEnum } from '@/common/commonEnum';

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
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const rules = {
    tagId: [
        {
            required: true,
            message: '请选择标签',
            trigger: ['change'],
        },
    ],
    code: [
        {
            required: true,
            message: '请输入编码',
            trigger: ['change', 'blur'],
        },
        {
            pattern: ResourceCodePattern.pattern,
            message: ResourceCodePattern.message,
            trigger: ['blur'],
        },
    ],
    name: [
        {
            required: true,
            message: '请输入别名',
            trigger: ['change', 'blur'],
        },
    ],
    protocol: [
        {
            required: true,
            message: '请选择机器类型',
            trigger: ['change', 'blur'],
        },
    ],
    ip: [
        {
            required: true,
            message: '请输入主机ip和端口',
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
    tagId: [],
    remark: '',
    sshTunnelMachineId: null as any,
    enableRecorder: -1,
};

const state = reactive({
    dialogVisible: false,
    sshTunnelMachineList: [] as any,
    form: defaultForm,
    submitForm: {},
    pwd: '',
});

const { dialogVisible, form, submitForm } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = machineApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveMachineExec } = machineApi.saveMachine.useApi(submitForm);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        state.form = defaultForm;
        return;
    }
    if (newValue.machine) {
        state.form = { ...newValue.machine };
        state.form.tagId = newValue.machine.tags.map((t: any) => t.tagId);
        state.form.authCerts = newValue.machine.authCerts || [];
    }
});

const testConn = async () => {
    machineForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        state.submitForm = getReqForm();
        await testConnExec();
        ElMessage.success('连接成功');
    });
};

const btnOk = async () => {
    machineForm.value.validate(async (valid: boolean) => {
        if (!valid) {
            ElMessage.error('请正确填写信息');
            return false;
        }

        if (state.form.authCerts.length == 0) {
            ElMessage.error('请完善授权凭证账号信息');
            return false;
        }

        state.submitForm = getReqForm();
        await saveMachineExec();
        ElMessage.success('保存成功');
        emit('val-change', submitForm);
        cancel();
    });
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
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
