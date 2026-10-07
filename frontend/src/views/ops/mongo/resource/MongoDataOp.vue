<template>
    <div class="mongo-data-op card h-full p-2! w-full flex flex-col gap-2">
        <Tabs
            v-model="state.activeKey"
            :tabs="tabKeys.map((key) => ({ key, label: key }))"
            class="flex-1 min-h-0"
            content-class="px-[10px] pb-[10px]"
            @close="onRemoveTab"
            @change="onTabChange"
        >
            <template #default="{ tab: pane }">
                <div class="h-full flex flex-col gap-2">
                    <!-- 集合头部：读数 + 集合级入口。唯一刷新入口在这里，数据区内不再重复 -->
                    <div class="coll-header flex items-center justify-between gap-3 flex-wrap">
                        <div class="metrics flex items-center gap-4 flex-wrap text-[13px]">
                            <span class="text-gray-500 font-medium">{{ pane.key }}</span>
                            <span v-for="metric in headerMetrics" :key="metric.label" class="metric">
                                <span class="text-gray-500">{{ metric.label }}</span>
                                <span class="ml-1 font-medium">{{ metric.value }}</span>
                            </span>
                            <el-tooltip v-if="tab.statsError" :content="tab.statsError" placement="top">
                                <span class="text-yellow-600">{{ $t('mongo.statsUnavailable') }}</span>
                            </el-tooltip>
                            <el-tag v-if="tab.readOnly" size="small" type="info">{{ $t('mongo.readonlyCollection') }}</el-tag>
                        </div>
                        <div class="flex items-center gap-2 flex-wrap">
                            <el-button size="small" icon="data-analysis" @click="aggVisible = true">{{ $t('mongo.aggregate') }}</el-button>
                            <el-button v-auth="perms.dataSave" size="small" icon="edit" :disabled="tab.readOnly" @click="openBatch('update')">
                                {{ $t('mongo.batch') }}
                            </el-button>
                            <el-button v-auth="perms.dataSave" size="small" icon="upload" :disabled="tab.readOnly" @click="importVisible = true">
                                {{ $t('mongo.importData') }}
                            </el-button>
                            <el-button v-auth="perms.dataExport" size="small" icon="download" @click="exportVisible = true">
                                {{ $t('mongo.exportData') }}
                            </el-button>
                            <el-divider direction="vertical" />
                            <el-radio-group v-model="viewMode" size="small">
                                <el-radio-button value="table" :aria-label="$t('mongo.tableView')">
                                    <el-tooltip :content="$t('mongo.tableView')" placement="top"
                                        ><el-icon><Grid /></el-icon
                                    ></el-tooltip>
                                </el-radio-button>
                                <el-radio-button value="card" :aria-label="$t('mongo.cardView')">
                                    <el-tooltip :content="$t('mongo.cardView')" placement="top"
                                        ><el-icon><Document /></el-icon
                                    ></el-tooltip>
                                </el-radio-button>
                            </el-radio-group>
                            <el-button size="small" icon="operation" @click="metaVisible = true">{{ $t('mongo.collMeta') }}</el-button>
                            <el-button size="small" icon="refresh" :loading="tab.loading" @click="refresh">{{ $t('common.refresh') }}</el-button>
                            <el-button
                                v-auth="perms.dataSave"
                                size="small"
                                type="primary"
                                icon="plus"
                                :disabled="!tab.loaded || tab.readOnly"
                                @click="onAddDoc"
                            >
                                {{ $t('mongo.addDoc') }}
                            </el-button>
                        </div>
                    </div>

                    <!-- 条件条：入口一律带文字并显示当前生效值，每条条件单独可清除 -->
                    <div class="condition-bar flex items-center gap-2 flex-wrap">
                        <el-button class="cond-entry" size="small" icon="filter" @click="openConditionDialog">
                            <span class="text-gray-500">{{ $t('mongo.condition') }}</span>
                        </el-button>
                        <span v-for="chip in chips" :key="chip.name" class="cond-chip">
                            <button type="button" class="chip-text" :title="chip.value" @click="openConditionDialog">
                                <span class="text-gray-500">{{ $t(CONDITION_LABEL_KEY[chip.name]) }}</span>
                                <span class="ml-1 font-medium">{{ chip.value }}</span>
                            </button>
                            <button
                                type="button"
                                class="chip-close"
                                :aria-label="$t('mongo.removeCondition')"
                                :title="$t('mongo.removeCondition')"
                                @click="clearOneCondition(chip.name)"
                            >
                                <el-icon><Close /></el-icon>
                            </button>
                        </span>
                        <span v-if="!chips.length" class="text-[13px] text-gray-400">{{ $t('mongo.noCondition') }}</span>

                        <div class="ml-auto flex items-center gap-2">
                            <el-select class="limit-select" :model-value="tab.limit" size="small" @change="onLimitChange">
                                <el-option v-for="size in LIMIT_OPTIONS" :key="size" :label="$t('mongo.perPage', { size })" :value="size" />
                            </el-select>
                            <el-button size="small" :type="tab.withCount ? 'primary' : 'default'" @click="toggleCount">
                                {{ $t('mongo.showTotal') }}
                            </el-button>
                            <el-button
                                v-if="viewMode === 'table'"
                                size="small"
                                :type="selection.enabled.value ? 'primary' : 'default'"
                                @click="selection.toggleMode()"
                            >
                                {{ $t('mongo.pickRows') }}
                            </el-button>
                            <el-button size="small" type="primary" :loading="tab.loading" @click="runQuery()">{{ $t('mongo.query') }}</el-button>
                        </div>
                    </div>

                    <!-- 失败与空结果必须能区分：只显示「没有匹配文档」会让用户以为是条件写错了 -->
                    <div v-if="tab.error" class="state-tip error">
                        <el-icon><WarningFilled /></el-icon>
                        <span class="ml-2">{{ tab.error }}</span>
                    </div>
                    <div v-else-if="!tab.loaded" class="state-tip">
                        <span>{{ $t('mongo.notQueriedYet') }}</span>
                    </div>
                    <div v-else-if="!tab.docs.length" class="state-tip">
                        <span>{{ $t('mongo.noMatchedDoc') }}</span>
                        <el-button v-if="chips.length" link type="primary" @click="clearCondition">{{ $t('mongo.clearCondition') }}</el-button>
                    </div>
                    <doc-table
                        v-else-if="viewMode === 'table'"
                        class="flex-1 min-h-0"
                        :docs="tab.docs"
                        :columns="inference.columns"
                        :optional="inference.optional"
                        :pinned="pinned"
                        :sort="sortDoc"
                        :loading="tab.loading"
                        :selectable="selection.enabled.value && !tab.readOnly"
                        @open="openDetail"
                        @sort="onSortColumn"
                        @pin="onPinColumns"
                        @selection-change="onSelectionChange"
                        @edit="editDoc"
                        @remove="deleteDoc"
                    />
                    <!-- 卡片视图内容长短不一，交给滚动容器；表格自己管滚动 -->
                    <el-scrollbar v-else class="flex-1 min-h-0">
                        <div class="doc-grid">
                            <el-card v-for="(doc, index) in tab.docs" :key="docKey(doc, index)" shadow="hover" :body-style="{ padding: '0' }" class="doc-card">
                                <template #header>
                                    <div class="flex items-center justify-between gap-2">
                                        <div class="flex items-center gap-2 min-w-0">
                                            <el-tag v-if="doc.mode === DOC_MODE_EXT_JSON" size="small" type="warning">BSON</el-tag>
                                            <el-link type="primary" underline="never" class="doc-id" :title="idText(doc)" @click="openDetail(doc)">
                                                {{ idText(doc) || $t('mongo.noId') }}
                                            </el-link>
                                            <span class="text-[12px] text-gray-400">{{ idKindLabel(doc.idKind) }}</span>
                                        </div>
                                        <div class="flex items-center gap-2">
                                            <el-button
                                                v-auth="perms.dataSave"
                                                link
                                                type="primary"
                                                :disabled="!doc.idToken || tab.readOnly"
                                                @click="editDoc(doc)"
                                            >
                                                {{ $t('common.edit') }}
                                            </el-button>
                                            <el-button
                                                v-auth="perms.dataDel"
                                                link
                                                type="danger"
                                                :disabled="!doc.idToken || tab.readOnly"
                                                @click="deleteDoc(doc)"
                                            >
                                                {{ $t('common.delete') }}
                                            </el-button>
                                        </div>
                                    </div>
                                </template>
                                <!-- 只读展示：旧实现用可编辑 textarea 冒充编辑入口，改了不保存等于骗人 -->
                                <pre class="doc-body">{{ prettyDoc(doc.doc) }}</pre>
                            </el-card>
                        </div>
                    </el-scrollbar>

                    <!-- 底部：常态显示读数与分页；进入勾选态后动作条原位顶上，工具条与列表都不重排 -->
                    <div class="footer-bar flex items-center justify-between gap-2">
                        <div class="flex items-center gap-2">
                            <template v-if="showBatchBar">
                                <span class="text-[13px] text-gray-500">{{ $t('mongo.pickedCount', { count: selection.selectedCount.value }) }}</span>
                                <el-button v-auth="perms.dataDel" size="small" type="danger" plain @click="onRemoveSelected">
                                    {{ $t('mongo.deletePicked', { count: selection.selectedCount.value }) }}
                                </el-button>
                                <el-button v-auth="perms.dataExport" size="small" plain @click="onExportPicked">
                                    {{ $t('mongo.exportPicked') }}
                                </el-button>
                            </template>
                            <span v-else class="text-[13px] text-gray-500">{{ listStatusText }}</span>
                        </div>
                        <div class="flex items-center gap-2">
                            <el-button size="small" :disabled="tab.page <= 1 || tab.loading" @click="turnPage(-1)">{{ $t('mongo.prevPage') }}</el-button>
                            <span class="text-[13px]">{{ $t('mongo.pageIndicator', { page: tab.page }) }}</span>
                            <el-button size="small" :disabled="!tab.loaded || tab.loading" @click="turnPage(1)">{{ $t('mongo.nextPage') }}</el-button>
                        </div>
                    </div>
                </div>
            </template>

            <template #empty>
                <span class="text-gray-400">{{ $t('mongo.selectCollectionTip') }}</span>
            </template>
        </Tabs>

        <!-- 条件编辑 -->
        <el-dialog v-model="conditionDialog.visible" :title="$t('mongo.condition')" width="620px" :close-on-click-modal="false">
            <div class="flex flex-col gap-3">
                <div v-for="field in CONDITION_ITEMS" :key="field.key">
                    <div class="text-[13px] text-gray-500 mb-1">{{ field.label }}</div>
                    <el-input v-model="conditionDialog.draft[field.key]" type="textarea" :rows="field.rows" :placeholder="field.placeholder" class="mono" />
                </div>
            </div>
            <template #footer>
                <el-button @click="conditionDialog.visible = false">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" @click="confirmCondition">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>

        <!-- 文档详情：字段清单是排查 Mongo 异构数据的主路径，比整段 JSON 更快定位 -->
        <el-drawer v-model="detail.visible" size="46%" :title="$t('mongo.docDetail')" :close-on-click-modal="false">
            <div v-if="detailDoc" class="flex flex-col gap-3">
                <div class="flex flex-wrap items-center gap-2 text-[13px]">
                    <el-tag size="small" type="info">{{ idKindLabel(detailDoc.idKind) }}</el-tag>
                    <span class="font-mono">{{ idText(detailDoc) || $t('mongo.noId') }}</span>
                    <el-tag v-if="detailDoc.mode === DOC_MODE_EXT_JSON" size="small" type="warning">BSON</el-tag>
                    <el-tooltip v-if="!detailDoc.idToken" :content="$t('mongo.idTokenMissing')" placement="top">
                        <el-tag size="small" type="danger">{{ $t('mongo.noId') }}</el-tag>
                    </el-tooltip>
                </div>
                <el-alert
                    v-if="detailView.typedSummary"
                    :title="$t('mongo.typedFieldsTip', { types: detailView.typedSummary })"
                    type="warning"
                    :closable="false"
                    show-icon
                />

                <el-table :data="detailView.fields" size="small" max-height="360">
                    <el-table-column min-width="150" :label="$t('mongo.field')" prop="path" show-overflow-tooltip>
                        <template #default="{ row }">
                            <span class="font-mono">{{ row.path }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column min-width="80" :label="$t('common.type')" prop="kind" />
                    <el-table-column min-width="200" :label="$t('mongo.value')" show-overflow-tooltip>
                        <template #default="{ row }">
                            <span class="font-mono" :class="{ 'text-yellow-700': row.typed }">{{ row.display }}</span>
                        </template>
                    </el-table-column>
                    <el-table-column min-width="150" :label="$t('common.operation')">
                        <template #default="{ row }">
                            <el-button link type="primary" @click="onCopyValue(row)">{{ $t('common.copy') }}</el-button>
                            <el-button v-if="row.queryPath" link type="primary" @click="onFilterByField(row)">
                                {{ $t('mongo.filterByValue') }}
                            </el-button>
                        </template>
                    </el-table-column>
                </el-table>

                <div class="flex items-center gap-2">
                    <el-button size="small" icon="copy-document" @click="onCopyDoc">{{ $t('mongo.copyDoc') }}</el-button>
                </div>
                <pre class="doc-body raw-doc">{{ detailView.raw }}</pre>
            </div>
            <template #footer>
                <el-button @click="detail.visible = false">{{ $t('common.close') }}</el-button>
                <el-button v-auth="perms.dataDel" type="danger" plain :disabled="!detailDoc?.idToken || tab?.readOnly" @click="onRemoveFromDetail">
                    {{ $t('common.delete') }}
                </el-button>
                <el-button v-auth="perms.dataSave" type="primary" :disabled="!detailDoc?.idToken || tab?.readOnly" @click="onEditFromDetail">
                    {{ $t('common.edit') }}
                </el-button>
            </template>
        </el-drawer>

        <!-- 文档编辑 -->
        <el-dialog
            v-model="editState.visible"
            :title="dialogTitle"
            width="60%"
            :close-on-click-modal="false"
            :before-close="() => closeEdit()"
            class="doc-edit-dialog"
        >
            <div class="flex flex-col gap-2">
                <!-- 类型标注提示：让人知道 $date/$oid 不是冗余装饰，删掉包装即改变字段类型 -->
                <el-alert v-if="typedSummary" :title="$t('mongo.typedFieldsTip', { types: typedSummary })" type="warning" :closable="false" show-icon />
                <el-alert v-if="editState.error" :title="editState.error" type="error" :closable="false" show-icon />
                <monaco-editor v-model="editState.text" language="json" height="420px" />
            </div>
            <template #footer>
                <el-button @click="closeEdit()">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="perms.dataSave" type="primary" :loading="editState.saving" @click="saveDoc">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>

        <meta-drawer v-model:visible="metaVisible" :target="target" @changed="refresh" />
        <agg-dialog v-model:visible="aggVisible" :target="target" @written="refresh" />
        <batch-dialog v-model:visible="batchVisible" :target="target" :filter-text="tab?.filterText ?? '{}'" :mode="batchMode" @done="refresh" />
        <export-dialog
            v-model:visible="exportVisible"
            :target="target"
            :filter-text="exportFilterText"
            :selected-docs="selection.selected.value"
            :default-scope="exportScope"
        />
        <import-dialog v-model:visible="importVisible" :target="target" @success="refresh" />
    </div>
</template>

<script lang="ts" setup>
import { computed, defineAsyncComponent, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Close, Document, Grid, WarningFilled } from '@element-plus/icons-vue';

import { Tabs } from '@/components/tabs';

import { formatByteSize } from '@/common/utils/format';
import { copyToClipboard } from '@/common/utils/string';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import type { MongoOpTabApi } from './index';
import { DOC_MODE_EXT_JSON, collectTypedFields, prettyDoc, typedFieldsSummary } from '../docview/extjson';
import { inferColumns } from '../docview/schema';
import { docKey, flattenFields, idKindLabel, idText, type DocField } from '../docview/fields';
import { activeChips, applyEquality, applySort, clearedText, CONDITION_LABEL_KEY, parseSortDoc, type ConditionName } from '../docview/conditions';
import { mongoApi } from '../api';
import { perms } from '../perms';
import type { CollectionParam, ExportScope, MongoDoc } from '../types';
import { useDocQuery, LIMIT_OPTIONS, TOTAL_UNKNOWN, type DocQueryTab } from './composables/useDocQuery';
import { useDocEdit } from './composables/useDocEdit';
import { useDocSelection } from './composables/useDocSelection';
import type { BatchMode } from './composables/useBatchOps';

const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));
const DocTable = defineAsyncComponent(() => import('../docview/DocTable.vue'));
const MetaDrawer = defineAsyncComponent(() => import('../meta/MetaDrawer.vue'));
const AggDialog = defineAsyncComponent(() => import('../aggregate/AggDialog.vue'));
const BatchDialog = defineAsyncComponent(() => import('../batch/BatchDialog.vue'));
const ExportDialog = defineAsyncComponent(() => import('../io/ExportDialog.vue'));
const ImportDialog = defineAsyncComponent(() => import('../io/ImportDialog.vue'));

