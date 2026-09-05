<template>
    <div class="mongo-data-tab card h-full p-1! w-full flex flex-col">
        <el-row v-if="nowColl">
            <el-descriptions class="w-full!" :column="10" size="small" border>
                <!-- <el-descriptions-item label-align="right" label="tag">xxx</el-descriptions-item> -->

                <el-descriptions-item label="ns" label-align="right">
                    {{ nowColl.stats?.ns }}
                </el-descriptions-item>
                <el-descriptions-item label="count" label-align="right">
                    {{ nowColl.stats?.count }}
                </el-descriptions-item>
                <el-descriptions-item label="avgObjSize" label-align="right">
                    {{ formatByteSize(Number(nowColl.stats?.avgObjSize)) }}
                </el-descriptions-item>
                <el-descriptions-item label="size" label-align="right">
                    {{ formatByteSize(Number(nowColl.stats?.size)) }}
                </el-descriptions-item>
                <el-descriptions-item label="totalSize" label-align="right">
                    {{ formatByteSize(Number(nowColl.stats?.totalSize)) }}
                </el-descriptions-item>
                <el-descriptions-item label="storageSize" label-align="right">
                    {{ formatByteSize(Number(nowColl.stats?.storageSize)) }}
                </el-descriptions-item>
                <el-descriptions-item label="freeStorageSize" label-align="right">
                    {{ formatByteSize(Number(nowColl.stats?.freeStorageSize)) }}
                </el-descriptions-item>
            </el-descriptions>
        </el-row>

        <el-row type="flex" class="flex-1 min-h-0">
            <el-tabs @tab-remove="removeDataTab" class="w-full! ml-1 h-full flex flex-col" v-model="state.activeName">
                <el-tab-pane closable v-for="dt in state.dataTabs" :key="dt.key" :label="dt.label" :name="dt.key">
                    <el-row>
                        <el-col :span="2">
                            <div class="mt-1">
                                <el-link @click="findCommand(state.activeName)" icon="refresh" underline="never" class=""> </el-link>
                                <el-divider direction="vertical" border-style="dashed" />
                                <el-link v-auth="perms.saveData" @click="onEditDoc(null)" type="primary" icon="plus" underline="never"> </el-link>
                            </div>
                        </el-col>
                        <el-col :span="22">
                            <el-input
                                ref="findParamInputRef"
                                v-model="dt.findParamStr"
                                :placeholder="$t('mongo.queryParamPlaceholder')"
                                @focus="showFindDialog(dt.key)"
                            >
                                <template #prepend>{{ $t('mongo.queryParam') }}</template>
                            </el-input>
                        </el-col>
                    </el-row>
                    <el-scrollbar class="mongo-data-tab-data flex-1 min-h-0" v-loading="findLoading">
                        <el-row>
                            <el-col :span="6" v-for="item in dt.datas" :key="item">
                                <el-card :body-style="{ padding: '0px', position: 'relative' }">
                                    <el-input type="textarea" v-model="item.value" :rows="10" />
                                    <div style="padding: 3px; float: right" class="mr-1 mongo-doc-btns">
                                        <div>
                                            <el-link @click="onEditDoc(item)" underline="never" type="success" icon="MagicStick"></el-link>

                                            <el-divider direction="vertical" border-style="dashed" />

                                            <el-popconfirm @confirm="onDeleteDoc(item.value)" :title="$t('mongo.deleteDocConfirm')" width="160">
                                                <template #reference>
                                                    <el-link v-auth="perms.delData" underline="never" type="danger" icon="DocumentDelete"> </el-link>
                                                </template>
                                            </el-popconfirm>
                                        </div>
                                    </div>
                                </el-card>
                            </el-col>
                        </el-row>
                    </el-scrollbar>
                </el-tab-pane>
            </el-tabs>
        </el-row>

        <el-dialog width="600px" title="find params" v-model="findDialog.visible">
            <auto-form v-model="findDialog.findParam" :items="findParamItems" label-width="auto">
                <template #filter>
                    <monaco-editor style="width: 100%" height="150px" ref="monacoEditorRef" v-model="findDialog.findParam.filter" language="json" />
                </template>
            </auto-form>
            <template #footer>
                <div>
                    <el-button @click="findDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button @click="confirmFindDialog" type="primary">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>

        <el-dialog
            width="60%"
            :title="`${state.docEditDialog.isAdd ? $t('common.add') : $t('common.edit')} '${state.activeName}' $t('mongo.doc')`"
            v-model="docEditDialog.visible"
            :close-on-click-modal="false"
        >
            <monaco-editor v-model="docEditDialog.doc" language="json" />
            <template #footer>
                <div>
                    <el-button @click="docEditDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                    <el-button v-auth="perms.saveData" @click="onSaveDoc" type="primary">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { isTrue, notBlank } from '@/common/assert';
import type { AutoFormItem } from '@/components/auto-form';
import { formatByteSize } from '@/common/utils/format';
import { Msg } from '@/hooks/useI18n';
import { mongoApi } from '@/views/ops/mongo/api';
import { computed, defineAsyncComponent, onMounted, reactive, ref, toRefs } from 'vue';
import type { InputInstance } from 'element-plus';
import { useI18n } from 'vue-i18n';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

