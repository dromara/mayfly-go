<template>
    <div>
        <div class="toolbar">
            <el-button @click="showSaveApiDialog" v-auth="permissions.saveProject" type="primary" icon="el-icon-plus" size="mini">添加</el-button>
            <el-button
                @click="showSaveApiDialog(chooseData)"
                v-auth="permissions.saveProject"
                :disabled="chooseId == null"
                type="primary"
                icon="el-icon-edit"
                size="mini"
                >编辑</el-button
            >

            <el-button v-auth="'role:del'" :disabled="chooseId == null" type="danger" icon="el-icon-delete" size="mini">删除</el-button>

            <div style="float: right">
                <el-select v-model="query.serviceId" @change="changeService" placeholder="请选择服务" filterable size="small">
                    <el-option v-for="item in services" :key="item.id" :label="`${item.name}`" :value="item.id"> </el-option>
                </el-select>
                <el-input
                    class="mr2 ml2"
                    placeholder="请输入服务名！"
                    size="small"
                    style="width: 140px"
                    v-model="query.name"
                    @clear="search"
                    clearable
                ></el-input>
                <el-button @click="search" type="success" icon="el-icon-search" size="mini"></el-button>
            </div>
        </div>
        <el-table :data="apis" @current-change="choose" border ref="table" style="width: 100%">
            <el-table-column label="选择" width="50px">
                <template #default="scope">
                    <el-radio v-model="chooseId" :label="scope.row.id">
                        <i></i>
                    </el-radio>
                </template>
            </el-table-column>
            <el-table-column prop="serviceName" label="服务"></el-table-column>
            <el-table-column prop="name" label="名称" show-overflow-tooltip></el-table-column>
            <el-table-column prop="code" label="code" show-overflow-tooltip> </el-table-column>
            <el-table-column prop="method" label="method" min-width="45" show-overflow-tooltip> </el-table-column>
            <el-table-column prop="uri" label="uri" show-overflow-tooltip> </el-table-column>
            <el-table-column prop="createTime" label="创建时间">
                <template #default="scope">
                    {{ $filters.dateFormat(scope.row.createTime) }}
                </template>
            </el-table-column>
            <el-table-column prop="creator" label="创建者" min-width="50"> </el-table-column>
            <el-table-column label="操作" min-width="50">
                <template #default="scope">
                    <el-button
                        @click="syncServiceApi(scope.row)"
                        type="success"
                        icom="el-icon-tickets"
                        size="mini"
                        plain
                        :disabled="scope.row.canSync == -1"
                        >同步</el-button
                    >
                </template>
            </el-table-column>
            <!-- <el-table-column label="查看更多" min-width="80px">
                <template #default="scope">
                    <el-link @click.prevent="showMembers(scope.row)" type="success">成员</el-link>

                    <el-link class="ml5" @click.prevent="showEnv(scope.row)" type="info">环境</el-link>
                </template>
            </el-table-column> -->
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

        <api-edit :services="services" v-model:visible="saveApiDialog.visible" :api="saveApiDialog.form" @val-change="valChange"/>

        <!-- <el-dialog width="400px" title="服务编辑" :before-close="cancelAddService" v-model="addServiceDialog.visible">
            <el-form :model="addServiceDialog.form" size="small" label-width="70px">
                <el-form-item label="服务名:" required>
                    <el-input :disabled="addServiceDialog.form.id ? true : false" v-model="addServiceDialog.form.name" auto-complete="off"></el-input>
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
        </el-dialog> -->
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { serviceApi } from '../api';
import { ElMessage } from 'element-plus';
import { notEmpty } from '@/common/assert';
import ApiEdit from './ApiEdit.vue'
export default defineComponent({
    name: 'ServiceList',
    components: {
        ApiEdit
    },
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
            apis: [],
            services: [],
            btnLoading: false,
            chooseId: null as any,
            chooseData: null as any,
            saveApiDialog: {
                title: '新增api',
                visible: false,
                form: { name: '', remark: '' },
            },
        });

        onMounted(() => {
            getServices();
            // search();
        });

        const search = async () => {
            let res = await serviceApi.serviceApis.request(state.query);
            state.apis = res.list;
            state.total = res.total;
        };

        const getServices = async () => {
            let res = await serviceApi.services.request({ pateNum: 1, pageSize: 100 });
            state.services = res.list;
        };

        const changeService = async () => {
            search()
        }

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const showSaveApiDialog = (data: any) => {
            if (data) {
                state.saveApiDialog.form = { ...data };
            } else {
                state.saveApiDialog.form = {} as any;
            }
            state.saveApiDialog.visible = true;
        };

        const cancelAddApi = () => {
            state.saveApiDialog.visible = false;
            state.saveApiDialog.form = {} as any;
        };

        const saveService = async () => {
            const form = state.saveApiDialog.form as any;
            notEmpty(form.name, '服务名不能为空');
            notEmpty(form.remark, '服务描述不能为空');

            await serviceApi.saveService.request(form);
            ElMessage.success('保存成功');
            search();
            cancelAddApi();
        };

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.chooseId = item.id;
            state.chooseData = item;
        };

        const syncServiceApi = async (item: any) => {
            await serviceApi.syncServiceApi.request({apiId: item.id, id: item.serviceId})
            ElMessage.success("同步成功")
            item.canSync = -1
        }

        const valChange = () => {
            search();
        }

        

        // const addEnv = async () => {
        //     const envForm = state.showEnvDialog.envForm;
        //     envForm.projectId = state.chooseData.id;
        //     await projectApi.saveProjectEnv.request(envForm);
        //     ElMessage.success('保存成功');
        //     state.showEnvDialog.envs = await projectApi.projectEnvs.request({ projectId: envForm.projectId });
        //     cancelAddEnv();
        // };

        return {
            ...toRefs(state),
            search,
            changeService,
            handlePageChange,
            choose,
            showSaveApiDialog,
            saveService,
            syncServiceApi,
            cancelAddApi,
            valChange,
        };
    },
});
</script>
<style lang="scss">
</style>
