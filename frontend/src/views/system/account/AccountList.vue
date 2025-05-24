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
                <el-button v-auth="perms.addAccount" type="primary" icon="plus" @click="onEditAccount(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.delAccount" :disabled="state.selectionData.length < 1" @click="onDeleteAccount()" type="danger" icon="delete">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <template #action="{ data }">
                <el-button link v-if="actionBtns[perms.addAccount]" @click="onEditAccount(data)" type="primary">{{ $t('common.edit') }}</el-button>

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
        <account-edit :title="accountDialog.title" v-model:visible="accountDialog.visible" v-model:account="accountDialog.data" @val-change="onValChange()" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import RoleAllocation from './RoleAllocation.vue';
import AccountEdit from './AccountEdit.vue';
import { AccountStatusEnum } from '../enums';
import { accountApi } from '../api';
import { formatDate } from '@/common/utils/format';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import { useI18nCreateTitle, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nEditTitle, useI18nOperateSuccessMsg } from '@/hooks/useI18n';

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

const pageTableRef: Ref<any> = ref(null);
const state = reactive({
    /**
     * 选中的数据
     */
    selectionData: [],
    /**
     * 查询条件
     */
    query: {
        username: '',
        pageNum: 1,
        pageSize: 0,
    },
    showRoleDialog: {
        title: '',
        visible: false,
        accountRoles: [],
    },
    showResourceDialog: {
        title: '',
        visible: false,
        resources: [],
        defaultProps: {
            children: 'children',
            label: 'name',
        },
    },
    roleDialog: {
        visible: false,
        account: null as any,
        roles: [],
    },
    accountDialog: {
        title: '',
        visible: false,
        data: null as any,
    },
});

const { selectionData, query, showRoleDialog, roleDialog, accountDialog } = toRefs(state);

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
});

const search = async () => {
    pageTableRef.value.search();
};

const onChangeStatus = async (row: any) => {
    let id = row.id;
    let status = row.status == AccountStatusEnum.Disable.value ? AccountStatusEnum.Enable.value : AccountStatusEnum.Disable.value;
    await accountApi.changeStatus.request({
        id,
        status,
    });
    useI18nOperateSuccessMsg();
    search();
};

const onResetOtpSecret = async (row: any) => {
    let id = row.id;
    await accountApi.resetOtpSecret.request({
        id,
    });
    useI18nOperateSuccessMsg();
    row.otpSecret = '-';
};

const onEditAccount = (data: any) => {
    if (!data) {
        state.accountDialog.title = useI18nCreateTitle('personal.accountInfo');
        state.accountDialog.data = null;
    } else {
        state.accountDialog.title = useI18nEditTitle('personal.accountInfo');
        state.accountDialog.data = data;
    }
    state.accountDialog.visible = true;
};

const onShowRoleEdit = (data: any) => {
    state.roleDialog.visible = true;
    state.roleDialog.account = data;
};

const onCancel = () => {
    state.roleDialog.visible = false;
    state.roleDialog.account = null;
};

const onValChange = () => {
    state.accountDialog.visible = false;
    search();
};

const onDeleteAccount = async () => {
    await useI18nDeleteConfirm(state.selectionData.map((x: any) => x.username).join('、'));
    await accountApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
    useI18nDeleteSuccessMsg();
    search();
};
</script>
<style lang="scss"></style>
