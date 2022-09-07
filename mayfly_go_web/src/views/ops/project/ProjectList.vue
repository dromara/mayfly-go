<template>
    <div class="project-list">
        <el-card>
            <div>
                <el-button @click="showAddProjectDialog" v-auth="permissions.saveProject" type="primary" icon="plus">添加</el-button>
                <el-button
                    @click="showAddProjectDialog(chooseData)"
                    v-auth="permissions.saveProject"
                    :disabled="chooseId == null"
                    type="primary"
                    icon="edit"
                    >编辑</el-button
                >
                <el-button @click="showMembers(chooseData)" :disabled="chooseId == null" type="success" icon="user">成员管理</el-button>

                <el-button @click="showEnv(chooseData)" :disabled="chooseId == null" type="info" icon="setting">环境管理</el-button>

                <el-button v-auth="permissions.delProject" @click="delProject" :disabled="chooseId == null" type="danger" icon="delete"
                    >删除</el-button
                >

                <div style="float: right">
                    <el-input class="mr2" placeholder="请输入项目名！" style="width: 200px" v-model="query.name" @clear="search" clearable></el-input>
                    <el-button @click="search" type="success" icon="search"></el-button>
                </div>
            </div>
            <el-table :data="projects" @current-change="choose" ref="table" style="width: 100%">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="项目名"></el-table-column>
                <el-table-column prop="remark" label="描述" min-width="180px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="createTime" label="创建时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="creator" label="创建者"> </el-table-column>
                <!-- <el-table-column label="查看更多" min-width="80px">
                <template #default="scope">
                    <el-link @click.prevent="showMembers(scope.row)" type="success">成员</el-link>

                    <el-link class="ml5" @click.prevent="showEnv(scope.row)" type="info">环境</el-link>
                </template>
            </el-table-column> -->
            </el-table>
            <el-row style="margin-top: 20px" type="flex" justify="end">
                <el-pagination
                    style="text-align: right"
                    @current-change="handlePageChange"
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                ></el-pagination>
            </el-row>
        </el-card>

        <el-dialog width="400px" title="项目编辑" :before-close="cancelAddProject" v-model="addProjectDialog.visible">
            <el-form :model="addProjectDialog.form" label-width="70px">
                <el-form-item prop="name" label="项目名:" required>
                    <el-input :disabled="addProjectDialog.form.id ? true : false" v-model="addProjectDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="描述:">
                    <el-input v-model="addProjectDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelAddProject()">取 消</el-button>
                    <el-button @click="addProject" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog width="500px" :title="showEnvDialog.title" v-model="showEnvDialog.visible">
            <div class="toolbar">
                <el-button @click="showAddEnvDialog" v-auth="permissions.saveMember" type="primary" icon="plus">添加</el-button>
                <el-button @click="deleteEnv" v-auth="permissions.delProject" :disabled="showEnvDialog.chooseId == null" type="danger" icon="delete"
                    >删除</el-button
                >
            </div>
            <el-table @current-change="chooseEnv" border :data="showEnvDialog.envs">
                <el-table-column label="选择" width="50px">
                    <template #default="scope">
                        <el-radio v-model="showEnvDialog.chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column property="name" label="环境名" width="125"></el-table-column>
                <el-table-column property="remark" label="描述" width="125"></el-table-column>
                <el-table-column property="createTime" label="创建时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
            </el-table>

            <el-dialog width="400px" title="添加环境" :before-close="cancelAddEnv" v-model="showEnvDialog.addVisible">
                <el-form :model="showEnvDialog.envForm" label-width="70px">
                    <el-form-item prop="name" label="环境名:" required>
                        <el-input v-model="showEnvDialog.envForm.name" auto-complete="off"></el-input>
                    </el-form-item>
                    <el-form-item label="描述:">
                        <el-input v-model="showEnvDialog.envForm.remark" auto-complete="off"></el-input>
                    </el-form-item>
                </el-form>
                <template #footer>
                    <div class="dialog-footer">
                        <el-button @click="cancelAddEnv()">取 消</el-button>
                        <el-button v-auth="permissions.saveEnv" @click="addEnv" type="primary" :loading="btnLoading">确 定</el-button>
                    </div>
                </template>
            </el-dialog>
        </el-dialog>

        <el-dialog width="500px" :title="showMemDialog.title" v-model="showMemDialog.visible">
            <div class="toolbar">
                <el-button v-auth="permissions.saveMember" @click="showAddMemberDialog()" type="primary" icon="plus">添加</el-button>
                <el-button v-auth="permissions.delMember" @click="deleteMember" :disabled="showMemDialog.chooseId == null" type="danger" icon="delete"
                    >移除</el-button
                >
            </div>
            <el-table @current-change="chooseMember" border :data="showMemDialog.members.list">
                <el-table-column label="选择" width="50px">
                    <template #default="scope">
                        <el-radio v-model="showMemDialog.chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column property="username" label="账号" width="125"></el-table-column>
                <el-table-column property="createTime" label="加入时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column property="creator" label="分配者" width="125"></el-table-column>
            </el-table>
            <el-pagination
                @current-change="setMemebers"
                style="text-align: center"
                background
                layout="prev, pager, next, total, jumper"
                :total="showMemDialog.members.total"
                v-model:current-page="showMemDialog.query.pageNum"
                :page-size="showMemDialog.query.pageSize"
            />

            <el-dialog width="400px" title="添加成员" :before-close="cancelAddMember" v-model="showMemDialog.addVisible">
                <el-form :model="showMemDialog.memForm" label-width="70px">
                    <el-form-item label="账号:">
                        <el-select
                            style="width: 100%"
                            remote
                            :remote-method="getAccount"
                            v-model="showMemDialog.memForm.accountId"
                            filterable
                            placeholder="请输入账号模糊搜索并选择"
                        >
                            <el-option v-for="item in showMemDialog.accounts" :key="item.id" :label="item.username" :value="item.id"> </el-option>
                        </el-select>
                    </el-form-item>
                    <!-- <el-form-item label="描述:">
                        <el-input v-model="showEnvDialog.envForm.remark" auto-complete="off"></el-input>
                    </el-form-item> -->
                </el-form>
                <template #footer>
                    <div class="dialog-footer">
                        <el-button @click="cancelAddMember()">取 消</el-button>
                        <el-button v-auth="permissions.saveMember" @click="addMember" type="primary" :loading="btnLoading">确 定</el-button>
                    </div>
                </template>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { projectApi } from './api';
