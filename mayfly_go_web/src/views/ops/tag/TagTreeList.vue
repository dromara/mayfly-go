<template>
    <div class="menu">
        <div class="toolbar">
            <el-input v-model="filterTag" placeholder="输入关键字过滤(右击进行操作)" style="width: 220px; margin-right: 10px" />
            <el-button v-auth="'tag:save'" type="primary" icon="plus" @click="showSaveTagDialog(null)">添加</el-button>
            <div style="float: right">
                <el-tooltip placement="top">
                    <template #content>
                        1. 用于将资产进行归类
                        <br />2. 可在团队管理中进行分配，用于资源隔离 <br />3. 拥有父标签的团队成员可访问操作其自身或子标签关联的资源
                    </template>
                    <span
                        >标签作用<el-icon>
                            <question-filled />
                        </el-icon>
                    </span>
                </el-tooltip>
            </div>
        </div>
        <el-tree
            ref="tagTreeRef"
            class="none-select"
            :indent="38"
            node-key="id"
            :props="props"
            :data="data"
            @node-expand="handleNodeExpand"
            @node-collapse="handleNodeCollapse"
            @node-contextmenu="nodeContextmenu"
            @node-click="treeNodeClick"
            :default-expanded-keys="defaultExpandedKeys"
            :expand-on-click-node="true"
            :filter-node-method="filterNode"
        >
            <template #default="{ data }">
                <span class="custom-tree-node">
                    <span style="font-size: 13px">
                        {{ data.code }}
                        <span style="color: #3c8dbc">【</span>
                        {{ data.name }}
                        <span style="color: #3c8dbc">】</span>
                        <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}</el-tag>
                    </span>
                </span>
            </template>
        </el-tree>

        <el-dialog width="500px" :title="saveTabDialog.title" :before-close="cancelSaveTag" v-model="saveTabDialog.visible">
            <el-form ref="tagForm" :rules="rules" :model="saveTabDialog.form" label-width="auto">
                <el-form-item prop="code" label="标识:" required>
                    <el-input :disabled="saveTabDialog.form.id ? true : false" v-model="saveTabDialog.form.code" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item prop="name" label="名称:" required>
                    <el-input v-model="saveTabDialog.form.name" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="备注:">
                    <el-input v-model="saveTabDialog.form.remark" auto-complete="off"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelSaveTag()">取 消</el-button>
                    <el-button @click="saveTag" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog v-model="infoDialog.visible">
            <el-descriptions title="节点信息" :column="2" border>
                <el-descriptions-item label="code">{{ infoDialog.data.code }}</el-descriptions-item>
                <el-descriptions-item label="code路径">{{ infoDialog.data.codePath }}</el-descriptions-item>
                <el-descriptions-item label="名称">{{ infoDialog.data.name }}</el-descriptions-item>
                <el-descriptions-item label="备注">{{ infoDialog.data.remark }}</el-descriptions-item>

                <el-descriptions-item label="创建者">{{ infoDialog.data.creator }}</el-descriptions-item>
                <el-descriptions-item label="创建时间">{{ dateFormat(infoDialog.data.createTime) }}</el-descriptions-item>
                <el-descriptions-item label="修改者">{{ infoDialog.data.modifier }}</el-descriptions-item>
                <el-descriptions-item label="更新时间">{{ dateFormat(infoDialog.data.updateTime) }}</el-descriptions-item>
            </el-descriptions>
        </el-dialog>

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, watch, reactive, onMounted } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { tagApi } from './api';
import { dateFormat } from '@/common/utils/date';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu/index';

interface Tree {
    id: number;
    codePath: string;
    name: string;
    children?: Tree[];
}

const tagForm: any = ref(null);
const tagTreeRef: any = ref(null);
const filterTag = ref('');
const contextmenuRef = ref();

const contextmenuInfo = new ContextmenuItem('info', '详情').withIcon('view').withOnClick((data: any) => info(data));

