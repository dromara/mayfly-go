<template>
    <div>
        <el-dialog
            @open="searchAccountRoles()"
            :title="account == null ? '' : $t('system.account.allocateRoleTitle', { name: account.username })"
            v-model="dialogVisible"
            :before-close="onCancel"
            :destroy-on-close="true"
            width="55%"
        >
            <el-tabs style="min-height: 540px" v-model="state.tabName" @tab-change="onTabChange">
                <el-tab-pane :label="$t('system.account.allocated')" :name="relatedTabName">
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
                            <el-button @click="onShowResources" icon="view" type="primary" link>{{ $t('system.account.menuAndPermission') }}</el-button>
                        </template>

                        <template #action="{ data }">
                            <el-button v-auth="'account:saveRoles'" type="danger" @click="onRelateRole(-1, data.roleId)" icon="delete" link plain>
                                {{ $t('system.account.remove') }}
                            </el-button>
                        </template>
                    </page-table>
                </el-tab-pane>

                <el-tab-pane :label="$t('system.account.undistributed')" :name="unRelatedTabName">
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
                                @click="onRelateRole(1, data.id)"
                                :disabled="data.code?.indexOf('COMMON') == 0 || data.status == RoleStatusEnum.Disable.value"
                                type="success"
                                icon="CirclePlus"
                                link
                                plain
                                >{{ $t('system.account.allocation') }}</el-button
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
                            <SvgIcon :name="getMenuIcon(data)" class="mb-0.5 mr-0.5" />

                            <span v-if="data.type == ResourceTypeEnum.Menu.value">{{ $t(node.label) }}</span>
                            <span v-if="data.type == ResourceTypeEnum.Permission.value" style="color: #67c23a">{{ $t(node.label) }}</span>
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
import { useI18n } from 'vue-i18n';
import { getMenuIcon } from '../resource/index';

const { t } = useI18n();

const props = defineProps({
    account: Object,
});

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

const relatedColumns = [
    TableColumn.new('roleName', 'system.role.roleName'),
    TableColumn.new('code', 'system.role.roleCode'),
    TableColumn.new('status', 'system.account.roleStatus').typeTag(RoleStatusEnum),
    TableColumn.new('creator', 'system.account.assigner'),
    TableColumn.new('createTime', 'system.account.allocateTime').isTime(),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(110).fixedRight().noShowOverflowTooltip().alignCenter(),
];

const unRelatedSearchItems = [SearchItem.input('name', 'system.role.roleName'), SearchItem.input('code', 'system.role.roleCode')];
const unRelatedColumns = [
    TableColumn.new('name', 'system.role.roleName'),
    TableColumn.new('code', 'system.role.roleCode'),
    TableColumn.new('status', 'system.account.roleStatus').typeTag(RoleStatusEnum),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('action', 'common.operation').isSlot().setMinWidth(110).fixedRight().noShowOverflowTooltip().alignCenter(),
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
        pageNum: 1,
        pageSize: 1000,
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

const onRelateRole = async (relateType: number, roleId: number) => {
    await accountApi.saveRole.request({
        id: props.account!.id,
        roleId,
        relateType,
    });
    ElMessage.success(t('common.operateSuccess'));
    if (state.tabName == relatedTabName) {
        searchAccountRoles();
    } else {
        relatedRoleIds.push(roleId);
        searchUnRelateRoles();
    }
};

const onShowResources = async () => {
    let showResourceDialog = state.showResourceDialog;
    showResourceDialog.title = t('system.account.userMenuTitle', { name: props.account?.username });
    showResourceDialog.resources = [];
    showResourceDialog.resources = await accountApi.resources.request({
        id: props.account?.id,
    });
    showResourceDialog.visible = true;
};

/**
 * 取消
 */
const onCancel = () => {
    state.unRelatedQuery.pageNum = 1;
    state.unRelatedQuery.name = null;
    state.unRelatedQuery.code = null;
    state.tabName = relatedTabName;
    dialogVisible.value = false;
    emit('cancel');
};
</script>
