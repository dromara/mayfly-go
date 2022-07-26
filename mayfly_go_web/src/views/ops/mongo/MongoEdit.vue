<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" width="38%" :destroy-on-close="true">
            <el-form :model="form" ref="mongoForm" :rules="rules" label-width="85px">
                <el-form-item prop="projectId" label="项目" required>
                    <el-select style="width: 100%" v-model="form.projectId" placeholder="请选择项目" @change="changeProject" filterable>
                        <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="envId" label="环境" required>
                    <el-select @change="changeEnv" style="width: 100%" v-model="form.envId" placeholder="请选择环境">
                        <el-option v-for="item in envs" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="name" label="名称" required>
                    <el-input v-model.trim="form.name" placeholder="请输入名称" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="uri" label="uri" required>
                    <el-input
                        type="textarea"
                        :rows="2"
                        v-model.trim="form.uri"
                        placeholder="形如 mongodb://username:password@host1:port1"
                        auto-complete="off"
                    ></el-input>
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
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, watch, defineComponent, ref } from 'vue';
import { mongoApi } from './api';
import { projectApi } from '../project/api.ts';
import { machineApi } from '../machine/api.ts';
import { ElMessage } from 'element-plus';

export default defineComponent({
    name: 'MongoEdit',
    props: {
        visible: {
            type: Boolean,
        },
        projects: {
            type: Array,
        },
        mongo: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const mongoForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            projects: [],
            envs: [],
            sshTunnelMachineList: [],
            form: {
                id: null,
                name: null,
                uri: null,
                enableSshTunnel: -1,
                sshTunnelMachineId: null,
                project: null,
                projectId: null,
                envId: null,
                env: null,
            },
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
                        message: '请输入名称',
                        trigger: ['change', 'blur'],
                    },
                ],
                uri: [
                    {
                        required: true,
                        message: '请输入mongo uri',
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
            if (newValue.mongo) {
                getEnvs(newValue.mongo.projectId);
                state.form = { ...newValue.mongo };
            } else {
                state.envs = [];
                state.form = { db: 0 } as any;
            }
            getSshTunnelMachines();
        });

        const getSshTunnelMachines = async () => {
            if (state.form.enableSshTunnel == 1 && state.sshTunnelMachineList.length == 0) {
                const res = await machineApi.list.request({ pageNum: 1, pageSize: 100 });
                state.sshTunnelMachineList = res.list;
            }
        };

        const getEnvs = async (projectId: any) => {
            state.envs = await projectApi.projectEnvs.request({ projectId });
        };

        const changeProject = (projectId: number) => {
            for (let p of state.projects as any) {
                if (p.id == projectId) {
                    state.form.project = p.name;
                }
            }
            state.form.envId = null;
            state.form.env = null;
            state.envs = [];
            getEnvs(projectId);
        };

        const changeEnv = (envId: number) => {
            for (let p of state.envs as any) {
                if (p.id == envId) {
                    state.form.env = p.name;
                }
            }
        };

        const btnOk = async () => {
            mongoForm.value.validate(async (valid: boolean) => {
                if (valid) {
                    const reqForm = { ...state.form };
                    // reqForm.uri = await RsaEncrypt(reqForm.uri);
                    mongoApi.saveMongo.request(reqForm).then(() => {
                        ElMessage.success('保存成功');
                        emit('val-change', state.form);
                        state.btnLoading = true;
                        setTimeout(() => {
                            state.btnLoading = false;
                        }, 1000);

                        cancel();
                    });
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
            mongoForm,
            changeProject,
            getSshTunnelMachines,
            changeEnv,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
