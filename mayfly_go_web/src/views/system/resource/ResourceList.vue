<template>
    <div class="menu">
        <div class="toolbar">
            <div>
                <span style="font-size: 14px"><SvgIcon name="info-filled"/>红色字体表示禁用状态</span>
            </div>
            <el-button v-auth="'resource:add'" type="primary" icon="plus" @click="addResource(false)">添加</el-button>
        </div>
        <el-tree
            class="none-select"
            :indent="38"
            node-key="id"
            :props="props"
            :data="data"
            @node-expand="handleNodeExpand"
            @node-collapse="handleNodeCollapse"
            :default-expanded-keys="defaultExpandedKeys"
            :expand-on-click-node="false"
        >
            <template #default="{ data }">
                <span class="custom-tree-node">
                    <span style="font-size: 13px" v-if="data.type === enums.ResourceTypeEnum.MENU.value">
                        <span style="color: #3c8dbc">【</span>
                        {{ data.name }}
                        <span style="color: #3c8dbc">】</span>
                        <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}</el-tag>
                    </span>
                    <span style="font-size: 13px" v-if="data.type === enums.ResourceTypeEnum.PERMISSION.value">
                        <span style="color: #3c8dbc">【</span>
                        <span :style="data.status == 1 ? 'color: #67c23a;' : 'color: #f67c6c;'">{{ data.name }}</span>
                        <span style="color: #3c8dbc">】</span>
                    </span>

                    <el-link @click.prevent="info(data)" style="margin-left: 25px" icon="view" type="info" :underline="false" />

                    <el-link
                        v-auth="'resource:update'"
                        @click.prevent="editResource(data)"
                        class="ml5"
                        type="primary"
                        icon="edit"
                        :underline="false"
                    />

                    <el-link
                        v-auth="'resource:add'"
                        @click.prevent="addResource(data)"
                        v-if="data.type === enums.ResourceTypeEnum.MENU.value"
                        icon="circle-plus"
                        :underline="false"
                        type="success"
                        class="ml5"
                    />

                    <el-link
                        v-auth="'resource:changeStatus'"
                        @click.prevent="changeStatus(data, -1)"
                        v-if="data.status === 1 && data.type === enums.ResourceTypeEnum.PERMISSION.value"
                        icon="circle-close"
                        :underline="false"
                        type="warning"
                        class="ml5"
                    />

                    <el-link
                        v-auth="'resource:changeStatus'"
                        @click.prevent="changeStatus(data, 1)"
                        v-if="data.status === -1 && data.type === enums.ResourceTypeEnum.PERMISSION.value"
                        type="success"
                        icon="circle-check"
                        :underline="false"
                        plain
                        class="ml5"
                    />

                    <el-link
                        v-auth="'resource:delete'"
                        v-if="data.children == null && data.name !== '首页'"
                        @click.prevent="deleteMenu(data)"
                        type="danger"
                        icon="delete"
                        :underline="false"
                        plain
                        class="ml5"
                    />
                </span>
            </template>
        </el-tree>

        <ResourceEdit
            :title="dialogForm.title"
            v-model:visible="dialogForm.visible"
            v-model:data="dialogForm.data"
            :typeDisabled="dialogForm.typeDisabled"
            :departTree="data"
            :type="dialogForm.type"
            @val-change="valChange"
        ></ResourceEdit>

        <el-dialog v-model="infoDialog.visible">
            <el-descriptions title="资源信息" :column="2" border>
                <el-descriptions-item label="类型">
                    <el-tag size="small">{{ enums.ResourceTypeEnum.getLabelByValue(infoDialog.data.type) }}</el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="名称">{{ infoDialog.data.name }}</el-descriptions-item>
                <el-descriptions-item label="code[菜单path]">{{ infoDialog.data.code }}</el-descriptions-item>
                <el-descriptions-item label="序号">{{ infoDialog.data.weight }}</el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue" label="路由名">
                    {{ infoDialog.data.meta.routeName }}
                </el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue" label="组件">
                    {{ infoDialog.data.meta.component }}
                </el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue" label="是否缓存">
                    {{ infoDialog.data.meta.isKeepAlive ? '是' : '否' }}
                </el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue" label="是否隐藏">
                    {{ infoDialog.data.meta.isHide ? '是' : '否' }}
                </el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue" label="tag不可删除">
                    {{ infoDialog.data.meta.isAffix ? '是' : '否' }}
                </el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue" label="是否iframe">
                    {{ infoDialog.data.meta.isIframe ? '是' : '否' }}
                </el-descriptions-item>
                <el-descriptions-item v-if="infoDialog.data.type == menuTypeValue && infoDialog.data.meta.isIframe" label="iframe url">
                    {{ infoDialog.data.meta.link }}
                </el-descriptions-item>

                <el-descriptions-item label="创建者">{{ infoDialog.data.creator }}</el-descriptions-item>
                <el-descriptions-item label="创建时间">{{ $filters.dateFormat(infoDialog.data.createTime) }}</el-descriptions-item>
                <el-descriptions-item label="修改者">{{ infoDialog.data.modifier }}</el-descriptions-item>
                <el-descriptions-item label="更新时间">{{ $filters.dateFormat(infoDialog.data.updateTime) }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import ResourceEdit from './ResourceEdit.vue';
import enums from '../enums';
import { resourceApi } from '../api';

export default defineComponent({
    name: 'ResourceList',
    components: {
        ResourceEdit,
    },
    setup() {
        const state = reactive({
            menuTypeValue: enums.ResourceTypeEnum['MENU'].value,
            permissionTypeValue: enums.ResourceTypeEnum['PERMISSION'].value,
            showBtns: false,
            // 当前鼠标右击的节点数据
            rightClickData: {},
            //弹出框对象
            dialogForm: {
                title: '',
                visible: false,
                data: { pid: 0, type: 1, weight: 1 },
                // 资源类型选择是否选
                typeDisabled: true,
            },
            //资源信息弹出框对象
            infoDialog: {
                title: '',
                visible: false,
                // 资源类型选择是否选
                data: {
                    meta: {},
                },
            },
            data: [],
            props: {
                label: 'name',
                children: 'children',
            },
            // 展开的节点
            defaultExpandedKeys: [] as any[],
        });

        onMounted(() => {
            search();
        });

        const search = async () => {
            let res = await resourceApi.list.request(null);
            state.data = res;
        };

        const deleteMenu = (data: any) => {
            ElMessageBox.confirm(`此操作将删除 [${data.name}], 是否继续?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            }).then(() => {
                resourceApi.del
                    .request({
                        id: data.id,
                    })
                    .then((res) => {
                        console.log(res);
                        ElMessage.success('删除成功！');
                        search();
                    });
            });
        };

        const addResource = (data: any) => {
            let dialog = state.dialogForm;
            dialog.data = { pid: 0, type: 1, weight: 1 };
            // 添加顶级菜单情况
            if (!data) {
                dialog.typeDisabled = true;
                dialog.data.type = state.menuTypeValue;
                dialog.title = '添加顶级菜单';
                dialog.visible = true;
                return;
            }
            // 添加子菜单，把当前菜单id作为新增菜单pid
            dialog.data.pid = data.id;

            dialog.title = '添加“' + data.name + '”的子资源 ';
            if (data.children === null || data.children.length === 0) {
                // 如果子节点不存在，则资源类型可选择
                dialog.typeDisabled = false;
            } else {
                dialog.typeDisabled = true;
                let hasPermission = false;
                for (let c of data.children) {
                    if (c.type === state.permissionTypeValue) {
                        hasPermission = true;
                        break;
                    }
                }
                // 如果子节点中存在权限资源，则只能新增权限资源，否则只能新增菜单资源
                if (hasPermission) {
                    dialog.data.type = state.permissionTypeValue;
                } else {
                    dialog.data.type = state.menuTypeValue;
                }
                dialog.data.weight = data.children.length + 1;
            }
            dialog.visible = true;
        };

        const editResource = async (data: any) => {
            state.dialogForm.visible = true;
            const res = await resourceApi.detail.request({
                id: data.id,
            });
            if (res.meta) {
                res.meta = JSON.parse(res.meta);
            }

            state.dialogForm.data = res;
            state.dialogForm.typeDisabled = true;
            state.dialogForm.title = '修改“' + data.name + '”菜单';
        };

        const valChange = () => {
            search();
            state.dialogForm.visible = false;
        };

        const changeStatus = async (data: any, status: any) => {
            await resourceApi.changeStatus.request({
                id: data.id,
                status: status,
            });
            data.status = status;
            ElMessage.success((status === 1 ? '启用' : '禁用') + '成功！');
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

        const removeDeafultExpandId = (id: any) => {
            let index = state.defaultExpandedKeys.indexOf(id);
            if (index > -1) {
                state.defaultExpandedKeys.splice(index, 1);
            }
        };

        const info = async (data: any) => {
            let info = await resourceApi.detail.request({ id: data.id });
            state.infoDialog.data = info;
            if (info.meta && info.meta != '') {
                state.infoDialog.data.meta = JSON.parse(info.meta);
            }
            state.infoDialog.visible = true;
        };

        return {
            ...toRefs(state),
            enums,
            deleteMenu,
            addResource,
            editResource,
            valChange,
            changeStatus,
            handleNodeExpand,
            handleNodeCollapse,
            info,
        };
    },
});
</script>
<style lang="scss">
.menu {
    .el-tree-node__content {
        height: 40px;
        line-height: 40px;
    }
}

.none-select {
    moz-user-select: -moz-none;
    -moz-user-select: none;
    -o-user-select: none;
    -khtml-user-select: none;
    -webkit-user-select: none;
    -ms-user-select: none;
    user-select: none;
}
</style>
