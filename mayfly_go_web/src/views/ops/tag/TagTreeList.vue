<template>
    <div class="tag-tree-list card">
        <Splitpanes class="default-theme">
            <Pane size="30" min-size="25" max-size="35">
                <div class="card pd5 mr5">
                    <el-input v-model="filterTag" clearable placeholder="关键字过滤(右击操作)" style="width: 200px; margin-right: 10px" />
                    <el-button
                        v-if="useUserInfo().userInfo.username == 'admin'"
                        v-auth="'tag:save'"
                        type="primary"
                        icon="plus"
                        @click="showSaveTagDialog(null)"
                    ></el-button>
                    <div style="float: right">
                        <el-tooltip placement="top">
                            <template #content>
                                1. 用于将资产进行归类
                                <br />2. 可在团队管理中进行分配，用于资源隔离 <br />3. 拥有父标签的团队成员可访问操作其自身或子标签关联的资源
                            </template>
                            <span>
                                <el-icon>
                                    <question-filled />
                                </el-icon>
                            </span>
                        </el-tooltip>
                    </div>
                </div>
                <el-scrollbar class="tag-tree-data">
                    <el-tree
                        ref="tagTreeRef"
                        node-key="id"
                        highlight-current
                        :props="props"
                        :data="data"
                        @node-expand="handleNodeExpand"
                        @node-collapse="handleNodeCollapse"
                        @node-contextmenu="nodeContextmenu"
                        @node-click="treeNodeClick"
                        :default-expanded-keys="defaultExpandedKeys"
                        :expand-on-click-node="false"
                        :filter-node-method="filterNode"
                    >
                        <template #default="{ data }">
                            <span class="custom-tree-node">
                                <SvgIcon :name="EnumValue.getEnumByValue(TagResourceTypeEnum, data.type)?.extra.icon" />

                                <span class="ml5">
                                    {{ data.code }}
                                    <span style="color: #3c8dbc">【</span>
                                    {{ data.name }}
                                    <span style="color: #3c8dbc">】</span>
                                    <el-tag v-if="data.children !== null" size="small">{{ data.children.length }}</el-tag>
                                </span>
                            </span>
                        </template>
                    </el-tree>
                </el-scrollbar>
            </Pane>

            <Pane min-size="40">
                <div class="ml10">
                    <el-tabs @tab-change="tabChange" v-model="state.activeTabName" v-if="currentTag">
                        <el-tab-pane label="标签详情" :name="TagDetail">
                            <el-descriptions :column="2" border>
                                <el-descriptions-item label="类型">
                                    <EnumTag :enums="TagResourceTypeEnum" :value="currentTag.type" />
                                </el-descriptions-item>

                                <el-descriptions-item label="code">{{ currentTag.code }}</el-descriptions-item>
                                <el-descriptions-item label="code路径">{{ currentTag.codePath }}</el-descriptions-item>
                                <el-descriptions-item label="名称">{{ currentTag.name }}</el-descriptions-item>
                                <el-descriptions-item label="备注">{{ currentTag.remark }}</el-descriptions-item>

                                <el-descriptions-item label="创建者">{{ currentTag.creator }}</el-descriptions-item>
                                <el-descriptions-item label="创建时间">{{ dateFormat(currentTag.createTime) }}</el-descriptions-item>
                                <el-descriptions-item label="修改者">{{ currentTag.modifier }}</el-descriptions-item>
                                <el-descriptions-item label="更新时间">{{ dateFormat(currentTag.updateTime) }}</el-descriptions-item>
                            </el-descriptions>
                        </el-tab-pane>

                        <el-tab-pane v-if="currentTag.type == TagResourceTypeEnum.Tag.value" :label="`机器 (${resourceCount.machine})`" :name="MachineTag">
                            <MachineList lazy ref="machineListRef" />
                        </el-tab-pane>

                        <el-tab-pane v-if="currentTag.type == TagResourceTypeEnum.Tag.value" :label="`数据库 (${resourceCount.db})`" :name="DbTag">
                            <DbList lazy ref="dbListRef" />
                        </el-tab-pane>

                        <el-tab-pane v-if="currentTag.type == TagResourceTypeEnum.Tag.value" :label="`Redis (${resourceCount.redis})`" :name="RedisTag">
                            <RedisList lazy ref="redisListRef" />
                        </el-tab-pane>

                        <el-tab-pane v-if="currentTag.type == TagResourceTypeEnum.Tag.value" :label="`Mongo (${resourceCount.mongo})`" :name="MongoTag">
                            <MongoList lazy ref="mongoListRef" />
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </Pane>
        </Splitpanes>

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

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, watch, reactive, onMounted, Ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { tagApi } from './api';
import { dateFormat } from '@/common/utils/date';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu/index';
import { useUserInfo } from '@/store/userInfo';
import { Splitpanes, Pane } from 'splitpanes';
import MachineList from '../machine/MachineList.vue';
import RedisList from '../redis/RedisList.vue';
import MongoList from '../mongo/MongoList.vue';
import DbList from '../db/DbList.vue';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import EnumValue from '@/common/Enum';

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
const machineListRef: Ref<any> = ref(null);
const dbListRef: Ref<any> = ref(null);
const redisListRef: Ref<any> = ref(null);
const mongoListRef: Ref<any> = ref(null);