const { t } = useI18n();

/** 文档视图模式：表格看批量形状，卡片看单条内容，两者的默认值不该互相迁就 */
type DocViewMode = 'table' | 'card';

/** 每个集合的视图偏好与固定列，随 tab 存续（切回来还是刚才那个看法） */
const viewModes = ref<Record<string, DocViewMode>>({});
const pinnedColumns = ref<Record<string, string[]>>({});

const { state, activeTab, tabKeys, openTab, closeTab, runQuery, goPage, changeLimit, setWithCount, setCondition, locatable } = useDocQuery();

// 拆到局部作域：模板只对顶层引用自动解包，放在对象里就要写 `.value` 才能读到值
const { state: editState, typedSummary, openAdd, openEdit, close: closeEdit, save, remove } = useDocEdit(() => runQuery());

const selection = useDocSelection();

const conditionDialog = ref({
    visible: false,
    draft: { filterText: '', sortText: '', projectionText: '' },
});

const metaVisible = ref(false);
const aggVisible = ref(false);
const batchVisible = ref(false);
const batchMode = ref<BatchMode>('update');
const exportVisible = ref(false);
const exportScope = ref<ExportScope>('condition');
const importVisible = ref(false);

const detail = reactive({
    visible: false,
    /** 打开时那份文档的令牌，用于从最新结果里取同一条 */
    token: '',
    /** 该文档已不在当前结果里（条件变了/被裁页）时的兜底内容 */
    snapshot: null as MongoDoc | null,
});

