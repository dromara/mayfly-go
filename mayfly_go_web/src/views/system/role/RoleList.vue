<template>
    <div>
        <page-table ref="pageTableRef" :query="state.queryConfig" v-model:query-form="query" :show-selection="true"
            v-model:selection-data="selectionData" :data="roles" :columns="state.columns" :total="total"
            v-model:page-size="query.pageSize" v-model:page-num="query.pageNum" @pageChange="search()">

            <template #queryRight>
                <el-button v-auth="'role:add'" type="primary" icon="plus" @click="editRole(false)">添加</el-button>
                <el-button v-auth="'role:update'" :disabled="selectionData.length != 1" @click="editRole(selectionData)"
                    type="primary" icon="edit">编辑</el-button>
                <el-button v-auth="'role:saveResources'" :disabled="selectionData.length != 1"
                    @click="editResource(selectionData)" type="success" icon="setting">分配菜单&权限</el-button>
                <el-button v-auth="'role:del'" :disabled="selectionData.length < 1" @click="deleteRole(selectionData)"
                    type="danger" icon="delete">删除</el-button>
            </template>

            <template #status="{ data }">
                <el-tag v-if="data.status == 1" type="success" size="small">正常</el-tag>
                <el-tag v-if="data.status == -1" type="danger" size="small">禁用</el-tag>
            </template>

            <template #showmore="{ data }">
                <el-link @click.prevent="showResources(data)" type="info">菜单&权限</el-link>
            </template>
        </page-table>

        <role-edit :title="roleEditDialog.title" v-model:visible="roleEditDialog.visible" :data="roleEditDialog.role"
            @val-change="roleEditChange" />
        <resource-edit v-model:visible="resourceDialog.visible" :role="resourceDialog.role"
            :resources="resourceDialog.resources" :defaultCheckedKeys="resourceDialog.defaultCheckedKeys"
            @cancel="cancelEditResources()" />
        <show-resource v-model:visible="showResourceDialog.visible" :title="showResourceDialog.title"
            v-model:resources="showResourceDialog.resources" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted } from 'vue';
import RoleEdit from './RoleEdit.vue';
import ResourceEdit from './ResourceEdit.vue';
import ShowResource from './ShowResource.vue';
import { roleApi, resourceApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const pageTableRef: any = ref(null)

const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 10,
        name: null,
    },
    queryConfig: [
        TableQuery.text("name", "角色名"),
    ],
    columns: [
        TableColumn.new("name", "角色名称"),
        TableColumn.new("code", "角色code"),
        TableColumn.new("remark", "备注"),
        TableColumn.new("status", "状态").setSlot("status"),
        TableColumn.new("creator", "创建账号"),
        TableColumn.new("createTime", "创建时间").isTime(),
        TableColumn.new("modifier", "更新账号"),
        TableColumn.new("updateTime", "更新时间").isTime(),
        TableColumn.new("showmore", "查看更多").setSlot("showmore").setMinWidth(150).fixedRight(),
    ],
    total: 0,
    roles: [],
    selectionData: [],
    resourceDialog: {
        visible: false,
        role: {},
        resources: [],
        defaultCheckedKeys: [],
    },
    roleEditDialog: {
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

const {
    query,
    total,
    roles,
    selectionData,
    resourceDialog,
    roleEditDialog,
    showResourceDialog,
} = toRefs(state)

onMounted(() => {
    search();
});

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        let res = await roleApi.list.request(state.query);
        state.roles = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
};

const roleEditChange = () => {
    ElMessage.success('修改成功！');
    search();
};

const editRole = (data: any) => {
    if (data) {
        state.roleEditDialog.role = data[0];
    } else {
        state.roleEditDialog.role = false;
    }

    state.roleEditDialog.visible = true;
};

const deleteRole = async (data: any) => {
    try {
        await ElMessageBox.confirm(
            `此操作将删除【${data.map((x: any) => x.name).join(", ")}】该角色，以及与该角色有关的账号角色关联信息和资源角色关联信息, 是否继续?`,
            '提示',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            }
        );
        await roleApi.del.request({
            id: data.map((x: any) => x.id).join(","),
        });
        ElMessage.success('删除成功！');
        search();
    } catch (err) { }
};

const showResources = async (row: any) => {
    state.showResourceDialog.resources = await roleApi.roleResources.request({
        id: row.id,
    });
    state.showResourceDialog.title = '"' + row.name + '"的菜单&权限';
    state.showResourceDialog.visible = true;
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
</script>
<style lang="scss"></style>
