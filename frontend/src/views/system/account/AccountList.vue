<template>
    <div class="h-full">
        <page-table
            ref="pageTableRef"
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

            <template #action="{ data }">
                <el-button link v-if="actionBtns[perms.addAccount]" @click="editEntity(data)" type="primary">{{ $t('common.edit') }}</el-button>

                <el-button link v-if="actionBtns[perms.saveAccountRole]" @click="onShowRoleEdit(data)" type="success">
                    {{ $t('system.account.roleAllocation') }}
                </el-button>

                <el-button link v-if="actionBtns[perms.changeAccountStatus] && data.status == 1" @click="onChangeStatus(data)" type="danger">
                    {{ $t('common.disable') }}
                </el-button>

                <el-button link v-if="actionBtns[perms.changeAccountStatus] && data.status == -1" type="success" @click="onChangeStatus(data)">
                    {{ $t('common.enable') }}
                </el-button>

                <el-button
                    link
                    v-if="actionBtns[perms.addAccount]"
                    :disabled="!data.otpSecret || data.otpSecret == '-'"
                    @click="onResetOtpSecret(data)"
                    type="warning"
                >
                    {{ $t('system.account.resetOtp') }}
                </el-button>
            </template>
        </page-table>

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
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem } from '@/components/page-table/SearchForm';
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
const actionColumn = TableColumn.new('action', 'common.operation').isSlot().fixedRight().setMinWidth(260).noShowOverflowTooltip().alignCenter();

const pageTableRef = useTemplateRef<InstanceType<typeof PageTable>>('pageTableRef');
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
    pageTableRef.value?.search();
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
    try {
        await useI18nDeleteConfirm(selectionData.value.map((x) => x.username).join('、'));
    } catch {
        return; // 用户取消
    }
    await accountApi.del.request({ id: selectionData.value.map((x) => x.id).join(',') });
    Msg.deleteSuccess();
    search();
};
</script>
<style lang="scss"></style>
