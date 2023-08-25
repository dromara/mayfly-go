<template>
    <div>
        <el-row>
            <el-col :span="4">
                <tag-tree @node-click="nodeClick" :load="loadNode">
                    <template #prefix="{ data }">
                        <span v-if="data.type == NodeType.Mongo">
                            <el-popover placement="right-start" title="mongo实例信息" trigger="hover" :width="210">
                                <template #reference>
                                    <SvgIcon name="iconfont icon-op-mongo" :size="18" />
                                </template>
                                <template #default>
                                    <el-form class="instances-pop-form" label-width="auto" :size="'small'">
                                        <el-form-item label="名称:">{{ data.params.name }}</el-form-item>
                                        <el-form-item label="链接:">{{ data.params.uri }}</el-form-item>
                                    </el-form>
                                </template>
                            </el-popover>
                        </span>

                        <SvgIcon v-if="data.type == NodeType.Dbs" name="Coin" color="#67c23a" />

                        <SvgIcon v-if="data.type == NodeType.Coll || data.type == NodeType.CollMenu" name="Document" class="color-primary" />
                    </template>

                    <template #label="{ data }">
                        <span v-if="data.type == NodeType.Dbs">
                            {{ data.params.dbName }}
                            <span style="color: #8492a6; font-size: 13px"> [{{ formatByteSize(data.params.size) }}] </span>
                        </span>

                        <span v-else>{{ data.label }}</span>
                    </template>
                </tag-tree>
            </el-col>

            <el-col :span="20">
                <div id="mongo-tab" style="border: 1px solid #eee; margin-top: 1px">
                    <el-row v-if="nowColl">
                        <el-descriptions :column="10" size="small" border>
                            <!-- <el-descriptions-item label-align="right" label="tag">xxx</el-descriptions-item> -->

                            <el-descriptions-item label="ns" label-align="right">
                                {{ nowColl.stats.ns }}
                            </el-descriptions-item>
                            <el-descriptions-item label="count" label-align="right">
                                {{ nowColl.stats.count }}
                            </el-descriptions-item>
                            <el-descriptions-item label="avgObjSize" label-align="right">
                                {{ formatByteSize(nowColl.stats.avgObjSize) }}
                            </el-descriptions-item>
                            <el-descriptions-item label="size" label-align="right">
                                {{ formatByteSize(nowColl.stats.size) }}
                            </el-descriptions-item>
                            <el-descriptions-item label="totalSize" label-align="right">
                                {{ formatByteSize(nowColl.stats.totalSize) }}
                            </el-descriptions-item>
                            <el-descriptions-item label="storageSize" label-align="right">
                                {{ formatByteSize(nowColl.stats.storageSize) }}
                            </el-descriptions-item>
                            <el-descriptions-item label="freeStorageSize" label-align="right">
                                {{ formatByteSize(nowColl.stats.freeStorageSize) }}
                            </el-descriptions-item>
                        </el-descriptions>
                    </el-row>

                    <el-row type="flex">
                        <el-tabs @tab-remove="removeDataTab" style="width: 100%; margin-left: 5px" v-model="state.activeName">
                            <el-tab-pane closable v-for="dt in state.dataTabs" :key="dt.key" :label="dt.label" :name="dt.key">
                                <el-row>
                                    <el-col :span="2">
                                        <div class="mt5">
                                            <el-link @click="findCommand(state.activeName)" icon="refresh" :underline="false" class=""> </el-link>
                                            <el-divider direction="vertical" border-style="dashed" />
                                            <el-link v-auth="perms.saveData" @click="onEditDoc(null)" type="primary" icon="plus" :underline="false"> </el-link>
                                        </div>
                                    </el-col>
                                    <el-col :span="22">
                                        <el-input
                                            ref="findParamInputRef"
                                            v-model="dt.findParamStr"
                                            placeholder="点击输入相应查询条件"
                                            @focus="showFindDialog(dt.key)"
                                        >
                                            <template #prepend>查询参数</template>
                                        </el-input>
                                    </el-col>
                                </el-row>
                                <el-row :style="`height: ${dataHeight}; overflow: auto;`">
                                    <el-col :span="6" v-for="item in dt.datas" :key="item">
                                        <el-card :body-style="{ padding: '0px', position: 'relative' }">
                                            <el-input type="textarea" v-model="item.value" :rows="10" />
                                            <div style="padding: 3px; float: right" class="mr5 mongo-doc-btns">
                                                <div>
                                                    <el-link @click="onEditDoc(item)" :underline="false" type="success" icon="MagicStick"></el-link>

                                                    <el-divider direction="vertical" border-style="dashed" />

                                                    <el-popconfirm @confirm="onDeleteDoc(item.value)" title="确定删除该文档?" width="160">
                                                        <template #reference>
                                                            <el-link v-auth="perms.delData" :underline="false" type="danger" icon="DocumentDelete"> </el-link>
                                                        </template>
                                                    </el-popconfirm>
                                                </div>
                                            </div>
                                        </el-card>
                                    </el-col>
                                </el-row>
                            </el-tab-pane>
                        </el-tabs>
                    </el-row>
                </div>
            </el-col>
        </el-row>

        <el-dialog width="600px" title="find参数" v-model="findDialog.visible">
            <el-form label-width="auto">
                <el-form-item label="filter">
                    <monaco-editor style="width: 100%" height="150px" ref="monacoEditorRef" v-model="findDialog.findParam.filter" language="json" />
                </el-form-item>
                <el-form-item label="sort">
                    <el-input v-model="findDialog.findParam.sort" type="textarea" :rows="3" clearable auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="limit">
                    <el-input v-model.number="findDialog.findParam.limit" type="number" auto-complete="off"></el-input>
                </el-form-item>
                <el-form-item label="skip">
                    <el-input v-model.number="findDialog.findParam.skip" type="number" auto-complete="off"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div>
                    <el-button @click="findDialog.visible = false">取 消</el-button>
                    <el-button @click="confirmFindDialog" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog
            width="60%"
            :title="`${state.docEditDialog.isAdd ? '新增' : '修改'}'${state.activeName}'集合文档`"
            v-model="docEditDialog.visible"
            :close-on-click-modal="false"
        >
            <monaco-editor v-model="docEditDialog.doc" language="json" />
            <template #footer>
                <div>
                    <el-button @click="docEditDialog.visible = false">取 消</el-button>
                    <el-button v-auth="perms.saveData" @click="onSaveDoc" type="primary">确 定</el-button>
                </div>
            </template>
        </el-dialog>

        <div style="text-align: center; margin-top: 10px"></div>
    </div>
