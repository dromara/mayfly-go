<template>
    <div class="mongo-data-tab card h-full !p-1 w-full">
        <el-row v-if="nowColl">
            <el-descriptions class="!w-full" :column="10" size="small" border>
                <!-- <el-descriptions-item label-align="right" label="tag">xxx</el-descriptions-item> -->

                <el-descriptions-item label="ns" label-align="right">
                    {{ nowColl.stats?.ns }}
                </el-descriptions-item>
                <el-descriptions-item label="count" label-align="right">
                    {{ nowColl.stats?.count }}
                </el-descriptions-item>
                <el-descriptions-item label="avgObjSize" label-align="right">
                    {{ formatByteSize(nowColl.stats?.avgObjSize) }}
                </el-descriptions-item>
                <el-descriptions-item label="size" label-align="right">
                    {{ formatByteSize(nowColl.stats?.size) }}
                </el-descriptions-item>
                <el-descriptions-item label="totalSize" label-align="right">
                    {{ formatByteSize(nowColl.stats?.totalSize) }}
                </el-descriptions-item>
                <el-descriptions-item label="storageSize" label-align="right">
                    {{ formatByteSize(nowColl.stats?.storageSize) }}
                </el-descriptions-item>
                <el-descriptions-item label="freeStorageSize" label-align="right">
                    {{ formatByteSize(nowColl.stats?.freeStorageSize) }}
                </el-descriptions-item>
            </el-descriptions>
        </el-row>

        <el-row type="flex">
            <el-tabs @tab-remove="removeDataTab" class="!w-full ml-1" v-model="state.activeName">
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
                    <el-scrollbar class="mongo-data-tab-data">
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
import { computed, defineAsyncComponent, getCurrentInstance, inject, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { ElMessage } from 'element-plus';
import { isTrue, notBlank } from '@/common/assert';
import { formatByteSize } from '@/common/utils/format';
import { useI18n } from 'vue-i18n';
import { useI18nDeleteSuccessMsg, useI18nSaveSuccessMsg } from '@/hooks/useI18n';
import { mongoApi } from '@/views/ops/mongo/api';
import { ResourceOpCtxKey } from '@/views/ops/resource/resource';
import { MongoOpComp } from '@/views/ops/mongo/resource';
import { ResourceOpCtx } from '../../component/tag';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

const { t } = useI18n();

const perms = {
    saveData: 'mongo:data:save',
    delData: 'mongo:data:del',
};

const resourceOpCtx: ResourceOpCtx | undefined = inject(ResourceOpCtxKey);

const emits = defineEmits(['init']);

const findParamInputRef: any = ref(null);

const state = reactive({
    defaultExpendKey: [] as any,
    tags: [],
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

const { findDialog, docEditDialog } = toRefs(state);

const nowColl = computed(() => {
    return getNowDataTab();
});

onMounted(() => {
    emits('init', { name: MongoOpComp.name, ref: getCurrentInstance()?.exposed });
});

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
        ElMessage.error(t('mongo.findParamErrMsg'));
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
            ElMessage.error(t('mongo.docErrMsg'));
        }
        const dataTab = getNowDataTab();
        const res = await mongoApi.insertCommand.request({
            id: dataTab.mongoId,
            database: dataTab.database,
            collection: dataTab.collection,
            doc: docObj,
        });
        isTrue(res.InsertedID, 'mongo.insertFail');
        ElMessage.success(t('mongo.insertSuccess'));
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
        useI18nSaveSuccessMsg();
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
    useI18nDeleteSuccessMsg();
    findCommand(state.activeName);
};

/**
 * 将json字符串解析为json对象
 */
const parseDocJsonString = (doc: string) => {
    try {
        return JSON.parse(doc);
    } catch (e) {
        ElMessage.error(t('mongo.docParse2jsonFail'));
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
    .mongo-data-tab-data {
        height: calc(100vh - 230px);
    }

    .el-tabs__header {
        margin: 0 0 5px;

        .el-tabs__item {
            padding: 0 5px;
        }
    }
}
</style>
