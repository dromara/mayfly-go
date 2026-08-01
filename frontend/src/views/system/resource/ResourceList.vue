<template>
    <div class="card p-2! system-resource-list h-full flex">
        <el-splitter>
            <el-splitter-panel size="30%" max="35%" min="25%" class="flex flex-col flex-1">
                <div class="card p-1! mr-1 flex flex-row items-center justify-between overflow-hidden">
                    <el-input v-model="filterResource" clearable :placeholder="$t('system.menu.filterPlaceholder')" class="mr-2" />
                    <el-button v-auth="perms.addResource" type="primary" icon="plus" @click="onAddResource(false)"></el-button>

                    <div class="ml-1">
                        <el-tooltip placement="top">
                            <template #content> {{ $t('system.menu.opTips') }} </template>
                            <SvgIcon name="question-filled" />
                        </el-tooltip>
                    </div>
                </div>
                <el-scrollbar>
                    <el-tree
                        class="inline-block min-w-full"
                        ref="resourceTreeRef"
                        :indent="24"
                        node-key="id"
                        :props="props"
                        :data="data"
                        highlight-current
                        @node-expand="handleNodeExpand"
                        @node-collapse="handleNodeCollapse"
                        @node-contextmenu="nodeContextmenu"
                        @node-click="onTreeNodeClick"
                        :default-expanded-keys="defaultExpandedKeys"
                        :expand-on-click-node="false"
                        draggable
                        :allow-drop="allowDrop"
                        @node-drop="handleDrop"
                        :filter-node-method="filterNode"
                    >
                        <template #default="{ data }">
                            <span class="custom-tree-node">
                                <SvgIcon :name="getMenuIcon(data)" class="mb-0.5!" />

                                <span style="font-size: 13px" v-if="data.type === menuTypeValue">
                                    <span style="color: #3c8dbc">【</span>
                                    <span v-if="data.status == 1">{{ $t(data.name) }}</span>
                                    <span v-if="data.status == -1" style="color: #e6a23c">{{ $t(data.name) }}</span>
                                    <span style="color: #3c8dbc">】</span>
                                    <el-tag v-if="data.children !== null" size="small">
                                        {{ data.children.length }}
                                    </el-tag>
                                </span>

                                <span style="font-size: 13px" v-if="data.type === permissionTypeValue">
                                    <span style="color: #3c8dbc">【</span>
                                    <span :style="data.status == 1 ? 'color: #67c23a;' : 'color: #f67c6c;'">
                                        {{ $t(data.name) }}
                                    </span>
                                    <span style="color: #3c8dbc">】</span>
                                </span>
                            </span>
                        </template>
                    </el-tree>
                </el-scrollbar>
            </el-splitter-panel>

            <el-splitter-panel>
                <div class="ml-2">
                    <el-tabs v-model="state.activeTabName" @tab-click="onTabClick" v-if="currentResource">
                        <el-tab-pane :label="$t('common.detail')" :name="ResourceDetail">
                            <el-descriptions :title="$t('system.menu.info')" :column="2" border>
                                <el-descriptions-item :label="$t('common.type')">
                                    <enum-tag :enums="ResourceTypeEnum" :value="currentResource?.type" />
                                </el-descriptions-item>
                                <el-descriptions-item :label="$t('common.name')">{{ currentResource.name }}</el-descriptions-item>
                                <el-descriptions-item :label="`code[${$t('system.menu.menu')} path]`">{{ currentResource.code }}</el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.icon')">
                                    <SvgIcon :name="currentMeta.icon" />
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.routerName')">
                                    {{ currentMeta.routeName }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.isCache')">
                                    {{ currentMeta.isKeepAlive ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.isHide')">
                                    {{ currentMeta.isHide ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.tagIsDelete')">
                                    {{ currentMeta.isAffix ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.externalLink')">
                                    {{ currentMeta.linkType ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item
                                    v-if="currentResource.type == menuTypeValue && currentMeta.linkType > 0"
                                    :label="$t('system.menu.externalLink')"
                                >
                                    {{ currentMeta.link }}
                                </el-descriptions-item>

                                <el-descriptions-item :label="$t('common.creator')">{{ currentResource.creator }}</el-descriptions-item>
                                <el-descriptions-item :label="$t('common.createTime')">{{ formatDate(currentResource.createTime) }} </el-descriptions-item>
                                <el-descriptions-item :label="$t('common.modifier')">{{ currentResource.modifier }}</el-descriptions-item>
                                <el-descriptions-item :label="$t('common.updateTime')">{{ formatDate(currentResource.updateTime) }} </el-descriptions-item>
                            </el-descriptions>
                        </el-tab-pane>

                        <el-tab-pane :label="$t('system.menu.assignedRole')" :name="ResourceRoles">
                            <el-table :loading="state.rolesLoading" :data="state.roles" max-height="calc(100vh - 200px)">
                                <el-table-column prop="roleCode" :label="$t('system.role.roleCode')"></el-table-column>
                                <el-table-column prop="roleName" :label="$t('system.role.roleName')"></el-table-column>
                                <el-table-column prop="roleStatus" :label="$t('system.account.roleStatus')">
                                    <template #default="scope">
                                        <enum-tag :enums="RoleStatusEnum" :value="scope.row.roleStatus"></enum-tag>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="assigner" :label="$t('system.role.assigner')"></el-table-column>
                                <el-table-column prop="allocateTime" :label="$t('system.role.allocateTime')" min-width="150">
                                    <template #default="scope">
                                        {{ formatDate(scope.row.allocateTime) }}
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </el-splitter-panel>
        </el-splitter>

        <ResourceEdit
            :title="dialogForm.title"
            v-model:visible="dialogForm.visible"
            v-model:data="dialogForm.data"
            :typeDisabled="dialogForm.typeDisabled"
            :type="dialogForm.type"
            @val-change="onValChange"
        />

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { formatDate } from '@/common/utils/format';
import { isPrefixSubsequence } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { defineAsyncComponent, computed, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { resourceApi } from '../api';
import { ResourceTypeEnum, RoleStatusEnum } from '../enums';
import type { ResourceMeta, SysResource, SysRole } from '../types';
import { getMenuIcon } from './index';

const ResourceEdit = defineAsyncComponent(() => import('./ResourceEdit.vue'));

const { t } = useI18n();

const menuTypeValue = ResourceTypeEnum.Menu.value;
const permissionTypeValue = ResourceTypeEnum.Permission.value;

const perms = {
    addResource: 'resource:add',
    delResource: 'resource:delete',
    updateResource: 'resource:update',
    changeStatus: 'resource:changeStatus',
};

const props = {
    label: 'name',
    children: 'children',
};

const contextmenuRef = ref();
const filterResource = ref();
const resourceTreeRef = ref();

const ResourceDetail = 'resourceDetail';
const ResourceRoles = 'resourceRoles';

const contextmenuAdd = new ContextmenuItem('add', 'system.menu.addSubResource')
    .withIcon('circle-plus')
    .withPermission(perms.addResource)
    .withOnClick((data: unknown) => onAddResource(data as SysResource));

const contextmenuEdit = new ContextmenuItem('edit', 'common.edit')
    .withIcon('edit')
    .withPermission(perms.updateResource)
    .withOnClick((data: unknown) => onEditResource(data as SysResource));

const contextmenuEnable = new ContextmenuItem('enable', 'system.menu.enable')
    .withIcon('circle-check')
    .withPermission(perms.updateResource)
    .withHideFunc((data: unknown) => (data as SysResource).status === 1)
    .withOnClick((data: unknown) => onChangeStatus(data as SysResource, 1));

const contextmenuDisable = new ContextmenuItem('disable', 'system.menu.disable')
    .withIcon('circle-close')
    .withPermission(perms.updateResource)
    .withHideFunc((data: unknown) => (data as SysResource).status === -1)
    .withOnClick((data: unknown) => onChangeStatus(data as SysResource, -1));

const contextmenuDel = new ContextmenuItem('delete', 'common.delete')
    .withIcon('delete')
    .withPermission(perms.delResource)
    .withOnClick((data: unknown) => onDeleteMenu(data as SysResource));

const state = reactive({
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [contextmenuAdd, contextmenuEdit, contextmenuEnable, contextmenuDisable, contextmenuDel],
    },
    //弹出框对象
    dialogForm: {
        type: null,
        title: '',
        visible: false,
        data: { pid: 0, type: 1 },
        // 资源类型选择是否选
        typeDisabled: true,
    },
    data: [] as SysResource[],
    rolesLoading: false,
    roles: [] as SysRole[], // 资源关联的角色列表
    // 展开的节点
    defaultExpandedKeys: [] as number[],
    activeTabName: ResourceDetail,
    currentResource: null as SysResource | null,
});

const { currentResource, dialogForm, data, defaultExpandedKeys } = toRefs(state);

const emptyMeta: ResourceMeta = {
    routeName: '',
    icon: '',
    isKeepAlive: false,
    isHide: false,
    isAffix: false,
    linkType: 0,
};

/** 当前选中资源的 meta（已解析为对象，未解析时返回空 meta） */
const currentMeta = computed<ResourceMeta>(() => {
    const meta = state.currentResource?.meta;
    return typeof meta === 'object' && meta !== null ? meta : emptyMeta;
});

onMounted(() => {
    search();
});

watch(filterResource, (val) => {
    resourceTreeRef.value!.filter(val);
});

const filterNode = (value: string, data: SysResource) => {
    return !value || isPrefixSubsequence(value, t(data.name));
};

const search = async () => {
    let res = await resourceApi.list.request(null);
    state.data = res;
};

// 树节点右击事件
const nodeContextmenu = (event: MouseEvent, data: SysResource) => {
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value?.openContextmenu(data);
};

const onTreeNodeClick = async (data: SysResource) => {
    state.activeTabName = ResourceDetail;
    // 关闭可能存在的右击菜单
    contextmenuRef.value?.closeContextmenu();

    const info = await resourceApi.detail.request({ id: data.id });
    if (typeof info.meta === 'string' && info.meta !== '') {
        info.meta = JSON.parse(info.meta);
    }
    state.currentResource = info;
};

const onTabClick = async (activeTab: { paneName?: string | number }) => {
    if (activeTab.paneName === ResourceRoles && state.currentResource) {
        try {
            state.rolesLoading = true;
            state.roles = await resourceApi.roles.request({ id: state.currentResource.id });
        } finally {
            state.rolesLoading = false;
        }
    }
};

const onDeleteMenu = async (data: SysResource) => {
    await useI18nDeleteConfirm(data.name);
    await resourceApi.del.request({
        id: data.id,
    });

    Msg.deleteSuccess();
    search();
};

const onAddResource = (data: SysResource | false) => {
    let dialog = state.dialogForm;
    dialog.data = { pid: 0, type: 1 };
    dialog.typeDisabled = false;

    // 添加顶级菜单情况
    if (!data) {
        dialog.data.type = menuTypeValue;
        dialog.title = t('system.menu.addTopMenu');
        dialog.visible = true;
        return;
    }

    // 父节点为权限类型，子节点也只允许添加权限类型
    if (data.type === permissionTypeValue) {
        dialog.typeDisabled = true;
        dialog.data.type = permissionTypeValue;
    }

    // 添加子菜单，把当前菜单id作为新增菜单pid
    dialog.data.pid = data.id;
    dialog.title = t('system.menu.addChildrenMenuTitle', { parentName: t(data.name) });
    dialog.visible = true;
};

const onEditResource = async (data: SysResource) => {
    const res = await resourceApi.detail.request({
        id: data.id,
    });
    if (typeof res.meta === 'string' && res.meta !== '') {
        res.meta = JSON.parse(res.meta);
    }

    state.dialogForm.data = res;
    state.dialogForm.typeDisabled = true;
    state.dialogForm.title = t('system.menu.updateMenu', { name: t(data.name) });
    state.dialogForm.visible = true;
};

const onValChange = () => {
    search();
    state.dialogForm.visible = false;
};

const onChangeStatus = async (data: SysResource, status: number) => {
    await resourceApi.changeStatus.request({
        id: data.id,
        status: status,
    });
    search();
    Msg.success((status === 1 ? t('system.menu.enable') : t('system.menu.disable')) + ' ' + t('system.menu.success'));
};

// 节点被展开时触发的事件
const handleNodeExpand = (data: SysResource, node: { data: SysResource }) => {
    const id = node.data.id;
    if (!state.defaultExpandedKeys.includes(id)) {
        state.defaultExpandedKeys.push(id);
    }
};

interface ResourceTreeNode {
    data: SysResource;
    childNodes: ResourceTreeNode[];
    expanded: boolean;
}

// 关闭节点
const handleNodeCollapse = (data: SysResource, node: ResourceTreeNode) => {
    removeDeafultExpandId(node.data.id);

    let childNodes = node.childNodes;
    for (let cn of childNodes) {
        if (cn.data.type == 2) {
            return;
        }
        if (cn.expanded) {
            removeDeafultExpandId(cn.data.id);
        }
        // 递归删除展开的子节点节点id
        handleNodeCollapse(data, cn);
    }
};

const allowDrop = (draggingNode: ResourceTreeNode, dropNode: ResourceTreeNode, type: string) => {
    // 如果是插入至目标节点
    if (type === 'inner') {
        // 只有目标节点下没有子节点才允许移动
        if (!dropNode.data.children || dropNode.data.children?.length == 0) {
            // 只有权限节点可移动至菜单节点下 或者移动菜单
            return (
                (draggingNode.data.type == permissionTypeValue && dropNode.data.type == menuTypeValue) ||
                (draggingNode.data.type == permissionTypeValue && dropNode.data.type == permissionTypeValue) ||
                (draggingNode.data.type == menuTypeValue && dropNode.data.type == menuTypeValue)
            );
        }
        return false;
    }
    return draggingNode.data.type === dropNode.data.type;
};

const handleDrop = async (draggingNode: ResourceTreeNode, dropNode: ResourceTreeNode, dropType: string) => {
    const draggingData = draggingNode.data;
    const dropData = dropNode.data;
    if (draggingData.pid !== dropData.pid) {
        draggingData.pid = dropData.pid;
    }
    if (dropType === 'inner') {
        draggingData.weight = 1;
        draggingData.pid = dropData.id;
    }
    if (dropType === 'before') {
        draggingData.weight = dropData.weight - 1;
    }
    if (dropType === 'after') {
        draggingData.weight = dropData.weight + 1;
    }

    await resourceApi.sort.request([
        {
            id: draggingData.id,
            name: draggingData.name,
            pid: draggingData.pid,
            weight: draggingData.weight,
        },
    ]);
};

const removeDeafultExpandId = (id: number) => {
    let index = state.defaultExpandedKeys.indexOf(id);
    if (index > -1) {
        state.defaultExpandedKeys.splice(index, 1);
    }
};
</script>
<style lang="scss">
.system-resource-list {
    .el-tree-node__content {
        height: 35px;
        line-height: 35px;
    }
}
</style>
