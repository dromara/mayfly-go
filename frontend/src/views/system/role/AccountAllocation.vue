<template>
    <div>
        <el-dialog
            @open="searchRoleAccount()"
            :title="role == null ? '' : `[${role.name}] 关联的账号`"
            v-model="dialogVisible"
            :destroy-on-close="true"
            width="55%"
        >
            <page-table ref="pageTableRef" :page-api="roleApi.roleAccounts" :search-items="searchItems" v-model:query-form="query" :columns="columns" lazy>
                <template #tableHeader>
                    <el-button v-auth="perms.saveAccountRole" type="primary" icon="plus" @click="showAddAccount()">添加</el-button>
                </template>

                <template #action="{ data }">
                    <el-button link v-if="actionBtns[perms.saveAccountRole]" @click="relateAccount(-1, data.accountId)" icon="delete" type="danger"
                        >移除</el-button
                    >
                </template>
            </page-table>

            <el-dialog width="400px" title="添加账号" :before-close="cancelAddAccount" v-model="addAccountDialog.visible" :destroy-on-close="true">
                <el-form label-width="auto">
                    <AccountSelectFormItem v-model="addAccountDialog.accountId" :focus="true" />
                </el-form>
                <template #footer>
                    <div class="dialog-footer">
                        <el-button @click="cancelAddAccount()">取 消</el-button>
                        <el-button @click="relateAccount(1, addAccountDialog.accountId)" type="primary">确 定</el-button>
                    </div>
                </template>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, Ref } from 'vue';
import { AccountStatusEnum } from '../enums';
import { accountApi, roleApi } from '../api';
import { ElMessage } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { hasPerms } from '@/components/auth/auth';
import { SearchItem } from '@/components/SearchForm';
import AccountSelectFormItem from '../account/components/AccountSelectFormItem.vue';

const props = defineProps({
    role: Object,
});

const perms = {
    saveAccountRole: 'account:saveRoles',
};

const searchItems = [SearchItem.input('name', '姓名'), SearchItem.input('username', '用户名')];
const columns = [
    TableColumn.new('accountName', '姓名'),
    TableColumn.new('username', '用户名'),
    TableColumn.new('accountStatus', '用户状态').typeTag(AccountStatusEnum),
    TableColumn.new('creator', '分配者'),
    TableColumn.new('createTime', '分配时间').isTime(),
];

// 该用户拥有的的操作列按钮权限
const actionBtns = hasPerms([perms.saveAccountRole]);
const actionColumn = TableColumn.new('action', '操作').isSlot().fixedRight().setMinWidth(80).noShowOverflowTooltip().alignCenter();

const pageTableRef: Ref<any> = ref(null);

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
        accounts: [] as any,
        accountId: null as any,
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
    state.query.id = props.role?.id;
    pageTableRef.value.search();
};

const relateAccount = async (relateType: number, accountId: number) => {
    await accountApi.saveRole.request({
        id: accountId,
        roleId: props.role?.id,
        relateType,
    });
    ElMessage.success('操作成功');
    // 如果是新增账号，则关闭新增账号弹窗
    if (relateType == 1) {
        cancelAddAccount();
    }
    searchRoleAccount();
};

const showAddAccount = () => {
    state.addAccountDialog.visible = true;
};

const cancelAddAccount = () => {
    state.addAccountDialog.accountId = null;
    state.addAccountDialog.accounts = [];
    state.addAccountDialog.visible = false;
};
</script>
<style lang="scss"></style>