/** 条件弹窗字段声明：顺序按「先筛后序再取列」的心智排 */
const CONDITION_ITEMS = computed(() => [
    { key: 'filterText' as const, label: `${t('mongo.filter')} (${t('mongo.filterHint')})`, rows: 5, placeholder: '{"status": "paid"}' },
    { key: 'sortText' as const, label: `${t('mongo.sort')} (${t('mongo.sortHint')})`, rows: 3, placeholder: '{"createdAt": -1}' },
    { key: 'projectionText' as const, label: `${t('mongo.projection')} (${t('mongo.projectionHint')})`, rows: 3, placeholder: '{"amount": 1, "_id": 0}' },
]);

/** 当前 tab 的别名，模板里反复取用，空态由 tabKeys 保证不会是 undefined */
const tab = computed<DocQueryTab>(() => activeTab.value as DocQueryTab);

/** 当前集合定位：结构、聚合、批量与进出口面板共用一份，避免各面板各拼一遍路径参数 */
const target = computed<CollectionParam | null>(() => {
    const current = tab.value;
    return current ? { id: current.mongoId, database: current.database, collection: current.collection } : null;
});

const viewMode = computed<DocViewMode>({
    get: () => viewModes.value[state.activeKey] ?? 'table',
    set: (mode) => {
        viewModes.value[state.activeKey] = mode;
    },
});

