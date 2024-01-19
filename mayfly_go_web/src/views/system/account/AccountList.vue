<template>
    <div>
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
                <el-button v-auth="perms.addAccount" type="primary" icon="plus" @click="editAccount(false)">添加</el-button>
                <el-button v-auth="perms.delAccount" :disabled="state.selectionData.length < 1" @click="deleteAccount()" type="danger" icon="delete"
                    >删除</el-button
                >
            </template>

            <template #action="{ data }">
                <el-button link v-if="actionBtns[perms.addAccount]" @click="editAccount(data)" type="primary">编辑</el-button>

                <el-button link v-if="actionBtns[perms.saveAccountRole]" @click="showRoleEdit(data)" type="success">角色分配</el-button>

                <el-button link v-if="actionBtns[perms.changeAccountStatus] && data.status == 1" @click="changeStatus(data)" type="danger">禁用</el-button>

                <el-button link v-if="actionBtns[perms.changeAccountStatus] && data.status == -1" type="success" @click="changeStatus(data)">启用</el-button>

                <el-button
                    link
                    v-if="actionBtns[perms.addAccount]"
                    :disabled="!data.otpSecret || data.otpSecret == '-'"
                    @click="resetOtpSecret(data)"
                    type="warning"
                    >重置OTP
                </el-button>
            </template>
        </page-table>

        <el-dialog width="500px" :title="showRoleDialog.title" v-model="showRoleDialog.visible">
            <el-table border :data="showRoleDialog.accountRoles">
                <el-table-column property="name" label="角色名" width="125"></el-table-column>
                <el-table-column property="creator" label="分配账号" width="125"></el-table-column>
                <el-table-column property="createTime" label="分配时间">
                    <template #default="scope">
                        {{ dateFormat(scope.row.createTime) }}
                    </template>
                </el-table-column>
            </el-table>
        </el-dialog>

        <role-allocation v-model:visible="roleDialog.visible" :account="roleDialog.account" @cancel="cancel()" />
        <account-edit v-model:visible="accountDialog.visible" v-model:account="accountDialog.data" @val-change="valChange()" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import RoleAllocation from './RoleAllocation.vue';
import AccountEdit from './AccountEdit.vue';
import { AccountStatusEnum } from '../enums';
import { accountApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dateFormat } from '@/common/utils/date';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';

const perms = {
    addAccount: 'account:add',
    delAccount: 'account:del',
    saveAccountRole: 'account:saveRoles',
    changeAccountStatus: 'account:changeStatus',
};

const searchItems = [SearchItem.input('username', '用户名')];
const columns = [
    TableColumn.new('name', '姓名'),
    TableColumn.new('username', '用户名'),
    TableColumn.new('status', '状态').typeTag(AccountStatusEnum),
    TableColumn.new('lastLoginTime', '最后登录时间').isTime(),
    TableColumn.new('creator', '创建账号'),
    TableColumn.new('createTime', '创建时间').isTime(),
    TableColumn.new('modifier', '更新账号'),
    TableColumn.new('updateTime', '更新时间').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.addAccount, perms.saveAccountRole, perms.changeAccountStatus]);
const actionColumn = TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(260).noShowOverflowTooltip().alignCenter();

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

const changeStatus = async (row: any) => {
    let id = row.id;
    let status = row.status == AccountStatusEnum.Disable.value ? AccountStatusEnum.Enable.value : AccountStatusEnum.Disable.value;
    await accountApi.changeStatus.request({
        id,
        status,
    });
    ElMessage.success('操作成功');
    search();
};

const resetOtpSecret = async (row: any) => {
    let id = row.id;
    await accountApi.resetOtpSecret.request({
        id,
    });
    ElMessage.success('操作成功');
    row.otpSecret = '-';
};

const editAccount = (data: any) => {
    if (!data) {
        state.accountDialog.data = null;
    } else {
        state.accountDialog.data = data;
    }
    state.accountDialog.visible = true;
};

const showRoleEdit = (data: any) => {
    state.roleDialog.visible = true;
    state.roleDialog.account = data;
};

const cancel = () => {
    state.roleDialog.visible = false;
    state.roleDialog.account = null;
};

const valChange = () => {
    state.accountDialog.visible = false;
    search();
};

const deleteAccount = async () => {
    try {
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(', ')}】的账号?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await accountApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(',') });
        ElMessage.success('删除成功');
        search();
    } catch (err) {
        //
    }
};
</script>
<style lang="scss"></style>