</template>

<script lang="ts" setup>
import { mongoApi } from './api';
import { computed, defineAsyncComponent, reactive, ref, toRefs } from 'vue';
import { ElMessage } from 'element-plus';

import { isTrue, notBlank } from '@/common/assert';
import { TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { formatByteSize } from '@/common/utils/format';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const perms = {
    saveData: 'mongo:data:save',
    delData: 'mongo:data:del',
};

/**
 * 树节点类型
 */
class NodeType {
    static Mongo = 1;
    static Dbs = 2;
    static CollMenu = 3;
    static Coll = 4;
}

const findParamInputRef: any = ref(null);
const state = reactive({
    tags: [],
    dataHeight: `${window.innerHeight - 194 - 35}px`,
    mongoList: [] as any,
    activeName: '', // 当前操作的tab
    dataTabs: {} as any, // 数据tabs
    findDialog: {
        visible: false,
        findParam: {
            limit: 0,
            skip: 0,
            filter: '',
            sort: '',
        },
    },
    docEditDialog: {
        isAdd: true,
        visible: false,
        doc: '',
    },
    insertDocDialog: {
        visible: false,
        doc: '',
    },
    jsonEditorDialog: {
        visible: false,
        doc: '',
        item: {} as any,
    },
});

const { dataHeight, findDialog, docEditDialog } = toRefs(state);

const nowColl = computed(() => {
    return getNowDataTab();
});

/**
 * instmap;  tagPaht -> mongo info[]
 */
const instMap: Map<string, any[]> = new Map();

const getInsts = async () => {
    const res = await mongoApi.mongoList.request({ pageNum: 1, pageSize: 1000 });
    if (!res.total) return;
    for (const mongoInfo of res.list) {
        const tagPath = mongoInfo.tagPath;
        let mongoInsts = instMap.get(tagPath) || [];
        mongoInsts.push(mongoInfo);
        instMap.set(tagPath, mongoInsts);
    }
};

/**
 * 加载文件树节点
 * @param {Object} node
 * @param {Object} resolve
 */
const loadNode = async (node: any) => {
    // 一级为tagPath
    if (node.level === 0) {
        await getInsts();
        const tagPaths = instMap.keys();
        const tagNodes = [];
        for (let tagPath of tagPaths) {
            tagNodes.push(new TagTreeNode(tagPath, tagPath));
        }
        return tagNodes;
    }

    const data = node.data;
    const params = data.params;
    const nodeType = data.type;

    // 点击标签 -> 显示mongo信息列表
    if (nodeType === TagTreeNode.TagPath) {
        const mongoInfos = instMap.get(data.key);
        return mongoInfos?.map((x: any) => {
            return new TagTreeNode(`${data.key}.${x.id}`, x.name, NodeType.Mongo).withParams(x);
        });
    }

    // 点击mongo -> 加载mongo数据库列表
    if (nodeType === NodeType.Mongo) {
        return await getDatabases(params);
    }

    // 点击数据库列表 -> 加载数据库下拥有的菜单列表
    if (nodeType === NodeType.Dbs) {
        return [new TagTreeNode(`${params.id}.${params.dbName}.mongo-coll`, '集合', NodeType.CollMenu).withParams(params)];
    }

    // 点击数据库集合节点 -> 加载集合列表
    if (nodeType === NodeType.CollMenu) {
        return await getCollections(params.id, params.dbName);
    }

    return [];
};

/**
 * 获取实例的所有库信息
 * @param inst 实例信息
 */
const getDatabases = async (inst: any) => {
    const res = await mongoApi.databases.request({ id: inst.id });
    return res.Databases.map((x: any) => {
        const dbName = x.Name;
        return new TagTreeNode(`${inst.id}.${dbName}`, dbName, NodeType.Dbs).withParams({
            id: inst.id,
            dbName,
            size: x.SizeOnDisk,
        });
    });
};

/**
 * 获取集合列表信息
 * @param inst
 */
const getCollections = async (id: any, database: string) => {
    const colls = await mongoApi.collections.request({ id, database });
    return colls.map((x: any) => {
        return new TagTreeNode(`${id}.${database}.${x}`, x, NodeType.Coll).withIsLeaf(true).withParams({
            id,
            database,
            collection: x,
        });
    });
};

const nodeClick = async (data: any) => {
    // 点击集合
    if (data.type === NodeType.Coll) {
        const { id, database, collection } = data.params;
        await changeCollection(id, database, collection);
    }
};

const changeCollection = async (id: any, schema: string, collection: string) => {
    const label = `${id}:\`${schema}\`.${collection}`;
    let dataTab = state.dataTabs[label];
    if (!dataTab) {
        // 默认查询参数
        const findParam = {
            filter: '{}',
            sort: '{"_id": -1}',
            skip: 0,
            limit: 12,
        };
        state.dataTabs[label] = {
            key: label,
            label: label,
            name: label,
            mongoId: id,
            database: schema,
            collection,
            datas: [],
            findParamStr: JSON.stringify(findParam),
            findParam,
        };
    }
    state.activeName = label;
    findCommand(label);
};

const showFindDialog = (key: string) => {
    // 获取当前tab的索引位置，将其输入框失去焦点，防止输入以及重复获取焦点
    const dataTabNames = Object.keys(state.dataTabs);
    for (let i = 0; i < dataTabNames.length; i++) {
        if (key == dataTabNames[i]) {
            findParamInputRef.value[i].blur();
        }
    }

    state.findDialog.findParam = state.dataTabs[key].findParam;
    state.findDialog.visible = true;
};

const confirmFindDialog = () => {
    state.dataTabs[state.activeName].findParam = state.findDialog.findParam;
    state.dataTabs[state.activeName].findParamStr = JSON.stringify(state.findDialog.findParam);
    state.findDialog.visible = false;
    findCommand(state.activeName);
};

const findCommand = async (key: string) => {
    const dataTab = getNowDataTab();
    const findParma = dataTab.findParam;
    let filter, sort;
    try {
        filter = findParma.filter ? JSON.parse(findParma.filter) : {};
        sort = findParma.sort ? JSON.parse(findParma.sort) : {};
    } catch (e) {
        ElMessage.error('filter或sort字段json字符串值错误。注意: json key需双引号');
        return;
    }

    const datas = await mongoApi.findCommand.request({
        id: dataTab.mongoId,
        database: dataTab.database,
        collection: dataTab.collection,
        filter,
        sort,
        limit: findParma.limit || 12,
        skip: findParma.skip || 0,
    });
    state.dataTabs[key].datas = wrapDatas(datas);

    // 获取coll stats
    state.dataTabs[key].stats = await mongoApi.runCommand.request({
        id: dataTab.mongoId,
        database: dataTab.database,
        command: [
            {
                collStats: dataTab.collection,
            },
        ],
    });
};

/**
 * 包装mongo查询回来的对象，即将其都转为json字符串并用value属性值描述，方便显示
 */
const wrapDatas = (datas: any) => {
    const wrapDatas = [] as any;
    if (!datas) {
        return wrapDatas;
    }
    for (let data of datas) {
        wrapDatas.push({ value: JSON.stringify(data, null, 4) });
    }
    return wrapDatas;
};

const showEditDocDialog = () => {
    // tab数据中的第一个文档，因为该集合的文档都类似，故使用第一个文档赋值至需要新增的文档输入框，方便直接修改新增
    const datasFirstDoc = state.dataTabs[state.activeName].datas[0];
    let doc = '';
    if (datasFirstDoc) {
        // 移除_id字段，因为新增无需该字段
        const docObj = JSON.parse(datasFirstDoc.value);
        delete docObj['_id'];
        doc = JSON.stringify(docObj, null, 4);
    }
    state.docEditDialog.doc = doc;
    state.docEditDialog.visible = true;
};

const onEditDoc = async (item: any) => {
    // 新增文档
    if (!item) {
        state.docEditDialog.isAdd = true;
        showEditDocDialog();
        return;
    }
    // 编辑修改文档
    // state.docEditDialog.item = item;
    state.docEditDialog.isAdd = false;
    state.docEditDialog.doc = item.value;
    state.docEditDialog.visible = true;
};

const onSaveDoc = async () => {
    if (state.docEditDialog.isAdd) {
        let docObj;
        try {
            docObj = JSON.parse(state.docEditDialog.doc);
        } catch (e) {
            ElMessage.error('文档内容错误,无法解析为json对象');
        }
        const dataTab = getNowDataTab();
        const res = await mongoApi.insertCommand.request({
            id: dataTab.mongoId,
            database: dataTab.database,
            collection: dataTab.collection,
            doc: docObj,
        });
        isTrue(res.InsertedID, '新增失败');
        ElMessage.success('新增成功');
    } else {
        const docObj = parseDocJsonString(state.docEditDialog.doc);
        const id = docObj._id;
        notBlank(id, '文档的_id属性不存在');
        delete docObj['_id'];
        const dataTab = getNowDataTab();
        const res = await mongoApi.updateByIdCommand.request({
            id: dataTab.mongoId,
            database: dataTab.database,
            collection: dataTab.collection,
            docId: id,
            update: { $set: docObj },
        });
        isTrue(res.ModifiedCount == 1, '修改失败');
        ElMessage.success('保存成功');
    }
    findCommand(state.activeName);
    state.docEditDialog.visible = false;
};

const onDeleteDoc = async (doc: string) => {
    const docObj = parseDocJsonString(doc);
    const id = docObj._id;
    notBlank(id, '文档的_id属性不存在');
    const dataTab = getNowDataTab();
    const res = await mongoApi.deleteByIdCommand.request({
        id: dataTab.mongoId,
        database: dataTab.database,
        collection: dataTab.collection,
        docId: id,
    });
    isTrue(res.DeletedCount == 1, '删除失败');
    ElMessage.success('删除成功');
    findCommand(state.activeName);
};

/**
 * 将json字符串解析为json对象
 */
const parseDocJsonString = (doc: string) => {
    try {
        return JSON.parse(doc);
    } catch (e) {
        ElMessage.error('文档内容解析为json对象失败');
        throw e;
    }
};

const removeDataTab = (targetName: string) => {
    const tabNames = Object.keys(state.dataTabs);
    let activeName = state.activeName;
    tabNames.forEach((name, index) => {
        if (name === targetName) {
            const nextTab = tabNames[index + 1] || tabNames[index - 1];
            if (nextTab) {
                activeName = nextTab;
            }
        }
    });
    state.activeName = activeName;
    delete state.dataTabs[targetName];
};

const getNowDataTab = () => {
    return state.dataTabs[state.activeName];
};
</script>

<style lang="scss">
.mongo-doc-btns {
    position: absolute;
    z-index: 2;
    right: 3px;
    top: 2px;
    max-width: 120px;
}

#mongo-tab {
    .el-tabs__header {
        margin: 0 0 5px;

        .el-tabs__item {
            padding: 0 5px;
        }
    }
}
</style>