const pinned = computed<string[]>(() => pinnedColumns.value[state.activeKey] ?? []);

/** 列集合 = 本页文档推断出的形状 + 用户固定的列（覆盖率与列数上限筛掉的进 optional） */
const inference = computed(() => inferColumns(tab.value?.docs ?? [], { pinned: pinned.value }));

const chips = computed(() => {
    const current = tab.value;
    if (!current) {
        return [];
    }
    return activeChips({ filterText: current.filterText, sortText: current.sortText, projectionText: current.projectionText });
});

const sortDoc = computed(() => parseSortDoc(tab.value?.sortText ?? ''));

/** 勾选态动作条：只有真的在勾选且有内容被选中时才顶替读数，平时不占行 */
const showBatchBar = computed(() => selection.enabled.value && selection.selectedCount.value > 0);

/**
 * 抽屉展示的文档。
 *
 * 按令牌从「当前结果」里找，而不是直接用打开时那份引用：列表重查后行是新对象，
 * 用旧引用会让「详情里看到的」与「列表里改的」分叉，从抽屉进编辑器还会预填陈旧内容。
 */
const detailDoc = computed<MongoDoc | null>(() => {
    if (!detail.token) {
        return detail.snapshot;
    }
    return (tab.value?.docs ?? []).find((doc) => docKey(doc) === detail.token) ?? detail.snapshot;
});