/** Mongo 数据Tab信息 */
interface MongoDataTab {
    key: string;
    label: string;
    name: string;
    mongoId: number;
    database: string;
    collection: string;
    datas: { value: string }[];
    findParamStr: string;
    findParam: { filter: string; sort: string; skip: number; limit: number };
    stats?: Record<string, unknown>;
}

const { t } = useI18n();

const perms = {
    saveData: 'mongo:data:save',
    delData: 'mongo:data:del',
};

const props = defineProps<{
    tabKey?: string;
}>();

const emits = defineEmits(['init']);

const findParamInputRef = ref<InputInstance[]>([]);

const state = reactive({
    defaultExpendKey: [] as string[],
    tags: [],
    mongoList: [] as Record<string, unknown>[],
    activeName: '', // 当前操作的tab
    dataTabs: {} as Record<string, MongoDataTab>, // 数据tabs
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
        item: {} as Record<string, unknown>,
    },
});

const { findDialog, docEditDialog } = toRefs(state);

/** find params 表单声明（filter 编辑器为 custom 插槽） */
const findParamItems: AutoFormItem[] = [
    { prop: 'filter', label: 'filter', type: 'custom' },
    { prop: 'sort', label: 'sort', type: 'textarea', props: { rows: 3, clearable: true, autoComplete: 'off' } },
    { prop: 'limit', label: 'limit', type: 'number' },
    { prop: 'skip', label: 'skip', type: 'number' },
];

const nowColl = computed(() => {
    return getNowDataTab();
});

onMounted(() => {});

const changeCollection = async (id: number, schema: string, collection: string) => {
    const label = `${schema}.${collection}`;
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
            findParamInputRef.value[i]?.blur();
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

const findLoading = ref(false);

const findCommand = async (key: string) => {
    const dataTab = getNowDataTab();
    const findParma = dataTab.findParam;
    let filter, sort;
    try {
        filter = findParma.filter ? JSON.parse(findParma.filter) : {};
        sort = findParma.sort ? JSON.parse(findParma.sort) : {};
    } catch (e) {
        Msg.error('mongo.findParamErrMsg');
        return;
    }

    try {
        findLoading.value = true;
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
    } finally {
        findLoading.value = false;
    }
};

/**
 * 包装mongo查询回来的对象，即将其都转为json字符串并用value属性值描述，方便显示
 */
const wrapDatas = (datas: Record<string, unknown>[]) => {
    const wrapDatas = [] as { value: string }[];
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

const onEditDoc = async (item: { value: string } | null) => {
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
            Msg.error('mongo.docErrMsg');
        }
        const dataTab = getNowDataTab();
        const res = await mongoApi.insertCommand.request({
            id: dataTab.mongoId,
            database: dataTab.database,
            collection: dataTab.collection,
            doc: docObj,
        });
        isTrue(!!res.InsertedID, 'mongo.insertFail');
        Msg.success('mongo.insertSuccess');
    } else {
        const docObj = parseDocJsonString(state.docEditDialog.doc);
        const id = docObj._id;
        notBlank(id, t('mongo.idNotExist'));
        delete docObj['_id'];
        const dataTab = getNowDataTab();
        const res = await mongoApi.updateByIdCommand.request({
            id: dataTab.mongoId,
            database: dataTab.database,
            collection: dataTab.collection,
            docId: id,
            update: { $set: docObj },
        });
        isTrue(res.ModifiedCount == 1, 'common.modifyFail');
        Msg.saveSuccess();
    }
    findCommand(state.activeName);
    state.docEditDialog.visible = false;
};

const onDeleteDoc = async (doc: string) => {
    const docObj = parseDocJsonString(doc);
    const id = docObj._id;
    notBlank(id, t('mongo.idNotExist'));
    const dataTab = getNowDataTab();
    const res = await mongoApi.deleteByIdCommand.request({
        id: dataTab.mongoId,
        database: dataTab.database,
        collection: dataTab.collection,
        docId: id,
    });
    isTrue(res.DeletedCount == 1, 'common.deleteFail');
    Msg.deleteSuccess();
    findCommand(state.activeName);
};

/**
 * 将json字符串解析为json对象
 */
const parseDocJsonString = (doc: string) => {
    try {
        return JSON.parse(doc);
    } catch (e) {
        Msg.error('mongo.docParse2jsonFail');
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

defineExpose({
    changeCollection,
    onRefresh: () => {
        findCommand(state.activeName);
    },
});
</script>

<style lang="scss">
.mongo-doc-btns {
    position: absolute;
    z-index: 2;
    right: 3px;
    top: 2px;
    max-width: 120px;
}

.mongo-data-tab {
    .el-tabs__content {
        flex: 1;
        min-height: 0;
        overflow: hidden;

        .el-tab-pane {
            height: 100%;
            display: flex;
            flex-direction: column;
        }
    }

    .el-tabs__header {
        margin: 0 0 5px;

        .el-tabs__item {
            padding: 0 5px;
        }
    }
}
</style>
