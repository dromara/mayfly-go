<template>
    <div>
        <el-dialog :title="title" v-model="visible" :show-close="false" :before-close="cancel" width="35%">
            <el-form :model="form" ref="redisForm" :rules="rules" label-width="85px" size="small">
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
                <el-form-item prop="host" label="host:" required>
                    <el-input v-model.trim="form.host" placeholder="请输入host:port" auto-complete="off"></el-input>
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
                    <el-input v-model.trim="form.db" placeholder="请输入库号"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button type="primary" :loading="btnLoading" @click="btnOk" size="mini">确 定</el-button>
                    <el-button @click="cancel()" size="mini">取 消</el-button>
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
            visible: false,
            projects: [],
            envs: [],
            form: {
                id: null,
                name: null,
                host: null,
                password: null,
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
            },
        });

        watch(props, async (newValue, oldValue) => {
            state.visible = newValue.visible;
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
                    redisApi.saveRedis.request(state.form).then((res: any) => {
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
            setTimeout(() => {
                redisForm.value.resetFields();
                //  重置对象属性为null
                state.form = {} as any;
            }, 200);
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
