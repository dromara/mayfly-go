<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="35%">
            <el-form :model="form" ref="redisForm" :rules="rules" label-width="85px">
                <el-form-item prop="projectId" label="项目:" required>
                    <el-select style="width: 100%" v-model="form.projectId" placeholder="请选择项目" @change="changeProject" filterable>
                        <el-option v-for="item in projects" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="envId" label="环境:" required>
                    <el-select @change="changeEnv" style="width: 100%" v-model="form.envId" placeholder="请选择环境">
                        <el-option v-for="item in envs" :key="item.id" :label="`${item.name} [${item.remark}]`" :value="item.id"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="mode" label="mode:" required>
                    <el-select style="width: 100%" v-model="form.mode" placeholder="请选择模式">
                        <el-option label="standalone" value="standalone"> </el-option>
                        <el-option label="cluster" value="cluster"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="host" label="host:" required>
                    <el-input v-model.trim="form.host" placeholder="请输入host:port，集群模式用','分割" auto-complete="off" type="textarea"></el-input>
                </el-form-item>
                <el-form-item prop="password" label="密码:">
                    <el-input
                        type="password"
                        show-password
                        v-model.trim="form.password"
                        placeholder="请输入密码"
                        autocomplete="new-password"
                    ></el-input>
                </el-form-item>
                <el-form-item prop="db" label="库号:" required>
                    <el-input v-model.number="form.db" placeholder="请输入库号"></el-input>
                </el-form-item>
                <el-form-item prop="remark" label="备注:">
                    <el-input v-model.trim="form.remark" auto-complete="off" type="textarea"></el-input>
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
import { redisApi } from './api';
import { projectApi } from '../project/api.ts';
import { ElMessage } from 'element-plus';

export default defineComponent({
    name: 'RedisEdit',
    props: {
        visible: {
            type: Boolean,
        },
        projects: {
            type: Array,
        },
        redis: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const redisForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            projects: [],
            envs: [],
            form: {
                id: null,
                name: null,
                mode: "standalone",
                host: null,
                password: null,
                project: null,
                projectId: null,
                envId: null,
                env: null,
                remark: "",
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
                host: [
                    {
                        required: true,
                        message: '请输入主机ip:port',
                        trigger: ['change', 'blur'],
                    },
                ],
                db: [
                    {
                        required: true,
                        message: '请输入库号',
                        trigger: ['change', 'blur'],
                    },
                ],
                mode: [
                    {
                        required: true,
                        message: '请输入模式',
                        trigger: ['change', 'blur'],
                    },
                ],
            },
        });

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            state.projects = newValue.projects;
            if (newValue.redis) {
                getEnvs(newValue.redis.projectId);
                state.form = { ...newValue.redis };
            } else {
                state.envs = [];
                state.form = { db: 0 } as any;
            }
        });

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
            redisForm.value.validate((valid: boolean) => {
                if (valid) {
                    redisApi.saveRedis.request(state.form).then(() => {
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
            redisForm,
            changeProject,
            changeEnv,
            btnOk,
            cancel,
        };
    },
});
</script>
<style lang="scss">
</style>