/** 详情抽屉的派生内容：字段清单、类型摘要与原文，一次打开算好，切字段不再重复遍历 */
const detailView = computed(() => {
    const doc = detailDoc.value;
    if (!doc) {
        return { fields: [] as DocField[], typedSummary: '', raw: '' };
    }
    return {
        fields: flattenFields(doc.doc),
        typedSummary: typedFieldsSummary(collectTypedFields(doc.doc)),
        raw: prettyDoc(doc.doc),
    };
});

/**
 * 列表读数。
 *
 * 三态必须分开：未统计总数时不能说「共 N 条」，被上限截断时不能完全依靠已加载条数表达「还有更多」，
 * 已统计时才能给出匹配总数。旧实现拿 collStats 的集合总数冒充匹配数，加条件后就是错的。
 */
const listStatusText = computed(() => {
    const current = tab.value;
    if (!current || !current.loaded) {
        return '';
    }
    const loaded = current.docs.length;
    if (current.total !== TOTAL_UNKNOWN) {
        return t('mongo.matchedTotal', { matched: current.total, loaded });
    }
    if (current.truncated) {
        return t('mongo.loadedTruncated', { loaded, limit: current.effectiveLimit });
    }
    return t('mongo.loadedOnly', { loaded });
});

/** 头部读数：取不到的项（为 0 且未查到）直接不显示，不留空占位 */
const headerMetrics = computed(() => {
    const stats = tab.value?.stats;
    if (!stats) {
        return [];
    }
    const metrics: { label: string; value: string }[] = [
        { label: t('mongo.docCount'), value: String(stats.count) },
        { label: t('mongo.avgObjSize'), value: formatByteSize(stats.avgObjSize) },
        { label: t('mongo.storageSize'), value: formatByteSize(stats.storageSize) },
        { label: t('mongo.indexCount'), value: String(stats.nindexes) },
    ];
    // 可回收空间只在真的回收了（>0）时才有决策价值，否则就是噪声
    if (stats.freeStorageSize > 0) {
        metrics.push({ label: t('mongo.freeStorageSize'), value: formatByteSize(stats.freeStorageSize) });
    }
    return metrics;
});

