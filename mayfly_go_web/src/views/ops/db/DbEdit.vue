<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" :close-on-click-modal="false" :destroy-on-close="true" width="35%">
            <el-form :model="form" ref="dbForm" :rules="rules" label-width="85px">
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
                <el-form-item prop="password" label="密码:">
                    <el-input
                        type="password"
                        show-password
                        v-model.trim="form.password"
                        placeholder="请输入密码，修改操作可不填"
                        autocomplete="new-password"
                    ></el-input>
                </el-form-item>
                <el-form-item prop="database" label="数据库名:" required>
                    <el-tag
                        v-for="db in databaseList"
                        :key="db"
                        class="ml5 mt5"
                        type="success"
                        effect="plain"
                        closable
                        :disable-transitions="false"
                        @close="handleClose(db)"
                    >
                        {{ db }}
                    </el-tag>
                    <el-input
                        v-if="inputDbVisible"
                        ref="InputDbRef"
                        v-model="inputDbValue"
                        style="width: 120px; margin-left: 5px; margin-top: 5px"
                        size="small"
                        @keyup.enter="handleInputDbConfirm"
                        @blur="handleInputDbConfirm"
                    />
                    <el-button v-else class="ml5 mt5" size="small" @click="showInputDb"> + 添加数据库 </el-button>
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
import { toRefs, reactive, nextTick, watch, defineComponent, ref } from 'vue';
import { dbApi } from './api';
import { projectApi } from '../project/api.ts';
import { ElMessage } from 'element-plus';
import type { ElInput } from 'element-plus';
import { notBlank } from '@/common/assert';

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
        const InputDbRef = ref<InstanceType<typeof ElInput>>();

        const state = reactive({
            dialogVisible: false,
            projects: [],
            envs: [],
            databaseList: [] as any,
            inputDbVisible: false,
            inputDbValue: '',
            form: {
                id: null,
                name: null,
                port: 3306,
                username: null,
                password: null,
                database: '',
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
                // password: [
                //     {
                //         required: true,
                //         message: '请输入密码',
                //         trigger: ['change', 'blur'],
                //     },
                // ],
                database: [
                    {
                        required: true,
                        message: '请添加数据库',
                        trigger: ['change', 'blur'],
                    },
                ],
            },
        });

        watch(props, (newValue) => {
            state.projects = newValue.projects;
            if (newValue.db) {
                getEnvs(newValue.db.projectId);
                state.form = { ...newValue.db };
                // 将数据库名使用空格切割，获取所有数据库列表
                state.databaseList = newValue.db.database.split(' ');
            } else {
                state.envs = [];
                state.form = { port: 3306 } as any;
                state.databaseList = [];
            }
            state.dialogVisible = newValue.visible;
        });

        const handleClose = (db: string) => {
            state.databaseList.splice(state.databaseList.indexOf(db), 1);
            changeDatabase();
        };

        const showInputDb = () => {
            state.inputDbVisible = true;
            nextTick(() => {
                InputDbRef.value!.input!.focus();
            });
        };

        const handleInputDbConfirm = () => {
            if (state.inputDbValue) {
                state.databaseList.push(state.inputDbValue);
                changeDatabase();
            }
            state.inputDbVisible = false;
            state.inputDbValue = '';
        };

        /**
         * 改变表单中的数据库字段，方便表单错误提示。如全部删光，可提示请添加数据库
         */
        const changeDatabase = () => {
            state.form.database = state.databaseList.length == 0 ? '' : state.databaseList.join(' ');
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
            if (!state.form.id) {
                notBlank(state.form.password, '新增操作，密码不可为空');
            }
            dbForm.value.validate((valid: boolean) => {
                if (valid) {
                    state.form.port = Number.parseInt(state.form.port as any);
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

        const resetInputDb = () => {
            state.inputDbVisible = false;
            state.databaseList = [];
            state.inputDbValue = '';
        };

        const cancel = () => {
            emit('update:visible', false);
            emit('cancel');
            setTimeout(() => {
                resetInputDb();
                dbForm.value.resetFields();
                //  重置对象属性为null
                state.form = {} as any;
            }, 200);
        };

        return {
            ...toRefs(state),
            dbForm,
            InputDbRef,
            handleClose,
            showInputDb,
            handleInputDbConfirm,
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
