<template>
    <div>
        <page-table ref="pageTableRef" :query="state.queryConfig" v-model:query-form="query" :show-selection="true"
            v-model:selection-data="selectionData" :data="datas" :columns="state.columns" :total="total"
            v-model:page-size="query.pageSize" v-model:page-num="query.pageNum" @pageChange="search()">

            <template #queryRight>
                <el-button v-auth="'account:add'" type="primary" icon="plus" @click="editAccount(true)">添加</el-button>
                <el-button v-auth="'account:add'" :disabled="state.selectionData.length != 1" @click="editAccount(false)"
                    type="primary" icon="edit">编辑</el-button>
                <el-button v-auth="'account:saveRoles'" :disabled="state.selectionData.length != 1" @click="showRoleEdit()"
                    type="success" icon="setting">角色分配</el-button>
                <el-button v-auth="'account:del'" :disabled="state.selectionData.length < 1" @click="deleteAccount()"
                    type="danger" icon="delete">删除</el-button>
            </template>

            <template #status="{ data }">
                <el-tag v-if="data.status == 1" type="success">正常</el-tag>
                <el-tag v-if="data.status == -1" type="danger">禁用</el-tag>
            </template>

            <template #showmore="{ data }">
                <el-link @click.prevent="showRoles(data)" type="success">角色</el-link>

                <el-link class="ml5" @click.prevent="showResources(data)" type="info">菜单&权限</el-link>
            </template>

            <template #action="{ data }">
                <el-button v-auth="'account:changeStatus'" @click="changeStatus(data)" v-if="data.status == 1" type="danger"
                    size="small" plain>禁用</el-button>

                <el-button v-auth="'account:changeStatus'" v-if="data.status == -1" type="success"
                    @click="changeStatus(data)" size="small" plain>启用</el-button>

                <el-button v-auth="'account:add'" :disabled="!data.otpSecret || data.otpSecret == '-'"
                    @click="resetOtpSecret(data)" type="warning" size="small" plain>重置OTP</el-button>
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

        <el-dialog :title="showResourceDialog.title" v-model="showResourceDialog.visible" width="400px">
            <el-tree style="height: 50vh; overflow: auto" :data="showResourceDialog.resources" node-key="id"
                :props="showResourceDialog.defaultProps" :expand-on-click-node="true">
                <template #default="{ node, data }">
                    <span class="custom-tree-node">
                        <span v-if="data.type == enums.ResourceTypeEnum['MENU'].value">{{ node.label }}</span>
                        <span v-if="data.type == enums.ResourceTypeEnum['PERMISSION'].value" style="color: #67c23a">{{
                            node.label
                        }}</span>
                    </span>
                </template>
            </el-tree>
        </el-dialog>

        <role-edit v-model:visible="roleDialog.visible" :account="roleDialog.account" @cancel="cancel()" />
        <account-edit v-model:visible="accountDialog.visible" v-model:account="accountDialog.data"
            @val-change="valChange()" />
    </div>
</template>

<script lang='ts' setup>
import { ref, toRefs, reactive, onMounted } from 'vue';
import RoleEdit from './RoleEdit.vue';
import AccountEdit from './AccountEdit.vue';
import enums from '../enums';
import { accountApi } from '../api';
import { ElMessage, ElMessageBox } from 'element-plus';
import { dateFormat } from '@/common/utils/date';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const pageTableRef: any = ref(null)

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
        pageSize: 10,
    },
    queryConfig: [
        TableQuery.text("username", "用户名"),
    ],
    columns: [
        TableColumn.new("name", "姓名"),
        TableColumn.new("username", "用户名"),
        TableColumn.new("status", "状态").setSlot("status"),
        TableColumn.new("lastLoginTime", "最后登录时间").isTime(),
        TableColumn.new("showmore", "查看更多").setSlot("showmore").setMinWidth(150),
        TableColumn.new("creator", "创建账号"),
        TableColumn.new("createTime", "创建时间").isTime(),
        TableColumn.new("modifier", "更新账号"),
        TableColumn.new("updateTime", "更新时间").isTime(),
        TableColumn.new("action", "操作").setSlot("action").fixedRight().setMinWidth(200),
    ],
    datas: [],
    total: 0,
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

const {
    selectionData,
    query,
    datas,
    total,
    showRoleDialog,
    showResourceDialog,
    roleDialog,
    accountDialog,
} = toRefs(state)

onMounted(() => {
    search();
});

const search = async () => {
    try {
        pageTableRef.value.loading(true);
        let res: any = await accountApi.list.request(state.query);
        state.datas = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false);
    }
};

const showResources = async (row: any) => {
    let showResourceDialog = state.showResourceDialog;
    showResourceDialog.title = '"' + row.username + '" 的菜单&权限';
    showResourceDialog.resources = [];
    showResourceDialog.resources = await accountApi.resources.request({
        id: row.id,
    });
    showResourceDialog.visible = true;
};

const showRoles = async (row: any) => {
    let showRoleDialog = state.showRoleDialog;
    showRoleDialog.title = '"' + row.username + '" 的角色信息';
    showRoleDialog.accountRoles = await accountApi.roles.request({
        id: row.id,
    });
    showRoleDialog.visible = true;
};

const changeStatus = async (row: any) => {
    let id = row.id;
    let status = row.status == -1 ? 1 : -1;
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
    row.otpSecret = "-";
};

const showRoleEdit = () => {
    state.roleDialog.visible = true;
    state.roleDialog.account = state.selectionData[0];
};

const editAccount = (isAdd = false) => {
    if (isAdd) {
        state.accountDialog.data = null;
    } else {
        state.accountDialog.data = state.selectionData[0];
    }
    state.accountDialog.visible = true;
};

const cancel = () => {
    state.roleDialog.visible = false;
    state.roleDialog.account = null;
    search();
};

const valChange = () => {
    state.accountDialog.visible = false;
    search();
};

const deleteAccount = async () => {
    try {
        await ElMessageBox.confirm(`确定删除【${state.selectionData.map((x: any) => x.name).join(", ")}】的账号?`, '提示', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning',
        });
        await accountApi.del.request({ id: state.selectionData.map((x: any) => x.id).join(",") });
        ElMessage.success('删除成功');
        search();
    } catch (err) { }
};
</script>
<style lang="scss"></style>
