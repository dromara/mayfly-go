<template>
    <div class="card !p-2 system-resource-list h-full flex">
        <el-splitter>
            <el-splitter-panel size="30%" max="35%" min="25%" class="flex flex-col flex-1">
                <div class="card !p-1 mr-1 flex justify-between">
                    <div class="mb-1">
                        <el-input v-model="filterResource" clearable :placeholder="$t('system.menu.filterPlaceholder')" class="mr-2 !w-[200px]" />
                        <el-button v-auth="perms.addResource" type="primary" icon="plus" @click="onAddResource(false)"></el-button>
                    </div>

                    <div>
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
                                <SvgIcon :name="getMenuIcon(data)" class="!mb-0.5" />

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
                                    <SvgIcon :name="currentResource.meta.icon" />
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.routerName')">
                                    {{ currentResource.meta.routeName }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.isCache')">
                                    {{ currentResource.meta.isKeepAlive ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.isHide')">
                                    {{ currentResource.meta.isHide ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.tagIsDelete')">
                                    {{ currentResource.meta.isAffix ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item v-if="currentResource.type == menuTypeValue" :label="$t('system.menu.externalLink')">
                                    {{ currentResource.meta.linkType ? $t('system.menu.yes') : $t('system.menu.no') }}
                                </el-descriptions-item>
                                <el-descriptions-item
                                    v-if="currentResource.type == menuTypeValue && currentResource.meta.linkType > 0"
                                    :label="$t('system.menu.externalLink')"
                                >
                                    {{ currentResource.meta.link }}
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
            :departTree="data"
            :type="dialogForm.type"
            @val-change="onValChange"
        />

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, onMounted, watch } from 'vue';
import { ElMessage } from 'element-plus';
import ResourceEdit from './ResourceEdit.vue';
import { ResourceTypeEnum, RoleStatusEnum } from '../enums';
import { resourceApi } from '../api';
import { formatDate } from '@/common/utils/format';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { isPrefixSubsequence } from '@/common/utils/string';
import { useI18n } from 'vue-i18n';
import { useI18nDeleteConfirm, useI18nDeleteSuccessMsg } from '@/hooks/useI18n';
import { getMenuIcon } from './index';

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
    .withHideFunc((data: any) => data.type !== menuTypeValue)
    .withOnClick((data: any) => onAddResource(data));

const contextmenuEdit = new ContextmenuItem('edit', 'common.edit')
    .withIcon('edit')
    .withPermission(perms.updateResource)
    .withOnClick((data: any) => onEditResource(data));

const contextmenuEnable = new ContextmenuItem('enable', 'system.menu.enable')
    .withIcon('circle-check')
    .withPermission(perms.updateResource)
    .withHideFunc((data: any) => data.status === 1)
    .withOnClick((data: any) => onChangeStatus(data, 1));

const contextmenuDisable = new ContextmenuItem('disable', 'system.menu.disable')
    .withIcon('circle-close')
    .withPermission(perms.updateResource)
    .withHideFunc((data: any) => data.status === -1)
    .withOnClick((data: any) => onChangeStatus(data, -1));

const contextmenuDel = new ContextmenuItem('delete', 'common.delete')
    .withIcon('delete')
    .withPermission(perms.delResource)
    .withOnClick((data: any) => onDeleteMenu(data));

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
    data: [],
    rolesLoading: false,
    roles: [], // 资源关联的角色列表
    // 展开的节点
    defaultExpandedKeys: [] as any[],
    activeTabName: ResourceDetail,
    currentResource: null as any,
});

const { currentResource, dialogForm, data, defaultExpandedKeys } = toRefs(state);

onMounted(() => {
    search();
});

watch(filterResource, (val) => {
    resourceTreeRef.value!.filter(val);
});

const filterNode = (value: string, data: any) => {
    return !value || isPrefixSubsequence(value, t(data.name));
};

const search = async () => {
    let res = await resourceApi.list.request(null);
    state.data = res;
};

// 树节点右击事件
const nodeContextmenu = (event: any, data: any) => {
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value.openContextmenu(data);
};

const onTreeNodeClick = async (data: any) => {
    state.activeTabName = ResourceDetail;
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();

    let info = await resourceApi.detail.request({ id: data.id });
    state.currentResource = info;
    if (info.meta && info.meta != '') {
        state.currentResource.meta = JSON.parse(info.meta);
    }
};

const onTabClick = async (activeTab: any) => {
    if (activeTab.paneName === ResourceRoles) {
        try {
            state.rolesLoading = true;
            state.roles = await resourceApi.roles.request({ id: state.currentResource.id });
        } finally {
            state.rolesLoading = false;
        }
    }
};

const onDeleteMenu = async (data: any) => {
    await useI18nDeleteConfirm(data.name);
    await resourceApi.del.request({
        id: data.id,
    });

    useI18nDeleteSuccessMsg();
    search();
};

const onAddResource = (data: any) => {
    let dialog = state.dialogForm;
    dialog.data = { pid: 0, type: 1 };
    // 添加顶级菜单情况
    if (!data) {
        dialog.typeDisabled = true;
        dialog.data.type = menuTypeValue;
        dialog.title = t('system.menu.addTopMenu');
        dialog.visible = true;
        return;
    }
    // 添加子菜单，把当前菜单id作为新增菜单pid
    dialog.data.pid = data.id;

    dialog.title = t('system.menu.addChildrenMenuTitle', { parentName: t(data.name) });
    if (data.children === null || data.children.length === 0) {
        // 如果子节点不存在，则资源类型可选择
        dialog.typeDisabled = false;
    } else {
        dialog.typeDisabled = true;
        let hasPermission = false;
        for (let c of data.children) {
            if (c.type === permissionTypeValue) {
                hasPermission = true;
                break;
            }
        }
        // 如果子节点中存在权限资源，则只能新增权限资源，否则只能新增菜单资源
        if (hasPermission) {
            dialog.data.type = permissionTypeValue;
        } else {
            dialog.data.type = menuTypeValue;
        }
    }
    dialog.visible = true;
};

const onEditResource = async (data: any) => {
    const res = await resourceApi.detail.request({
        id: data.id,
    });
    if (res.meta) {
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

const onChangeStatus = async (data: any, status: any) => {
    await resourceApi.changeStatus.request({
        id: data.id,
        status: status,
    });
    search();
    ElMessage.success((status === 1 ? t('system.menu.enable') : t('system.menu.disable')) + ' ' + t('system.menu.success'));
};

// 节点被展开时触发的事件
const handleNodeExpand = (data: any, node: any) => {
    const id: any = node.data.id;
    if (!state.defaultExpandedKeys.includes(id)) {
        state.defaultExpandedKeys.push(id);
    }
};

// 关闭节点
const handleNodeCollapse = (data: any, node: any) => {
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

const allowDrop = (draggingNode: any, dropNode: any, type: any) => {
    // 如果是插入至目标节点
    if (type === 'inner') {
        // 只有目标节点下没有子节点才允许移动
        if (!dropNode.data.children || dropNode.data.children == 0) {
            // 只有权限节点可移动至菜单节点下 或者移动菜单
            return (
                (draggingNode.data.type == permissionTypeValue && dropNode.data.type == menuTypeValue) ||
                (draggingNode.data.type == menuTypeValue && dropNode.data.type == menuTypeValue)
            );
        }
        return false;
    }
    return draggingNode.data.type === dropNode.data.type;
};

const handleDrop = async (draggingNode: any, dropNode: any, dropType: any) => {
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

const removeDeafultExpandId = (id: any) => {
    let index = state.defaultExpandedKeys.indexOf(id);
    if (index > -1) {
        state.defaultExpandedKeys.splice(index, 1);
    }
};
</script>
<style lang="scss">
.system-resource-list {
    .el-tree-node__content {
        height: 40px;
        line-height: 40px;
    }
}
</style>