const dialogTitle = computed(() => (editState.intent === 'add' ? t('mongo.addDoc') : t('mongo.editDoc')));

/** 导出默认带的条件：当前查询条件原样交给服务端重取，按选中导出由导出面板自己换成主键集合 */
const exportFilterText = computed(() => tab.value?.filterText ?? '{}');

function refresh() {
    runQuery();
}

function onTabChange() {
    // 切 tab 即清掉上一个集合的勾选：跨集合的选中集合没有意义，且会让动作条打到错的集合上
    selection.clear();
}

function onAddDoc() {
    // 同集合文档形状通常一致，以当前页第一个文档为模板比从空白写更省事
    openAdd(tab.value?.docs[0]);
}

function editDoc(doc: MongoDoc) {
    openEdit(doc);
}

async function deleteDoc(doc: MongoDoc) {
    const current = tab.value;
    if (current) {
        await remove(current, doc);
    }
}

async function saveDoc() {
    const current = tab.value;
    if (current) {
        await save(current);
    }
}

function turnPage(delta: number) {
    goPage(delta);
    runQuery();
}

async function onLimitChange(limit: number) {
    changeLimit(limit);
    await runQuery();
}

async function toggleCount() {
    setWithCount(!tab.value.withCount);
    await runQuery();
}

function openConditionDialog() {
    const current = tab.value;
    if (!current) {
        return;
    }
    conditionDialog.value.draft = { filterText: current.filterText, sortText: current.sortText, projectionText: current.projectionText };
    conditionDialog.value.visible = true;
}

