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
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { useI18n } from 'vue-i18n';
import { resourceApi, roleApi } from '../api';
import { RoleStatusEnum } from '../enums';
import type { SysRole } from '../types';
import type { SysResource } from '../types';

const RoleEdit = defineAsyncComponent(() => import('./RoleEdit.vue'));
const ShowResource = defineAsyncComponent(() => import('./ShowResource.vue'));
const ResourceEdit = defineAsyncComponent(() => import('./ResourceEdit.vue'));
const AccountAllocation = defineAsyncComponent(() => import('./AccountAllocation.vue'));

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

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
const state = reactive({
    query: {
        pageNum: 1,
        pageSize: 0,
        name: null,
    },
    selectionData: [] as SysRole[],
    resourceDialog: {
        visible: false,
        role: null as SysRole | null,
        resources: [] as SysResource[],
        defaultCheckedKeys: [] as number[],
    },
    roleEditDialog: {
        title: '',
        visible: false,
        role: false as SysRole | false,
    },
    showResourceDialog: {
        visible: false,
        resources: [] as SysResource[],
        title: '',
    },
    accountAllocationDialog: {
        visible: false,
        role: null as SysRole | null,
    },
});

const { query, selectionData, resourceDialog, roleEditDialog, showResourceDialog, accountAllocationDialog } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.value.push(actionColumn);
    }
});

const search = () => {
    pageTableRef.value?.search();
};

const roleEditChange = () => {
    Msg.saveSuccess();
    search();
};

const editRole = (data: SysRole | false) => {
    if (data) {
        state.roleEditDialog.title = useI18nEditTitle('common.role');
        state.roleEditDialog.role = data;
    } else {
        state.roleEditDialog.title = useI18nCreateTitle('common.role');
        state.roleEditDialog.role = false;
    }

    state.roleEditDialog.visible = true;
};

const deleteRole = async (data: SysRole[]) => {
    try {
        await useI18nDeleteConfirm(data.map((x: SysRole) => x.name).join('、'));
        await roleApi.del.request({
            id: data.map((x: SysRole) => x.id).join(','),
        });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        //
    }
};

const showResources = async (row: SysRole) => {
    state.showResourceDialog.resources = await roleApi.roleResources.request({
        id: row.id,
    });
    state.showResourceDialog.title = t('system.role.rolePermissionTitle', { name: row.name });
    state.showResourceDialog.visible = true;
};

const editResource = async (row: SysRole) => {
    let menus = await resourceApi.list.request(null);
    // 获取所有菜单列表
    state.resourceDialog.resources = menus;
    // 获取该角色拥有的菜单id
    let roles = await roleApi.roleResourceIds.request({
        id: row.id,
    });
    let hasIds = roles ? roles : [];
    let hasLeafIds: number[] = [];
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

const showAccountAllocation = (data: SysRole) => {
    state.accountAllocationDialog.role = data;
    state.accountAllocationDialog.visible = true;
};

/**
 * 获取所有菜单树的叶子节点
 * @param {Object} trees  菜单树列表
 */
const getAllLeafIds = (trees: SysResource[]): number[] => {
    let leafIds: number[] = [];
    for (let tree of trees) {
        setLeafIds(tree, leafIds);
    }
    return leafIds;
};

const setLeafIds = (tree: SysResource, ids: number[]) => {
    if (tree.children != null) {
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
        state.resourceDialog.role = null;
        state.resourceDialog.defaultCheckedKeys = [];
    }, 10);
};
</script>
<style lang="scss"></style>