const contextmenuAdd = new ContextmenuItem('addTag', '添加子标签')
    .withIcon('circle-plus')
    .withPermission('tag:save')
    .withOnClick((data: any) => showSaveTagDialog(data));

const contextmenuEdit = new ContextmenuItem('edit', '编辑')
    .withIcon('edit')
    .withPermission('tag:save')
    .withOnClick((data: any) => showEditTagDialog(data));

const contextmenuDel = new ContextmenuItem('delete', '删除')
    .withIcon('delete')
    .withPermission('tag:del')
    .withHideFunc((data: any) => {
        // 存在子标签，则不允许删除
        return data.children;
    })
    .withOnClick((data: any) => deleteTag(data));

const state = reactive({
    data: [],
    saveTabDialog: {
        title: '新增标签',
        visible: false,
        form: { id: 0, pid: 0, code: '', name: '', remark: '' },
    },
    infoDialog: {
        title: '',
        visible: false,
        // 资源类型选择是否选
        data: null as any,
    },
    // 展开的节点
    defaultExpandedKeys: [] as any,
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [contextmenuInfo, contextmenuEdit, contextmenuAdd, contextmenuDel],
    },
});

const { data, saveTabDialog, infoDialog, defaultExpandedKeys } = toRefs(state);

const props = {
    label: 'name',
    children: 'children',
};

const rules = {
    code: [
        { required: true, message: '标识符不能为空', trigger: 'blur' },
        // {
        //     pattern: /^\w+$/g,
        //     message: '标识符只能为空数字字母下划线等',
        //     trigger: 'blur',
        // },
    ],
    name: [{ required: true, message: '名称不能为空', trigger: 'blur' }],
};

onMounted(() => {
    search();
});

watch(filterTag, (val) => {
    tagTreeRef.value!.filter(val);
});

const filterNode = (value: string, data: Tree) => {
    if (!value) return true;
    return data.codePath.includes(value) || data.name.includes(value);
};

const search = async () => {
    let res = await tagApi.getTagTrees.request(null);
    state.data = res;
};

// 树节点右击事件
const nodeContextmenu = (event: any, data: any) => {
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value.openContextmenu(data);
};

const treeNodeClick = () => {
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();
};

const info = async (data: any) => {
    state.infoDialog.data = data;
    state.infoDialog.visible = true;
};

const showSaveTagDialog = (data: any) => {
    if (data) {
        state.saveTabDialog.form.pid = data.id;
        state.saveTabDialog.title = `新增 [${data.codePath}] 子标签信息`;
    } else {
        state.saveTabDialog.title = '新增根标签信息';
    }
    state.saveTabDialog.visible = true;
};

const showEditTagDialog = (data: any) => {
    state.saveTabDialog.form.id = data.id;
    state.saveTabDialog.form.code = data.code;
    state.saveTabDialog.form.name = data.name;
    state.saveTabDialog.form.remark = data.remark;
    state.saveTabDialog.title = `修改 [${data.codePath}] 信息`;
    state.saveTabDialog.visible = true;
};

const saveTag = async () => {
    tagForm.value.validate(async (valid: any) => {
        if (valid) {
            const form = state.saveTabDialog.form;
            await tagApi.saveTagTree.request(form);
            ElMessage.success('保存成功');
            search();
            cancelSaveTag();
        }
    });
};

const cancelSaveTag = () => {
    state.saveTabDialog.visible = false;
    state.saveTabDialog.form = {} as any;
    tagForm.value.resetFields();
};

const deleteTag = (data: any) => {
    ElMessageBox.confirm(`此操作将删除 [${data.codePath}], 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(async () => {
        await tagApi.delTagTree.request({ id: data.id });
        ElMessage.success('删除成功！');
        search();
    });
};

// const changeStatus = async (data: any, status: any) => {
//     await resourceApi.changeStatus.request({
//         id: data.id,
//         status: status,
//     });
//     data.status = status;
//     ElMessage.success((status === 1 ? '启用' : '禁用') + '成功！');
// };

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
</script>
<style lang="scss">
.menu {
    height: 100%;

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
@/components/contextmenu
