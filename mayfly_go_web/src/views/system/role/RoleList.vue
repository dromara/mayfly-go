<template>
    <div>
        <page-table
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
            :page-api="roleApi.list"
            ref="pageTableRef"
        >
            <template #tableHeader>
                <el-button v-auth="perms.addRole" type="primary" icon="plus" @click="editRole(false)">添加</el-button>
                <el-button v-auth="perms.delRole" :disabled="selectionData.length < 1" @click="deleteRole(selectionData)" type="danger" icon="delete"
                    >删除</el-button
                >
            </template>

            <template #action="{ data }">
                <el-button v-if="actionBtns[perms.updateRole]" @click="editRole(data)" type="primary" link>编辑</el-button>
                <el-button @click="showResources(data)" type="info" link>权限详情</el-button>
                <el-button v-if="actionBtns[perms.saveRoleResource]" @click="editResource(data)" type="success" link>权限分配</el-button>
                <el-button
                    v-if="actionBtns[perms.saveAccountRole]"
                    :disabled="data.code?.indexOf('COMMON') == 0"
                    @click="showAccountAllocation(data)"
                    type="success"
                    link
                    >用户管理</el-button
                >
            </template>
        </page-table>

        <role-edit :title="roleEditDialog.title" v-model:visible="roleEditDialog.visible" :data="roleEditDialog.role" @val-change="roleEditChange" />
        <resource-edit
            v-model:visible="resourceDialog.visible"
            :role="resourceDialog.role"
            :resources="resourceDialog.resources"
            :defaultCheckedKeys="resourceDialog.defaultCheckedKeys"
            @cancel="cancelEditResources()"
        />

        <account-allocation v-model:visible="accountAllocationDialog.visible" :role="accountAllocationDialog.role" />

        <show-resource v-model:visible="showResourceDialog.visible" :title="showResourceDialog.title" v-model:resources="showResourceDialog.resources" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import RoleEdit from './RoleEdit.vue';
import ResourceEdit from './ResourceEdit.vue';
import ShowResource from './ShowResource.vue';
import { roleApi, resourceApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { RoleStatusEnum } from '../enums';
import { SearchItem } from '@/components/SearchForm';
import AccountAllocation from './AccountAllocation.vue';

const perms = {
    addRole: 'role:add',
    delRole: 'role:del',
    updateRole: 'role:update',
    saveRoleResource: 'role:saveResources',
    saveAccountRole: 'account:saveRoles',
};

const searchItems = [SearchItem.input('name', '角色名')];
const columns = ref([
    TableColumn.new('name', '角色名称'),
    TableColumn.new('code', '角色code'),
    TableColumn.new('remark', '备注'),
    TableColumn.new('status', '状态').typeTag(RoleStatusEnum),
    TableColumn.new('creator', '创建账号'),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('modifier', '更新账号'),
    TableColumn.new('updateTime', '更新时间').isTime(),
]);

const actionBtns = hasPerms([perms.updateRole, perms.saveRoleResource, perms.saveAccountRole]);
const actionColumn = TableColumn.new('action', '操作').isSlot().setMinWidth(290).fixedRight().alignCenter();

const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
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
    accountAllocationDialog: {
        visible: false,
        role: {},
    },
});

const { query, selectionData, resourceDialog, roleEditDialog, showResourceDialog, accountAllocationDialog } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = () => {
    pageTableRef.value.search();
};

const roleEditChange = () => {
    ElMessage.success('修改成功！');
    search();
};

const editRole = (data: any) => {
    if (data) {
        state.roleEditDialog.role = data;
    } else {
        state.roleEditDialog.role = false;
    }

    state.roleEditDialog.visible = true;
};

const deleteRole = async (data: any) => {
    try {
        await ElMessageBox.confirm(
            `此操作将删除【${data.map((x: any) => x.name).join(', ')}】该角色，以及与该角色有关的账号角色关联信息和资源角色关联信息, 是否继续?`,
            '提示',
            {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            }
        );
        await roleApi.del.request({
            id: data.map((x: any) => x.id).join(','),
        });
        ElMessage.success('删除成功！');
        search();
    } catch (err) {
        //
    }
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

const showAccountAllocation = (data: any) => {
    state.accountAllocationDialog.role = data;
    state.accountAllocationDialog.visible = true;
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