const TagDetail = 'tagDetail';
const MachineTag = 'machineTag';
const DbTag = 'dbTag';
const RedisTag = 'redisTag';
const MongoTag = 'mongoTag';

const contextmenuAdd = new ContextmenuItem('addTag', '添加子标签')
    .withIcon('circle-plus')
    .withPermission('tag:save')
    .withHideFunc((data: any) => {
        // 非标签类型不可添加子标签
        return data.type != TagResourceTypeEnum.Tag.value || (data.children && data.children?.[0].type != TagResourceTypeEnum.Tag.value);
    })
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
        return data.children || data.type != TagResourceTypeEnum.Tag.value;
    })
    .withOnClick((data: any) => deleteTag(data));

const state = reactive({
    data: [],
    saveTabDialog: {
        title: '新增标签',
        visible: false,
        form: { id: 0, pid: 0, code: '', name: '', remark: '' },
    },
    resourceDialog: {
        title: '',
        visible: false,
        tagPath: '',
        data: null as any,
    },
    // 展开的节点
    defaultExpandedKeys: [] as any,
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [contextmenuEdit, contextmenuAdd, contextmenuDel],
    },
    activeTabName: TagDetail,
    currentTag: null as any,
    resourceCount: {} as any,
});

const { data, saveTabDialog, currentTag, resourceCount, defaultExpandedKeys } = toRefs(state);

const props = {
    label: 'name',
    children: 'children',
};

const rules = {
    code: [{ required: true, message: '标识符不能为空', trigger: 'blur' }],
    name: [{ required: true, message: '名称不能为空', trigger: 'blur' }],
};

onMounted(() => {
    search();
});

watch(filterTag, (val) => {
    tagTreeRef.value!.filter(val);
});

watch(
    () => state.currentTag,
    (val: any) => {
        tagApi.countTagResource.request({ tagPath: val.codePath }).then((res: any) => {
            state.resourceCount = res;
        });
        setNowTabData();
    }
);

const tabChange = () => {
    setNowTabData();
};

const setNowTabData = () => {
    const tagPath = state.currentTag.codePath;
    switch (state.activeTabName) {
        case MachineTag:
            machineListRef.value.search(tagPath);
            break;
        case DbTag:
            dbListRef.value.search(tagPath);
            break;
        case RedisTag:
            redisListRef.value.search(tagPath);
            break;
        case MongoTag:
            mongoListRef.value.search(tagPath);
            break;
        default:
            break;
    }
};

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

const treeNodeClick = (data: any) => {
    state.currentTag = data;
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();
};

const showSaveTagDialog = (data: any) => {
    if (data) {
        state.saveTabDialog.form.pid = data.id;
        state.saveTabDialog.title = `新增[ ${data.codePath} ]子标签信息`;
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
            state.currentTag = null;
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
.tag-tree-list {
    .tag-tree-data {
        height: calc(100vh - 202px);

        .el-tree-node__content {
            height: 40px;
            line-height: 40px;
        }
    }

    .el-tree {
        display: inline-block;
        min-width: 100%;
    }
}
</style>
