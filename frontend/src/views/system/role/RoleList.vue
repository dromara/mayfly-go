<template>
    <div class="h-full">
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
                <el-button v-auth="perms.addRole" type="primary" icon="plus" @click="editRole(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delRole" :disabled="selectionData.length < 1" @click="deleteRole(selectionData)" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #action="{ data }">
                <el-button v-if="actionBtns[perms.updateRole]" @click="editRole(data)" type="primary" link>{{ $t('common.edit') }}</el-button>
                <el-button @click="showResources(data)" type="info" link>{{ $t('system.role.permissionDetail') }}</el-button>
                <el-button v-if="actionBtns[perms.saveRoleResource]" @click="editResource(data)" type="success" link>
                    {{ $t('system.role.permissionAllocate') }}
                </el-button>
                <el-button
                    v-if="actionBtns[perms.saveAccountRole]"
                    :disabled="data.code?.indexOf('COMMON') == 0"
                    @click="showAccountAllocation(data)"
                    type="success"
                    link
                    >{{ $t('system.role.userManage') }}</el-button
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
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { RoleStatusEnum } from '../enums';
import { SearchItem } from '@/components/SearchForm';
import AccountAllocation from './AccountAllocation.vue';
import { useI18n } from 'vue-i18n';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nSaveSuccessMsg } from '@/hooks/useI18n';

const { t } = useI18n();

const perms = {
    addRole: 'role:add',
    delRole: 'role:del',
    updateRole: 'role:update',
    saveRoleResource: 'role:saveResources',
    saveAccountRole: 'account:saveRoles',
};

const searchItems = [SearchItem.input('name', 'system.role.roleName')];
const columns = ref([
    TableColumn.new('name', 'system.role.roleName'),
    TableColumn.new('code', 'system.role.roleCode'),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('status', 'common.status').typeTag(RoleStatusEnum),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
]);

const actionBtns = hasPerms([perms.updateRole, perms.saveRoleResource, perms.saveAccountRole]);
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().setMinWidth(300).fixedRight().noShowOverflowTooltip().alignCenter();

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
        title: '',
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
    useI18nSaveSuccessMsg();
    search();
};

const editRole = (data: any) => {
    if (data) {
        state.roleEditDialog.title = useI18nEditTitle('common.role');
        state.roleEditDialog.role = data;
    } else {
        state.roleEditDialog.title = useI18nCreateTitle('common.role');
        state.roleEditDialog.role = false;
    }

    state.roleEditDialog.visible = true;
};

const deleteRole = async (data: any) => {
    try {
        await useI18nDeleteConfirm(data.map((x: any) => x.name).join('、'));
        await roleApi.del.request({
            id: data.map((x: any) => x.id).join(','),
        });
        useI18nDeleteSuccessMsg();
        search();
    } catch (err) {
        //
    }
};

const showResources = async (row: any) => {
    state.showResourceDialog.resources = await roleApi.roleResources.request({
        id: row.id,
    });
    state.showResourceDialog.title = t('system.role.rolePermissionTitle', { name: row.name });
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
