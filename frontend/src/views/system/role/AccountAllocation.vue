<template>
    <div>
        <el-dialog
            @open="searchRoleAccount()"
            :title="role == null ? '' : $t('system.role.allocateAccountTitle', { roleName: role.name })"
            v-model="dialogVisible"
            :destroy-on-close="true"
            width="55%"
        >
            <data-table ref="dataTableRef" :page-api="roleApi.roleAccounts" :search-items="searchItems" v-model:query-form="query" :columns="columns" lazy>
                <template #tableHeader>
                    <el-button v-auth="perms.saveAccountRole" type="primary" icon="plus" @click="onShowAddAccount()">{{ $t('common.add') }}</el-button>
                </template>

                <!-- 行内操作由表格统一渲染（见 columns 里的 setActions），列宽与按钮同源 -->
            </data-table>

            <el-dialog
                width="400px"
                :title="$t('system.role.addAccount')"
                :before-close="onCancelAddAccount"
                v-model="addAccountDialog.visible"
                :destroy-on-close="true"
            >
                <el-form label-width="auto">
                    <AccountSelectFormItem v-model="addAccountDialog.accountId" :focus="true" />
                </el-form>
                <template #footer>
                    <div class="dialog-footer">
                        <el-button @click="onCancelAddAccount()">{{ $t('common.cancel') }}</el-button>
                        <el-button @click="onRelateAccount(1, addAccountDialog.accountId ?? 0)" type="primary">{{ $t('common.confirm') }}</el-button>
                    </div>
                </template>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { hasPerms } from '@/components/auth/auth';
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg } from '@/hooks/useI18n';
import { onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import AccountSelectFormItem from '../account/components/AccountSelectFormItem.vue';
import { accountApi, roleApi } from '../api';
import { AccountStatusEnum } from '../enums';
import type { Account, AccountRole, SysRole } from '../types';

const props = defineProps({
    role: Object as () => SysRole | null,
});

const perms = {
    saveAccountRole: 'account:saveRoles',
};

const searchItems = [SearchItem.input('name', 'system.account.name'), SearchItem.input('username', 'common.username')];
const columns = [
    TableColumn.new('accountName', 'system.account.name'),
    TableColumn.new('username', 'common.username'),
    TableColumn.new('accountStatus', 'system.role.userStatus').typeTag(AccountStatusEnum),
    TableColumn.new('creator', 'system.role.assigner'),
    TableColumn.new('createTime', 'system.role.allocateTime').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.saveAccountRole]);
const actionColumn = TableColumn.new('action', 'common.operation')
    .setActions<AccountRole>([
        {
            key: 'remove',
            label: 'common.remove',
            type: 'danger',
            icon: 'delete',
            perm: perms.saveAccountRole,
            onClick: (row) => onRelateAccount(-1, row.accountId),
        },
    ])
    .fixedRight()
    .noShowOverflowTooltip()
    .alignCenter();

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const state = reactive({
    /**
     * 查询条件
     */
    query: {
        username: '',
        name: '',
        id: 0,
        pageNum: 1,
        pageSize: 0,
    },
    addAccountDialog: {
        visible: false,
        accounts: [] as Account[],
        accountId: null as number | null,
    },
});

const { query, addAccountDialog } = toRefs(state);

const dialogVisible = defineModel('visible', { default: false });

onMounted(() => {
    if (Object.keys(actionBtns).length > 0) {
        columns.push(actionColumn);
    }
});

const searchRoleAccount = () => {
    state.query.id = props.role?.id || 0;
    dataTableRef.value?.search();
};

const onRelateAccount = async (relateType: number, accountId: number) => {
    await accountApi.saveRole.request({
        id: accountId,
        roleId: props.role?.id || 0,
        relateType,
    });
    Msg.operateSuccess();
    // 如果是新增账号，则关闭新增账号弹窗
    if (relateType == 1) {
        onCancelAddAccount();
    }
    searchRoleAccount();
};

const onShowAddAccount = () => {
    state.addAccountDialog.visible = true;
};

const onCancelAddAccount = () => {
    state.addAccountDialog.accountId = null;
    state.addAccountDialog.accounts = [];
    state.addAccountDialog.visible = false;
};
</script>
<style lang="scss"></style>
