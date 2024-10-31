<template>
    <div>
        <el-dialog
            @open="searchAccountRoles()"
            :title="account == null ? '' : '分配“' + account.username + '”的角色'"
            v-model="dialogVisible"
            :before-close="cancel"
            :destroy-on-close="true"
            width="55%"
        >
            <el-tabs style="min-height: 540px" v-model="state.tabName" @tab-change="onTabChange">
                <el-tab-pane label="已分配" :name="relatedTabName">
                    <page-table
                        ref="relatePageTableRef"
                        :pageable="false"
                        :page-api="accountApi.roles"
                        v-model:query-form="releateQuery"
                        :columns="relatedColumns"
                        :tool-button="false"
                        lazy
                    >
                        <template #tableHeader>
                            <el-button @click="showResources" icon="view" type="primary" link>用户菜单&权限</el-button>
                        </template>

                        <template #action="{ data }">
                            <el-button v-auth="'account:saveRoles'" type="danger" @click="relateRole(-1, data.roleId)" icon="delete" link plain>移除</el-button>
                        </template>
                    </page-table>
                </el-tab-pane>

                <el-tab-pane label="未分配" :name="unRelatedTabName">
                    <page-table
                        ref="unRelatePageTableRef"
                        :page-api="roleApi.list"
                        :search-items="unRelatedSearchItems"
                        v-model:query-form="unRelatedQuery"
                        :columns="unRelatedColumns"
                        :search-col="3"
                        lazy
                    >
                        <template #action="{ data }">
                            <el-button
                                v-auth="'account:saveRoles'"
                                @click="relateRole(1, data.id)"
                                :disabled="data.code?.indexOf('COMMON') == 0 || data.status == RoleStatusEnum.Disable.value"
                                type="success"
                                icon="CirclePlus"
                                link
                                plain
                                >分配</el-button
                            >
                        </template>
                    </page-table>
                </el-tab-pane>
            </el-tabs>

            <el-dialog :title="showResourceDialog.title" v-model="showResourceDialog.visible" width="400px">
                <el-tree
                    style="height: 50vh; overflow: auto"
                    :data="showResourceDialog.resources"
                    node-key="id"
                    :props="showResourceDialog.defaultProps"
                    :expand-on-click-node="true"
                >
                    <template #default="{ node, data }">
                        <span class="custom-tree-node">
                            <span v-if="data.type == ResourceTypeEnum.Menu.value">{{ node.label }}</span>
                            <span v-if="data.type == ResourceTypeEnum.Permission.value" style="color: #67c23a">{{ node.label }}</span>
                        </span>
                    </template>
                </el-tree>
            </el-dialog>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, reactive, ref } from 'vue';
import { roleApi, accountApi } from '../api';
import { ElMessage } from 'element-plus';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { SearchItem } from '@/components/SearchForm';
import { ResourceTypeEnum, RoleStatusEnum } from '../enums';

const props = defineProps({
    account: Object,
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const relatedColumns = [
    TableColumn.new('roleName', '角色名'),
    TableColumn.new('code', '角色code'),
    TableColumn.new('status', '角色状态').typeTag(RoleStatusEnum),
    TableColumn.new('creator', '分配者'),
    TableColumn.new('createTime', '分配时间').isTime(),
    TableColumn.new('action', '操作').isSlot().setMinWidth(80).fixedRight().alignCenter(),
];

const unRelatedSearchItems = [SearchItem.input('name', '角色名'), SearchItem.input('code', '角色code')];
const unRelatedColumns = [
    TableColumn.new('name', '角色名'),
    TableColumn.new('code', '角色code'),
    TableColumn.new('status', '角色状态').typeTag(RoleStatusEnum),
    TableColumn.new('remark', '备注'),
    TableColumn.new('action', '操作').isSlot().setMinWidth(80).fixedRight().alignCenter(),
];

const relatePageTableRef: any = ref(null);
const unRelatePageTableRef: any = ref(null);

// 已分配与未分配tab名
const relatedTabName = 'related';
const unRelatedTabName = 'unRelated';

const state = reactive({
    tabName: relatedTabName,
    // 该账号拥有的角色id
    unRelatedQuery: {
        pageNum: 1,
        pageSize: 7,
        name: null,
        code: null,
        notIds: '',
    },
    releateQuery: {
        id: 0, //账号id
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
});

let relatedRoleIds: Number[] = []; // 用户已关联的角色ids

const { releateQuery, unRelatedQuery, showResourceDialog } = toRefs(state);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const searchAccountRoles = async () => {
    state.releateQuery.id = props.account?.id;
    await relatePageTableRef.value?.search();
    relatedRoleIds = relatePageTableRef.value.getData()?.map((x: any) => x.roleId);
};

const searchUnRelateRoles = () => {
    state.unRelatedQuery.notIds = relatedRoleIds.join(',');
    unRelatePageTableRef.value?.search();
};

const onTabChange = () => {
    if (state.tabName == unRelatedTabName) {
        searchUnRelateRoles();
        return;
    }

    searchAccountRoles();
};

const relateRole = async (relateType: number, roleId: number) => {
    await accountApi.saveRole.request({
        id: props.account!.id,
        roleId,
        relateType,
    });
    ElMessage.success('操作成功');
    if (state.tabName == relatedTabName) {
        searchAccountRoles();
    } else {
        relatedRoleIds.push(roleId);
        searchUnRelateRoles();
    }
};

const showResources = async () => {
    let showResourceDialog = state.showResourceDialog;
    showResourceDialog.title = '"' + props.account?.username + '" 的菜单&权限';
    showResourceDialog.resources = [];
    showResourceDialog.resources = await accountApi.resources.request({
        id: props.account?.id,
    });
    showResourceDialog.visible = true;
};

/**
 * 取消
 */
const cancel = () => {
    state.unRelatedQuery.pageNum = 1;
    state.unRelatedQuery.name = null;
    state.unRelatedQuery.code = null;
    state.tabName = relatedTabName;
    dialogVisible.value = false;
    emit('cancel');
};
</script>