async function confirmCondition() {
    const current = tab.value;
    if (!current) {
        return;
    }
    const { filterText, sortText, projectionText } = conditionDialog.value.draft;
    conditionDialog.value.visible = false;
    // 条件变了还停在第 3 页，新条件下很可能根本没有那么多条，回到第一页才符合预期
    setCondition({ filterText, sortText, projectionText });
    await runQuery();
}

/** 单独清掉某一条条件：整条件弹窗对「只想去掉排序」这种诉求来说太重 */
async function clearOneCondition(name: ConditionName) {
    const current = tab.value;
    if (!current) {
        return;
    }
    const text = clearedText(name);
    setCondition(name === 'filter' ? { filterText: text } : name === 'sort' ? { sortText: text } : { projectionText: text });
    await runQuery();
}

async function clearCondition() {
    const current = tab.value;
    if (!current) {
        return;
    }
    setCondition({ filterText: '{}', sortText: '', projectionText: '' });
    await runQuery();
}

/** 点列头切换排序：三态循环（升 → 降 → 取消）后直接重查，排序改动必然要换数据 */
async function onSortColumn(field: string) {
    const current = tab.value;
    if (!current) {
        return;
    }
    const next = applySort(current.sortText, field);
    setCondition({ sortText: next });
    await runQuery();
}

/** 固定列只影响显示，不影响取数：改列集不该重发请求 */
function onPinColumns(keys: string[]) {
    pinnedColumns.value[state.activeKey] = keys;
}

// 换页/重查后按新结果裁剪勾选：留着不可见的行会让「删除选中」打到看不见的文档上
watch(
    () => tab.value?.docs,
    (docs) => selection.prune(docs ?? [])
);

/** 表格勾选直接接管：投影排除 _id 的行没有令牌，勾选它们会让批量删除打空，因此先剔掉 */
function onSelectionChange(docs: MongoDoc[]) {
    selection.setSelected(docs.filter((doc) => locatable(doc)));
}

function openDetail(doc: MongoDoc) {
    detail.token = doc.idToken || docKey(doc);
    detail.snapshot = doc;
    detail.visible = true;
}

async function onFilterByField(field: DocField) {
    const current = tab.value;
    if (!current) {
        return;
    }
    // 用 queryPath 而不是展示路径：数组子项要问的是「数组里有没有这个值」，
    // 把 `tags[0]` 写进条件是合法的 JSON 却永远查不出文档，那比报错更坏
    setCondition({ filterText: applyEquality(current.filterText, field.queryPath, field.value) });
    detail.visible = false;
    await runQuery();
}

async function onCopyValue(field: DocField) {
    // 复制成功提示由 copyToClipboard 统一发出，这里再叠一条会出现两个一模一样的 toast
    await copyToClipboard(typeof field.value === 'string' ? field.value : JSON.stringify(field.value, null, 2));
}

async function onCopyDoc() {
    await copyToClipboard(detailView.value.raw);
}

function onEditFromDetail() {
    const doc = detailDoc.value;
    if (!doc) {
        return;
    }
    detail.visible = false;
    editDoc(doc);
}

async function onRemoveFromDetail() {
    const doc = detailDoc.value;
    if (!doc) {
        return;
    }
    detail.visible = false;
    await deleteDoc(doc);
}

function openBatch(mode: BatchMode) {
    batchMode.value = mode;
    batchVisible.value = true;
}

function onExportPicked() {
    exportScope.value = 'selected';
    exportVisible.value = true;
}

