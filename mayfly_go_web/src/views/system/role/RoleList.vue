<template>
    <div class="role-list">
        <el-card>
            <el-button v-auth="'role:add'" type="primary" icon="plus" @click="editRole(false)">添加</el-button>
            <el-button v-auth="'role:update'" :disabled="chooseId == null" @click="editRole(chooseData)" type="primary" icon="edit">编辑</el-button>
            <el-button v-auth="'role:saveResources'" :disabled="chooseId == null" @click="editResource(chooseData)" type="success" icon="setting"
                >分配菜单&权限</el-button
            >
            <el-button v-auth="'role:del'" :disabled="chooseId == null" @click="deleteRole(chooseData)" type="danger" icon="delete">删除</el-button>

            <div style="float: right">
                <el-input
                    placeholder="请输入角色名称"
                    class="mr2"
                    style="width: 200px"
                    v-model="query.name"
                    @clear="search"
                    clearable
                ></el-input>
                <el-button @click="search" type="success" icon="search"></el-button>
            </div>
            <el-table :data="roles" @current-change="choose" ref="table" style="width: 100%">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="chooseId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="角色名称"></el-table-column>
                <el-table-column prop="code" label="角色code"></el-table-column>
                <el-table-column prop="remark" label="描述" min-width="160px" show-overflow-tooltip></el-table-column>
                <el-table-column prop="createTime" label="创建时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
                <el-table-column prop="updateTime" label="修改时间">
                    <template #default="scope">
                        {{ $filters.dateFormat(scope.row.updateTime) }}
                    </template>
                </el-table-column>
                <el-table-column label="查看更多" min-width="80px">
                    <template #default="scope">
                        <el-link @click.prevent="showResources(scope.row)" type="info">菜单&权限</el-link>
                    </template>
                </el-table-column>
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

        <role-edit :title="roleEdit.title" v-model:visible="roleEdit.visible" :data="roleEdit.role" @val-change="roleEditChange" />
        <resource-edit
            v-model:visible="resourceDialog.visible"
            :role="resourceDialog.role"
            :resources="resourceDialog.resources"
            :defaultCheckedKeys="resourceDialog.defaultCheckedKeys"
            @cancel="cancelEditResources()"
        />
        <show-resource
            v-model:visible="showResourceDialog.visible"
            :title="showResourceDialog.title"
            v-model:resources="showResourceDialog.resources"
        />
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import RoleEdit from './RoleEdit.vue';
import ResourceEdit from './ResourceEdit.vue';
import ShowResource from './ShowResource.vue';
import { roleApi, resourceApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
export default defineComponent({
    name: 'RoleList',
    components: {
        RoleEdit,
        ResourceEdit,
        ShowResource,
    },
    setup() {
        const state = reactive({
            dialogFormVisible: false,
            currentEditPermissions: false,
            query: {
                pageNum: 1,
                pageSize: 10,
                name: null,
            },
            total: 0,
            roles: [],
            chooseId: null,
            chooseData: null,
            resourceDialog: {
                visible: false,
                role: {},
                resources: [],
                defaultCheckedKeys: [],
            },
            roleEdit: {
                title: '角色编辑',
                visible: false,
                role: {},
            },
            showResourceDialog: {
                visible: false,
                resources: [],
                title: '',
            },
        });

        onMounted(() => {
            search();
        });

        const search = async () => {
            let res = await roleApi.list.request(state.query);
            state.roles = res.list;
            state.total = res.total;
        };

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            search();
        };

        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.chooseId = item.id;
            state.chooseData = item;
        };

        const roleEditChange = () => {
            ElMessage.success('修改成功！');
            state.chooseId = null;
            state.chooseData = null;
            search();
        };

        const editRole = (data: any) => {
            if (data) {
                state.roleEdit.role = data;
            } else {
                state.roleEdit.role = false;
            }

            state.roleEdit.visible = true;
        };

        const deleteRole = async (data: any) => {
            try {
                await ElMessageBox.confirm(
                    `此操作将删除 [${data.name}] 该角色，以及与该角色有关的账号角色关联信息和资源角色关联信息, 是否继续?`,
                    '提示',
                    {
                        confirmButtonText: '确定',
                        cancelButtonText: '取消',
                        type: 'warning',
                    }
                );
                await roleApi.del.request({
                    id: data.id,
                });
                ElMessage.success('删除成功！');
                search();
            } catch (err) {}
        };

        const showResources = async (row: any) => {
            state.showResourceDialog.resources = await roleApi.roleResources.request({
                id: row.id,
            });
            state.showResourceDialog.title = '"' + row.name + '"的菜单&权限';
            state.showResourceDialog.visible = true;
        };

        const closeShowResourceDialog = () => {
            state.showResourceDialog.visible = false;
            state.showResourceDialog.resources = [];
        };

        const editResource = async (row: any) => {
            let menus = await resourceApi.list.request(null);
            // 获取所有菜单列表
            state.resourceDialog.resources = menus;
            // 获取该角色拥有的菜单id
            let roles = await roleApi.roleResourceIds.request({
                id: row.id,
            });
            let hasIds = roles ? roles : [];
            let hasLeafIds: any = [];
            // 获取菜单的所有叶子节点
            let leafIds = getAllLeafIds(state.resourceDialog.resources);
            for (let id of leafIds) {
                // 判断角色拥有的菜单id中，是否含有该叶子节点，有则添加进入用户拥有的叶子节点
                if (hasIds.includes(id)) {
                    hasLeafIds.push(id);
                }
            }
            state.resourceDialog.defaultCheckedKeys = hasLeafIds;
            // 显示
            state.resourceDialog.visible = true;
            state.resourceDialog.role = row;
        };

        /**
         * 获取所有菜单树的叶子节点
         * @param {Object} trees  菜单树列表
         */
        const getAllLeafIds = (trees: any) => {
            let leafIds: any = [];
            for (let tree of trees) {
                setLeafIds(tree, leafIds);
            }
            return leafIds;
        };

        const setLeafIds = (tree: any, ids: any) => {
            if (tree.children !== null) {
                for (let t of tree.children) {
                    setLeafIds(t, ids);
                }
            } else {
                ids.push(tree.id);
            }
        };

        /**
         * 取消编辑资源权限树
         */
        const cancelEditResources = () => {
            state.resourceDialog.visible = false;
            setTimeout(() => {
                state.resourceDialog.role = {};
                state.resourceDialog.defaultCheckedKeys = [];
            }, 10);
        };

        return {
            ...toRefs(state),
            search,
            handlePageChange,
            choose,
            roleEditChange,
            editRole,
            deleteRole,
            showResources,
            closeShowResourceDialog,
            editResource,
            cancelEditResources,
        };
    },
});
</script>
<style lang="scss">
</style>