import { accountApi } from '../../system/api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { notEmpty, notNull } from '@/common/assert';
export default defineComponent({
    name: 'ProjectList',
    components: {},
    setup() {
        const state = reactive({
            permissions: {
                saveProject: 'project:save',
                delProject: 'project:del',
                saveMember: 'project:member:add',
                delMember: 'project:member:del',
                saveEnv: 'project:env:add',
            },
            query: {
                pageNum: 1,
                pageSize: 10,
                name: null,
            },
            total: 0,
            projects: [],
            btnLoading: false,
            chooseId: null as any,
            chooseData: null as any,
            addProjectDialog: {
                title: '新增项目',
                visible: false,
                form: { name: '', remark: '' },
            },
            showEnvDialog: {
                visible: false,
                chooseId: null,
                chooseData: null,
                envs: [],
                title: '',
                addVisible: false,
                envForm: {
                    name: '',
                    remark: '',
                    projectId: 0,
                },
            },
            showMemDialog: {
                visible: false,
                chooseId: null,
                chooseData: null,
                query: {
                    pageSize: 8,
                    pageNum: 1,
                    projectId: null,
                },
                members: {
                    list: [],
                    total: null,
                },
                title: '',
                addVisible: false,
                memForm: {},
                accounts: [],
            },
        });

        onMounted(() => {
            search();
        });

        const search = async () => {
            let res = await projectApi.projects.request(state.query);
            state.projects = res.list;
            state.total = res.total;
        };

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const showAddProjectDialog = (data: any) => {
            if (data) {
                state.addProjectDialog.form = { ...data };
            } else {
                state.addProjectDialog.form = {} as any;
            }
            state.addProjectDialog.visible = true;
        };

        const cancelAddProject = () => {
            state.addProjectDialog.visible = false;
            state.addProjectDialog.form = {} as any;
        };

        const addProject = async () => {
            const form = state.addProjectDialog.form as any;
            notEmpty(form.name, '项目名不能为空');
            notEmpty(form.remark, '项目描述不能为空');

            await projectApi.saveProject.request(form);
            ElMessage.success('保存成功');
            search();
            cancelAddProject();
        };

        const delProject = async () => {
            try {
                await ElMessageBox.confirm(`确定删除该项目?`, '提示', {
                    confirmButtonText: '确定',
                    cancelButtonText: '取消',
                    type: 'warning',
                });
                await projectApi.delProject.request({ id: state.chooseId });
                ElMessage.success('删除成功');
                state.chooseData = null;
                state.chooseId = null;
                search();
            } catch (err) {}
        };

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.chooseId = item.id;
            state.chooseData = item;
        };

        const showMembers = async (project: any) => {
            state.showMemDialog.query.projectId = project.id;
            await setMemebers();
            state.showMemDialog.title = `${project.name}的成员信息`;
            state.showMemDialog.visible = true;
        };

        /**
         * 选中成员
         */
        const chooseMember = (item: any) => {
            if (!item) {
                return;
            }
            state.showMemDialog.chooseData = item;
            state.showMemDialog.chooseId = item.id;
        };

        const deleteMember = async () => {
            notNull(state.showMemDialog.chooseData, '请选选择成员');
            await projectApi.deleteProjectMem.request(state.showMemDialog.chooseData);
            ElMessage.success('移除成功');
            // 重新赋值成员列表
            setMemebers();
        };

        /**
         * 设置成员列表信息
         */
        const setMemebers = async () => {
            const res = await projectApi.projectMems.request(state.showMemDialog.query);
            state.showMemDialog.members.list = res.list;
            state.showMemDialog.members.total = res.total;
        };

        const showEnv = async (project: any) => {
            state.showEnvDialog.envs = await projectApi.projectEnvs.request({ projectId: project.id });
            state.showEnvDialog.title = `${project.name}的环境信息`;
            state.showEnvDialog.visible = true;
        };

        const chooseEnv = (item: any) => {
            if (!item) {
                return;
            }
            state.showEnvDialog.chooseData = item;
            state.showEnvDialog.chooseId = item.id;
        };

        const deleteEnv = async () => {
            notNull(state.showEnvDialog.chooseData, '请选选择环境');
            await projectApi.delProjectEnvs.request({ id: state.showEnvDialog.chooseId });
            ElMessage.success('删除成功');
            state.showEnvDialog.envs = await projectApi.projectEnvs.request({ projectId: state.chooseId });
        };

        const showAddMemberDialog = () => {
            state.showMemDialog.addVisible = true;
        };

        const addMember = async () => {
            const memForm = state.showMemDialog.memForm as any;
            memForm.projectId = state.chooseData.id;
            notEmpty(memForm.accountId, '请先选择账号');

            await projectApi.saveProjectMem.request(memForm);
            ElMessage.success('保存成功');
            setMemebers();
            cancelAddMember();
        };

        const cancelAddMember = () => {
            state.showMemDialog.memForm = {};
            state.showMemDialog.addVisible = false;
            state.showMemDialog.chooseData = null;
            state.showMemDialog.chooseId = null;
        };

        const getAccount = (username: any) => {
            accountApi.list.request({ username }).then((res) => {
                state.showMemDialog.accounts = res.list;
            });
        };

        const showAddEnvDialog = () => {
            state.showEnvDialog.addVisible = true;
        };

        const addEnv = async () => {
            const envForm = state.showEnvDialog.envForm;
            envForm.projectId = state.chooseData.id;
            await projectApi.saveProjectEnv.request(envForm);
            ElMessage.success('保存成功');
            state.showEnvDialog.envs = await projectApi.projectEnvs.request({ projectId: envForm.projectId });
            cancelAddEnv();
        };

        const cancelAddEnv = () => {
            state.showEnvDialog.envForm = {} as any;
            state.showEnvDialog.addVisible = false;
        };

        return {
            ...toRefs(state),
            search,
            handlePageChange,
            choose,
            showAddProjectDialog,
            addProject,
            delProject,
            cancelAddProject,
            showMembers,
            setMemebers,
            showEnv,
            deleteEnv,
            showAddMemberDialog,
            addMember,
            chooseMember,
            deleteMember,
            cancelAddMember,
            showAddEnvDialog,
            chooseEnv,
            addEnv,
            cancelAddEnv,
            getAccount,
        };
    },
});
</script>
<style lang="scss">
</style>
