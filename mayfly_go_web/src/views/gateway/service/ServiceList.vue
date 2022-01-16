<template>
    <div>
        <div class="toolbar">
            <el-button @click="showAddServiceDialog" v-auth="permissions.saveProject" type="primary" icon="el-icon-plus" size="mini">添加</el-button>
            <el-button
                @click="showAddServiceDialog(chooseData)"
                v-auth="permissions.saveProject"
                :disabled="chooseId == null"
                type="primary"
                icon="el-icon-edit"
                size="mini"
                >编辑</el-button
            >

            <el-button v-auth="'role:del'" :disabled="chooseId == null" type="danger" icon="el-icon-delete" size="mini">删除</el-button>

            <div style="float: right">
                <el-input
                    class="mr2"
                    placeholder="请输入项目名！"
                    size="small"
                    style="width: 140px"
                    v-model="query.name"
                    @clear="search"
                    clearable
                ></el-input>
                <el-button @click="search" type="success" icon="el-icon-search" size="mini"></el-button>
            </div>
        </div>
        <el-table :data="services" @current-change="choose" border ref="table" style="width: 100%">
            <el-table-column label="选择" width="50px">
                <template #default="scope">
                    <el-radio v-model="chooseId" :label="scope.row.id">
                        <i></i>
                    </el-radio>
                </template>
            </el-table-column>
            <el-table-column prop="name" label="服务名"></el-table-column>
             <el-table-column prop="routePath" label="路由路径"></el-table-column>
            <el-table-column prop="urls" label="服务地址">
                <template #default="scope">
                    {{ scope.row.urls ? scope.row.urls : '注册中心' }}
                </template>
            </el-table-column>
            <el-table-column prop="remark" label="描述" min-width="80px" show-overflow-tooltip></el-table-column>
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
            <el-table-column label="操作" min-width="80px">
                <template #default="scope">
                    <el-button
                        @click="syncService(scope.row)"
                        type="success"
                        icom="el-icon-tickets"
                        size="mini"
                        plain
                        :disabled="scope.row.canSync == -1"
                        >同步</el-button
                    >
                </template>
            </el-table-column>
        </el-table>
        <el-pagination
            @current-change="handlePageChange"
            style="text-align: center"
            background
            layout="prev, pager, next, total, jumper"
            :total="total"
            v-model:current-page="query.pageNum"
            :page-size="query.pageSize"
        />

        <el-dialog width="400px" title="服务编辑" :before-close="cancelAddService" v-model="addServiceDialog.visible">
            <el-form :model="addServiceDialog.form" size="small" label-width="85px">
                <el-form-item label="服务名:" required>
                    <el-input :disabled="addServiceDialog.form.id ? true : false" v-model="addServiceDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>
                 <el-form-item label="路由路径:" required>
                    <el-input v-model="addServiceDialog.form.routePath" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="描述:" required>
                    <el-input v-model="addServiceDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="地址:">
                    <el-input v-model="addServiceDialog.form.urls" auto-complete="off" placeholder="不填则注册中心获取"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="saveService" type="primary" size="small">确 定</el-button>
                    <el-button @click="cancelAddService()" size="small">取 消</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { serviceApi } from '../api';
import { ElMessage } from 'element-plus';
import { notEmpty } from '@/common/assert';
export default defineComponent({
    name: 'ServiceList',
    components: {},
    setup() {
        const state = reactive({
            permissions: {
                saveProject: 'project:save',
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
            services: [],
            btnLoading: false,
            chooseId: null as any,
            chooseData: null as any,
            addServiceDialog: {
                title: '新增服务',
                visible: false,
                form: { name: '', remark: '' },
            },
        });

        onMounted(() => {
            search();
        });

        const search = async () => {
            let res = await serviceApi.services.request(state.query);
            state.services = res.list;
            state.total = res.total;
        };

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const showAddServiceDialog = (data: any) => {
            if (data) {
                state.addServiceDialog.form = {...data};
            } else {
                state.addServiceDialog.form = {} as any;
            }
            state.addServiceDialog.visible = true;
        };

        const cancelAddService = () => {
            state.addServiceDialog.visible = false;
            state.addServiceDialog.form = {} as any;
        };

        const saveService = async () => {
            const form = state.addServiceDialog.form as any;
            notEmpty(form.name, '服务名不能为空');
            notEmpty(form.remark, '服务描述不能为空');

            await serviceApi.saveService.request(form);
            ElMessage.success('保存成功');
            search();
            cancelAddService();
        };

        const syncService = async (item: any) => {
            await serviceApi.syncService.request({id: item.id})
            ElMessage.success("同步成功")
            item.canSync = -1
        }

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.chooseId = item.id;
            state.chooseData = item;
        };

        // const addEnv = async () => {
        //     const envForm = state.showEnvDialog.envForm;
        //     envForm.projectId = state.chooseData.id;
        //     await projectApi.saveProjectEnv.request(envForm);
        //     ElMessage.success('保存成功');
        //     state.showEnvDialog.envs = await projectApi.projectEnvs.request({ projectId: envForm.projectId });
        //     cancelAddEnv();
        // };

        // const roleEditChange = (data: any) => {
        //     ElMessage.success('修改成功！');
        //     search();
        // };

        return {
            ...toRefs(state),
            search,
            handlePageChange,
            choose,
            showAddServiceDialog,
            saveService,
            syncService,
            cancelAddService,
        };
    },
});
</script>
<style lang="scss">
</style>
