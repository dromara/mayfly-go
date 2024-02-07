<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :close-on-click-modal="false" :destroy-on-close="true" :before-close="cancel" width="650px">
            <el-form :model="form" ref="machineForm" :rules="rules" label-width="auto">
                <el-tabs v-model="tabActiveName">
                    <el-tab-pane label="基础信息" name="basic">
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
                                :resource-code="form.code"
                                :resource-type="TagResourceTypeEnum.Machine.value"
                                style="width: 100%"
                            />
                        </el-form-item>
                        <el-form-item prop="name" label="名称" required>
                            <el-input v-model.trim="form.name" placeholder="请输入机器别名" auto-complete="off"></el-input>
                        </el-form-item>
                        <el-form-item prop="ip" label="ip" required>
                            <el-col :span="18">
                                <el-input :disabled="form.id" v-model.trim="form.ip" placeholder="主机ip" auto-complete="off"> </el-input>
                            </el-col>
                            <el-col style="text-align: center" :span="1">:</el-col>
                            <el-col :span="5">
                                <el-input type="number" v-model.number="form.port" placeholder="端口"></el-input>
                            </el-col>
                        </el-form-item>

                        <el-form-item prop="username" label="用户名">
                            <el-input v-model.trim="form.username" placeholder="请输授权用户名" autocomplete="new-password"> </el-input>
                        </el-form-item>

                        <el-form-item label="认证方式" required>
                            <el-select @change="changeAuthMethod" style="width: 100%" v-model="state.authType" placeholder="请选认证方式">
                                <el-option key="1" label="密码" :value="1"> </el-option>
                                <el-option key="2" label="授权凭证" :value="2"> </el-option>
                            </el-select>
                        </el-form-item>
                        <el-form-item v-if="state.authType == 1" prop="password" label="密码">
                            <el-input type="password" show-password v-model.trim="form.password" placeholder="请输入密码" autocomplete="new-password">
                            </el-input>
                        </el-form-item>

                        <el-form-item v-if="state.authType == 2" prop="authCertId" label="授权凭证" required>
                            <auth-cert-select ref="authCertSelectRef" v-model="form.authCertId" />
                        </el-form-item>

                        <el-form-item prop="remark" label="备注">
                            <el-input type="textarea" v-model="form.remark"></el-input>
                        </el-form-item>
                    </el-tab-pane>

                    <el-tab-pane label="其他配置" name="other">
                        <el-form-item prop="enableRecorder" label="终端回放">
                            <el-checkbox v-model="form.enableRecorder" :true-label="1" :false-label="-1"></el-checkbox>
                        </el-form-item>

                        <el-form-item prop="sshTunnelMachineId" label="SSH隧道">
                            <ssh-tunnel-select v-model="form.sshTunnelMachineId" />
                        </el-form-item>
                    </el-tab-pane>
                </el-tabs>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="testConn" :loading="testConnBtnLoading" type="success">测试连接</el-button>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="saveBtnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, watch, ref } from 'vue';
import { machineApi } from './api';
import { ElMessage } from 'element-plus';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import SshTunnelSelect from '../component/SshTunnelSelect.vue';
import AuthCertSelect from './authcert/AuthCertSelect.vue';
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
    name: [
        {
            required: true,
            message: '请输入别名',
            trigger: ['change', 'blur'],
        },
    ],
    ip: [
        {
            required: true,
            message: '请输入主机ip和端口',
            trigger: ['change', 'blur'],
        },
    ],
    authCertId: [
        {
            required: true,
            message: '请选择授权凭证',
            trigger: ['change', 'blur'],
        },
    ],
    username: [
        {
            required: true,
            message: '请输入授权用户名',
            trigger: ['change', 'blur'],
        },
    ],
};

const machineForm: any = ref(null);
const authCertSelectRef: any = ref(null);
const tagSelectRef: any = ref(null);

const state = reactive({
    dialogVisible: false,
    tabActiveName: 'basic',
    sshTunnelMachineList: [] as any,
    authCerts: [] as any,
    authType: 1,
    form: {
        id: null,
        code: '',
        tagPath: '',
        ip: null,
        port: 22,
        name: null,
        authCertId: null as any,
        username: '',
        password: '',
        tagId: [],
        remark: '',
        sshTunnelMachineId: null as any,
        enableRecorder: -1,
    },
    submitForm: {},
    pwd: '',
});

const { dialogVisible, tabActiveName, form, submitForm } = toRefs(state);

const { isFetching: testConnBtnLoading, execute: testConnExec } = machineApi.testConn.useApi(submitForm);
const { isFetching: saveBtnLoading, execute: saveMachineExec } = machineApi.saveMachine.useApi(submitForm);

watch(props, async (newValue: any) => {
    state.dialogVisible = newValue.visible;
    if (!state.dialogVisible) {
        return;
    }
    state.tabActiveName = 'basic';
    if (newValue.machine) {
        state.form = { ...newValue.machine };

        // 如果凭证类型为公共的，则表示使用授权凭证认证
        const authCertId = (state.form as any).authCertId;
        if (authCertId > 0) {
            state.authType = 2;
        } else {
            state.authType = 1;
        }
    } else {
        state.form = { port: 22, tagId: [] } as any;
        state.authType = 1;
    }
});

const changeAuthMethod = (val: any) => {
    if (state.form.id) {
        if (val == 2) {
            state.form.authCertId = null;
        } else {
            state.form.password = '';
        }
    }
};

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

        state.submitForm = getReqForm();
        await saveMachineExec();
        ElMessage.success('保存成功');
        emit('val-change', submitForm);
        cancel();
    });
};

const getReqForm = () => {
    const reqForm: any = { ...state.form };
    // 如果为密码认证，则置空授权凭证id
    if (state.authType == 1) {
        reqForm.authCertId = -1;
    }
    if (!state.form.sshTunnelMachineId || state.form.sshTunnelMachineId <= 0) {
        reqForm.sshTunnelMachineId = -1;
    }
    return reqForm;
};

const cancel = () => {
    emit('update:visible', false);
    emit('cancel');
};
</script>
<style lang="scss"></style>
