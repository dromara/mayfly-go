<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :show-close="false" :before-close="cancel" width="35%">
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="85px" size="small">
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
                <el-form-item prop="name" label="别名:" required>
                    <el-input v-model.trim="form.name" placeholder="请输入数据库别名" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="type" label="类型:" required>
                    <el-select style="width: 100%" v-model="form.type" placeholder="请选择数据库类型">
                        <el-option key="item.id" label="mysql" value="mysql"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item prop="host" label="host:" required>
                    <el-input v-model.trim="form.host" placeholder="请输入主机ip" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="port" label="port:" required>
                    <el-input type="number" v-model.trim="form.port" placeholder="请输入端口"></el-input>
                </el-form-item>
                <el-form-item prop="username" label="用户名:" required>
                    <el-input v-model.trim="form.username" placeholder="请输入用户名"></el-input>
                </el-form-item>
                <el-form-item prop="password" label="密码:" required>
                    <el-input
                        type="password"
                        show-password
                        v-model.trim="form.password"
                        placeholder="请输入密码"
                        autocomplete="new-password"
                    ></el-input>
                </el-form-item>
                <el-form-item prop="database" label="数据库名:" required>
                    <el-input v-model.trim="form.database" placeholder="请输入数据库名"></el-input>
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
import { dbApi } from './api';
import { projectApi } from '../project/api.ts';
import { ElMessage } from 'element-plus';

export default defineComponent({
    name: 'DbEdit',
    props: {
        visible: {
            type: Boolean,
        },
        projects: {
            type: Array,
        },
        db: {
            type: [Boolean, Object],
        },
        title: {
            type: String,
        },
    },
    setup(props: any, { emit }) {
        const dbForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            projects: [],
            envs: [],
            form: {
                id: null,
                name: null,
                port: 3306,
                username: null,
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
                name: [
                    {
                        required: true,
                        message: '请输入别名',
                        trigger: ['change', 'blur'],
                    },
                ],
                type: [
                    {
                        required: true,
                        message: '请选择数据库类型',
                        trigger: ['change', 'blur'],
                    },
                ],
                host: [
                    {
                        required: true,
                        message: '请输入主机ip',
                        trigger: ['change', 'blur'],
                    },
                ],
                port: [
                    {
                        required: true,
                        message: '请输入端口',
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
                password: [
                    {
                        required: true,
                        message: '请输入密码',
                        trigger: ['change', 'blur'],
                    },
                ],
                database: [
                    {
                        required: true,
                        message: '请输入数据库名',
                        trigger: ['change', 'blur'],
                    },
                ],
            },
        });

        watch(props, async (newValue) => {
            state.dialogVisible = newValue.visible;
            state.projects = newValue.projects;
            if (newValue.db) {
                getEnvs(newValue.db.projectId);
                state.form = { ...newValue.db };
            } else {
                state.envs = [];
                state.form = { port: 3306 } as any;
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
            dbForm.value.validate((valid: boolean) => {
                if (valid) {
                    dbApi.saveDb.request(state.form).then(() => {
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
                dbForm.value.resetFields();
                //  重置对象属性为null
                state.form = {} as any;
            }, 200);
        };

        return {
            ...toRefs(state),
            dbForm,
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
