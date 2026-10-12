<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="accountApi.list"
            :search-items="searchItems"
            v-model:query-form="query"
            :show-selection="true"
            v-model:selection-data="selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.addAccount" type="primary" icon="plus" @click="editEntity()">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delAccount" :disabled="selectionData.length < 1" @click="onDelete" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
        </data-table>

        <el-dialog width="500px" :title="showRoleDialog.title" v-model="showRoleDialog.visible">
            <el-table border :data="showRoleDialog.accountRoles">
                <el-table-column property="name" :label="$t('system.role.roleName')" width="125"></el-table-column>
                <el-table-column property="creator" :label="$t('system.account.assigner')" width="125"></el-table-column>
                <el-table-column property="createTime" :label="$t('system.account.allocateTime')">
                    <template #default="scope">
                        {{ formatDate(scope.row.createTime) }}
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>

        <role-allocation v-model:visible="roleDialog.visible" :account="roleDialog.account" @cancel="onCancel()" />
        <account-edit :title="editDialog.title" v-model:visible="editDialog.visible" v-model:data="editDialog.data" @val-change="onValChange" />
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { useEditDialog } from '@/hooks/useResourceForm';
import { defineAsyncComponent, onMounted, ref, useTemplateRef } from 'vue';
import { accountApi } from '../api';
import { AccountStatusEnum } from '../enums';
import type { Account } from '../types';

const AccountEdit = defineAsyncComponent(() => import('./AccountEdit.vue'));
const RoleAllocation = defineAsyncComponent(() => import('./RoleAllocation.vue'));

const perms = {
    addAccount: 'account:add',
    delAccount: 'account:del',
    saveAccountRole: 'account:saveRoles',
    changeAccountStatus: 'account:changeStatus',
};

const searchItems = [SearchItem.input('username', 'common.username')];
const columns = [
    TableColumn.new('name', 'system.account.name'),
    TableColumn.new('username', 'common.username'),
    TableColumn.new('mobile', 'common.mobile'),
    TableColumn.new('email', 'common.email'),
    TableColumn.new('status', 'common.status').typeTag(AccountStatusEnum),
    TableColumn.new('lastLoginTime', 'system.account.lastLoginTime').isTime(),
    TableColumn.new('creator', 'common.creator'),
    TableColumn.new('createTime', 'common.createTime').isTime(),
    TableColumn.new('modifier', 'common.modifier'),
    TableColumn.new('updateTime', 'common.updateTime').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.addAccount, perms.saveAccountRole, perms.changeAccountStatus]);
const actionColumn = TableColumn.new('action', 'common.operation')
    .setActions<Account>([
        { key: 'edit', label: 'common.edit', type: 'primary', perm: perms.addAccount, onClick: (row) => editEntity(row) },
        { key: 'role', label: 'system.account.roleAllocation', type: 'success', perm: perms.saveAccountRole, onClick: (row) => onShowRoleEdit(row) },
        // 启用/禁用互斥：两格各自带行级条件与配色，计宽按同组最宽的一格
        {
            key: 'disable',
            label: 'common.disable',
            type: 'danger',
            group: 'status',
            perm: perms.changeAccountStatus,
            show: (row) => row.status == 1,
            onClick: (row) => onChangeStatus(row),
        },
        {
            key: 'enable',
            label: 'common.enable',
            type: 'success',
            group: 'status',
            perm: perms.changeAccountStatus,
            show: (row) => row.status == -1,
            onClick: (row) => onChangeStatus(row),
        },
        {
            key: 'resetOtp',
            label: 'system.account.resetOtp',
            type: 'warning',
            perm: perms.addAccount,
            disabled: (row) => !row.otpSecret || row.otpSecret == '-',
            onClick: (row) => onResetOtpSecret(row),
        },
    ])
    .fixedRight()
    .noShowOverflowTooltip()
    .alignCenter();

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');
const { editDialog, editEntity } = useEditDialog<Account>('personal.accountInfo');

const selectionData = ref<Account[]>([]);
const query = ref({
    username: '',
    pageNum: 1,
    pageSize: 0,
});

const showRoleDialog = ref({
    title: '',
    visible: false,
    accountRoles: [] as Record<string, unknown>[],
});

const roleDialog = ref({
    visible: false,
    account: null as Account | null,
});

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
});

const search = () => {
    dataTableRef.value?.search();
};

const onChangeStatus = async (row: Account) => {
    const id = row.id;
    const status = row.status == AccountStatusEnum.Disable.value ? AccountStatusEnum.Enable.value : AccountStatusEnum.Disable.value;
    await accountApi.changeStatus.request({ id, status });
    Msg.operateSuccess();
    search();
};

const onResetOtpSecret = async (row: Account) => {
    await accountApi.resetOtpSecret.request({ id: row.id });
    Msg.operateSuccess();
    row.otpSecret = '-';
};

const onShowRoleEdit = (data: Account) => {
    roleDialog.value.visible = true;
    roleDialog.value.account = data;
};

const onCancel = () => {
    roleDialog.value.visible = false;
    roleDialog.value.account = null;
};

const onValChange = () => {
    editDialog.value.visible = false;
    search();
};

const onDelete = async () => {
    if (!(await useI18nDeleteConfirm(selectionData.value.map((x) => x.username).join('、')))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }
    await accountApi.del.request({ id: selectionData.value.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};
</script>
<style lang="scss"></style>