/** 删除勾选的文档：走主键令牌而不是条件，勾了哪几条就删哪几条 */
async function onRemoveSelected() {
    const current = tab.value;
    const docs = selection.selected.value;
    if (!current || !docs.length) {
        return;
    }
    if (!(await useI18nDeleteConfirm(`${current.database}.${current.collection} (${docs.length})`))) {
        // 取消或关掉弹窗：不继续后续操作
        return;
    }

    const idTokens = docs.map((doc) => doc.idToken).filter(Boolean);
    const res = await mongoApi.deleteDocs.request({
        id: current.mongoId,
        database: current.database,
        collection: current.collection,
        idTokens,
    });
    // deletedCount 可能小于请求条数（他人已先删掉部分文档），如实说明而不是假装成功
    if (res.deletedCount < idTokens.length) {
        Msg.warning('mongo.deletedPart', { deleted: res.deletedCount, requested: idTokens.length });
    } else {
        Msg.deleteSuccess();
    }
    selection.clear();
    await runQuery();
}

function onRemoveTab(key: string | number) {
    closeTab(String(key));
    const tabKey = String(key);
    delete viewModes.value[tabKey];
    delete pinnedColumns.value[tabKey];
}

async function changeCollection(id: number, database: string, collection: string, readOnly = false) {
    openTab(id, database, collection, readOnly);
    selection.clear();
    // 首次打开该集合才需要拉数据；已查过的 tab 保留原来的条件与结果，切回来不该被重置
    const current = state.tabs[`${database}.${collection}`];
    if (current && !current.loaded) {
        await runQuery(`${database}.${collection}`);
    }
}

defineExpose({
    changeCollection,
    onRefresh: () => runQuery(),
} satisfies MongoOpTabApi);
</script>

<style lang="scss" scoped>
.mongo-data-op {
    .coll-header {
        .metric {
            white-space: nowrap;
        }
    }

    .condition-bar {
        .cond-entry {
            flex-shrink: 0;
        }

        .cond-chip {
            display: inline-flex;
            align-items: center;
            max-width: 320px;
            border: 1px solid var(--el-border-color-lighter);
            border-radius: 999px;
            background: var(--el-fill-color-light);
            font-size: 12px;
            line-height: 22px;

            .chip-text {
                min-width: 0;
                max-width: 270px;
                padding: 0 4px 0 10px;
                overflow: hidden;
                border: 0;
                background: none;
                color: inherit;
                font: inherit;
                text-align: left;
                text-overflow: ellipsis;
                white-space: nowrap;
                cursor: pointer;

                &:hover {
                    color: var(--el-color-primary);
                }
            }

            .chip-close {
                display: inline-flex;
                align-items: center;
                padding: 0 8px 0 2px;
                border: 0;
                background: none;
                color: var(--el-text-color-secondary);
                cursor: pointer;

                &:hover {
                    color: var(--el-color-danger);
                }
            }
        }

        .limit-select {
            width: 118px;
        }
    }

    .state-tip {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 28px 12px;
        color: var(--el-text-color-secondary);

        &.error {
            color: var(--el-color-danger);
            justify-content: flex-start;
        }
    }

    .doc-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
        gap: 10px;
    }

    .doc-card {
        :deep(.el-card__header) {
            padding: 8px 12px;
        }
    }

    .doc-id {
        min-width: 0;
        overflow: hidden;
        font-family: var(--el-font-family-mono, monospace);
        font-size: 12px;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .doc-body {
        margin: 0;
        padding: 10px 12px;
        max-height: 260px;
        overflow: auto;
        font-family: var(--el-font-family-mono, monospace);
        font-size: 12px;
        line-height: 1.6;
        background: var(--el-fill-color-light);
    }

    .raw-doc {
        border-radius: 4px;
        max-height: 200px;
    }

    .footer-bar {
        padding-top: 6px;
        border-top: 1px solid var(--el-border-color-lighter);
    }

    .mono :deep(.el-textarea__inner) {
        font-family: var(--el-font-family-mono, monospace);
        font-size: 12px;
        line-height: 1.7;
    }
}
</style>
