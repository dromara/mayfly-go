<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :close-on-click-modal="false" :destroy-on-close="true" :before-close="cancel" width="38%">
            <el-form :model="form" ref="machineForm" :rules="rules" label-width="85px">
                <el-form-item prop="projectId" label="项目:" required>
                    <el-select style="width: 100%" v-model="form.projectId" placeholder="请选择项目" @change="changeProject" filterable>
                        <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="name" label="名称:" required>
                    <el-input v-model.trim="form.name" placeholder="请输入机器别名" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="ip" label="ip:" required>
                    <el-col :span="18">
                        <el-input :disabled="form.id" v-model.trim="form.ip" placeholder="主机ip" auto-complete="off"></el-input>
                    </el-col>
                    <el-col style="text-align: center" :span="1">:</el-col>
                    <el-col :span="5">
                        <el-input type="number" v-model.number="form.port" placeholder="端口"></el-input>
                    </el-col>
                </el-form-item>
                <el-form-item prop="username" label="用户名:" required>
                    <el-input v-model.trim="form.username" placeholder="请输入用户名"></el-input>
                </el-form-item>
                <el-form-item prop="authMethod" label="认证方式:" required>
                    <el-select style="width: 100%" v-model="form.authMethod" placeholder="请选择认证方式">
                        <el-option key="1" label="Password" :value="1"> </el-option>
                        <el-option key="2" label="PublicKey" :value="2"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item v-if="form.authMethod == 1" prop="password" label="密码:">
                    <el-input
                        type="password"
                        show-password
                        v-model.trim="form.password"
                        placeholder="请输入密码，修改操作可不填"
                        autocomplete="new-password"
                    >
                        <template v-if="form.id && form.id != 0" #suffix>
                            <el-popover @hide="pwd = ''" placement="right" title="原密码" :width="200" trigger="click" :content="pwd">
                                <template #reference>
                                    <el-link @click="getPwd" :underline="false" type="primary" class="mr5">原密码</el-link>
                                </template>
                            </el-popover>
                        </template>
                    </el-input>
                </el-form-item>
                <el-form-item v-if="form.authMethod == 2" prop="password" label="秘钥:">
                    <el-input type="textarea" :rows="3" v-model="form.password" placeholder="请将私钥文件内容拷贝至此，修改操作可不填"></el-input>
                </el-form-item>
                <el-form-item prop="remark" label="备注:">
                    <el-input type="textarea" v-model="form.remark"></el-input>
                </el-form-item>

                <el-form-item prop="enableRecorder" label="终端回放:">
                    <el-checkbox v-model="form.enableRecorder" :true-label="1" :false-label="-1"></el-checkbox>
                </el-form-item>

                <el-form-item prop="enableSshTunnel" label="SSH隧道:">
                    <el-col :span="3">
                        <el-checkbox @change="getSshTunnelMachines" v-model="form.enableSshTunnel" :true-label="1" :false-label="-1"></el-checkbox>
                    </el-col>
                    <el-col :span="2" v-if="form.enableSshTunnel == 1"> 机器: </el-col>
                    <el-col :span="19" v-if="form.enableSshTunnel == 1">
                        <el-select style="width: 100%" v-model="form.sshTunnelMachineId" placeholder="请选择SSH隧道机器">
                            <el-option
                                v-for="item in sshTunnelMachineList"
                                :key="item.id"
                                :label="`${item.ip}:${item.port} [${item.name}]`"
                                :value="item.id"
                            >
                            </el-option>
                        </el-select>
                    </el-col>
                </el-form-item>
            </el-form>

            <template #footer>
                <div>
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, watch, defineComponent, ref } from 'vue';
import { machineApi } from './api';
import { ElMessage } from 'element-plus';
import { notBlank } from '@/common/assert';
import { RsaEncrypt } from '@/common/rsa';

export default defineComponent({
    name: 'MachineEdit',
    props: {
        visible: {
            type: Boolean,
        },
        projects: {
            type: Array,
        },
        machine: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const machineForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            projects: [],
            sshTunnelMachineList: [],
            form: {
                id: null,
                projectId: null,
                projectName: null,
                name: null,
                authMethod: 1,
                port: 22,
                username: '',
                password: '',
                remark: '',
                enableSshTunnel: null,
                sshTunnelMachineId: null,
                enableRecorder: -1,
            },
            pwd: '',
            btnLoading: false,
            rules: {
                projectId: [
                    {
                        required: true,
                        message: '请选择项目',
                        trigger: ['change', 'blur'],
                    },
                ],
                envId: [
                    {
                        required: true,
                        message: '请选择环境',
                        trigger: ['change', 'blur'],
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
                username: [
                    {
                        required: true,
                        message: '请输入用户名',
                        trigger: ['change', 'blur'],
                    },
                ],
                authMethod: [
                    {
                        required: true,
                        message: '请选择认证方式',
                        trigger: ['change', 'blur'],
                    },
                ],
            },
        });

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            if (!state.dialogVisible) {
                return;
            }
            state.projects = newValue.projects;
            if (newValue.machine) {
                state.form = { ...newValue.machine };
            } else {
                state.form = { port: 22, authMethod: 1 } as any;
            }
            getSshTunnelMachines();
        });

        const getSshTunnelMachines = async () => {
            if (state.form.enableSshTunnel == 1 && state.sshTunnelMachineList.length == 0) {
                const res = await machineApi.list.request({ pageNum: 1, pageSize: 100 });
                state.sshTunnelMachineList = res.list;
            }
        };

        const getSshTunnelMachine = (machineId: any) => {
            notBlank(machineId, '请选择或先创建一台隧道机器');
            return state.sshTunnelMachineList.find((x: any) => x.id == machineId);
        };

        const getPwd = async () => {
            state.pwd = await machineApi.getMachinePwd.request({ id: state.form.id });
        };

        const changeProject = (projectId: number) => {
            for (let p of state.projects as any) {
                if (p.id == projectId) {
                    state.form.projectName = p.name;
                }
            }
        };

        const btnOk = async () => {
            if (!state.form.id) {
                notBlank(state.form.password, '新增操作，密码不可为空');
            }
            machineForm.value.validate(async (valid: boolean) => {
                if (valid) {
                    const form: any = state.form;
                    if (form.enableSshTunnel == 1) {
                        const tunnelMachine: any = getSshTunnelMachine(form.sshTunnelMachineId);
                        if (tunnelMachine.ip == form.ip && tunnelMachine.port == form.port) {
                            ElMessage.error('隧道机器不能与本机器一致');
                            return;
                        }
                    }
                    const reqForm: any = { ...form };
                    if (reqForm.authMethod == 1) {
                        reqForm.password = await RsaEncrypt(state.form.password);
                    }
                    state.btnLoading = true;
                    try {
                        await machineApi.saveMachine.request(reqForm);
                        ElMessage.success('保存成功');
                        emit('val-change', state.form);
                        cancel();
                    } finally {
                        state.btnLoading = false;
                    }
                } else {
                    ElMessage.error('请正确填写信息');
                    return false;
                }
            });
        };

        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
        };

        return {
            ...toRefs(state),
            machineForm,
            getSshTunnelMachines,
            getPwd,
            changeProject,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
