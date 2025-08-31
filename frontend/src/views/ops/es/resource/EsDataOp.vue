<template>
    <div class="es-op-box h-full">
        <div id="es-op-tabs" class="mt-1">
            <el-tabs @tab-remove="removeDataTab" type="card" class="!h-full w-full" v-model="state.activeName">
                <template v-for="dt in state.dataTabs" :key="dt.key">
                    <el-tab-pane closable class="!h-full" v-if="dt.tabType === EsNodeType.Index" :label="dt.label" :name="dt.key">
                        <el-descriptions class="!w-full" :column="20" size="small" border>
                            <el-descriptions-item label-align="center">
                                <template #label>
                                    <SvgIcon name="PieChart" />
                                </template>
                                {{ dt.params.size }}
                            </el-descriptions-item>
                            <el-descriptions-item label-align="center">
                                <template #label>
                                    <el-space><SvgIcon name="refresh" @click="onRefreshStats(dt)" /> {{ t('es.docs') }}</el-space>
                                </template>
                                {{ dt.params.idx['docs.count'] }}
                            </el-descriptions-item>
                            <el-descriptions-item :label="t('es.health')" label-align="center">
                                <el-tag size="small" :type="getHealthTagType(dt.params.idx.health)">{{ dt.params.idx.health }}</el-tag>
                            </el-descriptions-item>
                            <el-descriptions-item :label="t('es.status')" label-align="center">
                                <el-tag size="small" :type="dt.params.idx.status === 'open' ? 'success' : 'danger'">{{ dt.params.idx.status }}</el-tag>
                            </el-descriptions-item>
                        </el-descriptions>
                        <el-row class="es-op-header">
                            <el-col :span="20">
                                <el-space>
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('common.refresh')">
                                        <el-link @click="onRefreshData(dt)" icon="refresh" underline="never" />
                                    </el-tooltip>

                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.opSearch')">
                                        <el-link @click="onBasicSearch(dt)" icon="Search" underline="never" />
                                    </el-tooltip>

                                    <el-tooltip :show-after="tooltipTime" v-auth="perms.saveData" effect="dark" placement="top" :content="t('common.create')">
                                        <el-link @click="onAddDoc(dt)" icon="plus" underline="never" />
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" v-auth="perms.delData" effect="dark" placement="top" :content="t('common.delete')">
                                        <el-link :disabled="dt.selectKeys.length === 0" @click="onDeleteDocs(dt)" icon="Minus" underline="never" />
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" v-auth="perms.saveData" effect="dark" placement="top" :content="t('common.edit')">
                                        <el-link :disabled="dt.selectKeys.length !== 1" @click="onEditSelectDoc(dt)" icon="EditPen" underline="never" />
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.indexDetail')">
                                        <el-link @click="onIndexDetail(dt)" icon="InfoFilled" underline="never" />
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.page.home')">
                                        <el-link :disabled="dt.search.from === 0" @click="onFirstPage(dt)" icon="DArrowLeft" underline="never" />
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.page.prev')">
                                        <el-link :disabled="dt.search.from === 0" @click="onPrevPage(dt)" icon="ArrowLeft" underline="never" />
                                    </el-tooltip>

                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.page.changeSize')">
                                        <el-dropdown placement="bottom" size="small">
                                            <el-link underline="never" :style="{ fontSize: '12px' }">
                                                {{ dt.currentFrom + 1 }} - {{ Math.min(dt.currentFrom + dt.search.size, dt.total) }}</el-link
                                            >
                                            <template #dropdown>
                                                <el-dropdown-menu>
                                                    <el-dropdown-item @click="onChangePageSize(25, dt)">25</el-dropdown-item>
                                                    <el-dropdown-item @click="onChangePageSize(50, dt)">50</el-dropdown-item>
                                                    <el-dropdown-item @click="onChangePageSize(100, dt)">100</el-dropdown-item>
                                                    <el-dropdown-item @click="onChangePageSize(200, dt)">200</el-dropdown-item>
                                                    <el-dropdown-item @click="onChangePageSize(1000, dt)">1000</el-dropdown-item>
                                                </el-dropdown-menu>
                                            </template>
                                        </el-dropdown>
                                    </el-tooltip>
                                    /
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.page.total')">
                                        <el-link
                                            underline="never"
                                            @click="onSwitchTrackTotal(dt)"
                                            :type="dt.search.track_total_hits === true ? 'success' : 'info'"
                                            :style="{ fontSize: '12px' }"
                                        >
                                            {{ dt.searchRes.hits?.total?.value || 0 }}</el-link
                                        >
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.page.next')">
                                        <el-link
                                            :disabled="dt.search.from + dt.search.size >= (dt.total || 0)"
                                            @click="onNextPage(dt)"
                                            icon="ArrowRight"
                                            underline="never"
                                        />
                                    </el-tooltip>

                                    <!-- <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('common.download')">-->
                                    <!--     <el-link @click="onDownload(dt)" icon="Download" underline="never" />-->
                                    <!-- </el-tooltip>-->
                                    <!-- <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('common.upload')">-->
                                    <!--     <el-link @click="onUpload(dt)" icon="Upload" underline="never" />-->
                                    <!-- </el-tooltip>-->
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.Reindex')">
                                        <el-link @click="onReindex(dt.params.instId, dt.idxName)" icon="Switch" underline="never" />
                                    </el-tooltip>
                                    <el-tooltip :show-after="tooltipTime" effect="dark" placement="top" :content="t('es.opViewColumns')">
                                        <el-dropdown placement="bottom" size="small" :max-height="300" :hide-on-click="false" trigger="click">
                                            <el-link icon="Operation" underline="never" />
                                            <template #dropdown>
                                                <el-dropdown-menu class="dropdown-menu">
                                                    <el-dropdown-item>
                                                        <el-space>
                                                            <el-checkbox @change="onCheckAllColumns(dt)" v-model="dt.checkAllColumns" />
                                                            <el-input
                                                                v-model="dt.columnsFilterText"
                                                                @input="onFilterColumns(dt)"
                                                                :placeholder="t('es.filterColumn')"
                                                                clearable
                                                                size="small"
                                                            />
                                                        </el-space>
                                                    </el-dropdown-item>
                                                    <template v-for="column in dt.columns" :key="column.key">
                                                        <el-dropdown-item v-if="column._filterd" :command="column.key">
                                                            <el-checkbox v-model="column._show" @change="onCheckColumnFilter(column)">
                                                                {{ column.title }}
                                                            </el-checkbox>
                                                        </el-dropdown-item>
                                                    </template>
                                                </el-dropdown-menu>
                                            </template>
                                        </el-dropdown>
                                    </el-tooltip>
                                </el-space>
                            </el-col>
                        </el-row>
                        <div class="es-table-data mt-1" style="height: calc(-208px + 100vh)">
                            <el-auto-resizer>
                                <template #default="{ height, width }">
                                    <el-table-v2
                                        ref="tableRef"
                                        :row-height="dt.rowHeight"
                                        :columns="dt.columns"
                                        :data="dt.datas"
                                        :width="width"
                                        :height="height"
                                        fixed
                                        :header-height="22"
                                        class="es-table"
                                    >
                                        <template #header="{ columns }">
                                            <div
                                                v-for="(column, i) in columns"
                                                :key="i"
                                                :style="{
                                                    width: `${column.width}px`,
                                                    textAlign: 'center',
                                                    borderRight: 'var(--el-table-border)',
                                                }"
                                            >
                                                <el-checkbox
                                                    :style="{ height: '100%' }"
                                                    v-if="column.key === '_selected'"
                                                    v-model="dt.selectAll"
                                                    @change="onSelectAll(dt)"
                                                    :indeterminate="dt.selectKeys.length > 0 && !dt.selectAll"
                                                />
                                                <b v-else> {{ column.title }} </b>
                                            </div>
                                        </template>

                                        <template #cell="{ rowData, column, rowIndex, columnIndex }">
                                            <div v-if="column.key === '_table_index'" class="table-data-cell">
                                                <span class="el-text el-text--small is-truncated">
                                                    {{ rowIndex + 1 + dt.currentFrom }}
                                                </span>
                                            </div>
                                            <div v-if="column.key === '_selected'" class="table-data-cell">
                                                <span class="el-text el-text--small is-truncated">
                                                    <el-checkbox v-model="rowData._selected" @change="onSelectRow(dt, rowData)" />
                                                </span>
                                            </div>
                                            <div v-else @contextmenu="dataContextmenuClick($event, rowIndex, column, rowData)" class="table-data-cell">
                                                <span v-if="rowData[column.dataKey] === null" style="color: var(--el-color-info-light-5)"> NULL </span>
                                                <span v-else :title="rowData[column.dataKey]" class="el-text el-text--small is-truncated">
                                                    {{ rowData[column.dataKey] }}
                                                </span>
                                            </div>
                                        </template>

                                        <template v-if="dt.loading" #overlay>
                                            <div class="el-loading-mask flex flex-col items-center justify-center">
                                                <div>
                                                    <SvgIcon class="is-loading" name="loading" color="var(--el-color-primary)" :size="28" />
                                                    <el-text class="ml-1" tag="b">{{ t('db.execTime') }} - {{ dt.execTime?.toFixed(1) || 0 }}s</el-text>
                                                </div>
                                                <div v-if="dt.loading && dt.abortSearch" class="!mt-2">
                                                    <el-button @click="dt.abortSearch" type="info" size="small" plain>{{ t('common.cancel') }}</el-button>
                                                </div>
                                            </div>
                                        </template>
                                    </el-table-v2>
                                </template>
                            </el-auto-resizer>
                        </div>

                        <es-search
                            :instId="dt.instId"
                            :idxName="dt.idxName"
                            :fields="dt.fields"
                            v-model:visible="dt.searchDialogVisible"
                            @search="onEsSearch"
                        />
                    </el-tab-pane>
                    <el-tab-pane closable class="!h-full" v-if="dt.tabType === EsNodeType.Dashboard" :label="dt.label" :name="dt.key">
                        <es-dashboard :inst-id="dt.instId" />
                    </el-tab-pane>
                </template>
            </el-tabs>
        </div>

        <es-edit-row v-model="docEditDialog" v-model:visible="docEditDialog.visible" @success="onEditRowSuccess" />

        <es-index-detail ref="esIndexDetailRef" />

        <es-add-index
            v-model:visible="addIndexDialog.visible"
            :instId="addIndexDialog.instId"
            :idxNames="addIndexDialog.idxNames"
            @success="onAddIndexSuccess"
        />

        <es-reindex
            :instId="reIndexDialog.instId"
            :idxName="reIndexDialog.idxName"
            :idxNames="reIndexDialog.idxNames"
            v-model:visible="reIndexDialog.visible"
            @success="onReIndexSuccess"
        />

        <es-index-template :inst-id="templateDialog.instId" :version="templateDialog.version" v-model="templateDialog.visible" />
    </div>
</template>

<script lang="tsx" setup>
import { defineAsyncComponent, inject, reactive, ref, toRefs, getCurrentInstance, onMounted } from 'vue';

import { ContextmenuItem } from '@/components/contextmenu';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/svgIcon/index.vue';
import { copyToClipboard } from '@/common/utils/string';
import { ElCheckbox, ElMessage } from 'element-plus';
import { useI18nConfirm, useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { useIntervalFn } from '@vueuse/core';
import Api from '@/common/Api';
import { esApi } from '@/views/ops/es/api';
import { ResourceOpCtx, TagTreeNode } from '@/views/ops/component/tag';
import { formatDocSize } from '@/common/utils/format';
import { ResourceOpCtxKey } from '@/views/ops/resource/resource';
import { EsOpComp } from '@/views/ops/es/resource';

const EsAddIndex = defineAsyncComponent(() => import('../component/EsAddIndex.vue'));
const EsDashboard = defineAsyncComponent(() => import('../component/EsDashboard.vue'));
const EsEditRow = defineAsyncComponent(() => import('../component/EsEditRow.vue'));
const EsIndexDetail = defineAsyncComponent(() => import('../component/EsIndexDetail.vue'));
const EsIndexTemplate = defineAsyncComponent(() => import('../component/EsIndexTemplate.vue'));
const EsReindex = defineAsyncComponent(() => import('../component/EsReindex.vue'));
const EsSearch = defineAsyncComponent(() => import('../component/EsSearch.vue'));

const emits = defineEmits(['init']);

const { t } = useI18n();

const perms = {
    saveData: 'es:data:save',
    delData: 'es:data:del',
};

const resourceOpCtx: ResourceOpCtx | undefined = inject(ResourceOpCtxKey);

const getDefaultSearch = () => ({ sort: {}, query: { bool: { must: [], should: [], must_not: [] } }, aggs: {}, from: 0, size: 25 });

const tooltipTime = 300;

const state = reactive({
    defaultExpendKey: [] as any,
    tags: [],
    mongoList: [] as any,
    activeName: '', // 当前操作的tab
    dataTabs: {} as any, // 数据tabs
    reloadStatus: false,
    showSysIndex: false,
    contextmenu: { items: [] as any[], dropdown: { x: 0, y: 0 } },
    docEditDialog: {
        instId: '',
        isAdd: true,
        loading: false,
        visible: false,
        doc: '',
        idxName: '',
        _id: '',
    },
    addIndexDialog: {
        instId: '' as any,
        visible: false,
        data: {} as any,
        idxNames: [] as string[],
    },
    reIndexDialog: {
        instId: '' as any,
        idxName: '' as any,
        visible: false,
        idxNames: [] as string[],
    },
    templateDialog: {
        instId: '' as any,
        version: '',
        visible: false,
    },
});

const { docEditDialog, addIndexDialog, reIndexDialog, templateDialog } = toRefs(state);
/**
 * 树节点类型
 */
class EsNodeType {
    static Inst = 1; // 实例
    static Indexs = 2; // 索引管理
    static Index = 21; // 索引管理
    static BasicSearch = 3; // 基础搜索
    static SeniorSearch = 4; // 高级搜索
    static Dashboard = 5; // 数据看板
    static Settings = 6; // 基本设置
    static Templates = 7; // 模板管理
}

const contextmenuRef = ref();
const esTabDataRef = ref();
const esIndexDetailRef = ref();
const tableRef = ref();
const instIndicesMap = new Map();

onMounted(() => {
    emits('init', { name: EsOpComp.name, ref: getCurrentInstance()?.exposed });
});

const getIndicesByInstId = async (instId: any) => {
    if (!instIndicesMap.has(instId)) {
        await refreshIndices(instId);
    }
    return instIndicesMap.get(instId);
};

const refreshIndices = async (instId: any) => {
    let indicesRes = await esApi.proxyReq('get', instId, `/_cat/indices/?h=index,health,status,uuid,pri,rep,docs.count,docs.deleted,store.size,sc,cd`);
    instIndicesMap.set(instId, indicesRes);
};

const onRefreshIndices = async (instId: any, key: any) => {
    await refreshIndices(instId);
    reloadNode(key);
};

const onAddIndex = async (data: any) => {
    // 弹出新增/修改索引窗口
    state.addIndexDialog.data = data;
    state.addIndexDialog.instId = data.params.instId;
    state.addIndexDialog.visible = true;
    let indices = await getIndicesByInstId(data.params.instId);
    state.addIndexDialog.idxNames = indices
        .map((x: any) => x.index)
        .filter((x: any) => x.indexOf('.') < 0)
        .sort();
};

const onAddIndexSuccess = async () => {
    await onRefreshIndices(state.addIndexDialog.data.params.instId, state.addIndexDialog.data.key);
};

const onReIndexSuccess = () => {
    console.log('onReIndexSuccess');
};
const onIdxCopyName = async (data: any) => {
    await copyToClipboard(data.params.idxName);
};
const onRefreshIdx = async (data: any) => {
    await esApi.proxyReq('post', data.params.instId, `/${data.params.idxName}/_refresh`);
    useI18nOperateSuccessMsg();
};
const onClearIdxCache = async (data: any) => {
    await useI18nConfirm('es.clearCacheConfirm', { name: data.params.idxName });
    await esApi.proxyReq('post', data.params.instId, `/${data.params.idxName}/_cache/clear`);
    useI18nOperateSuccessMsg();
};
const onFlushIdx = async (data: any) => {
    await esApi.proxyReq('post', data.params.instId, `/${data.params.idxName}/_flush`);
    useI18nOperateSuccessMsg();
};
const onIdxReindex = async (data: any) => {
    await onReindex(data.params.instId, data.params.idxName);
};
const onIdxClose = async (data: any) => {
    await useI18nConfirm('es.closeIndexConfirm', { name: data.params.idxName });
    await esApi.proxyReq('post', data.params.instId, `/${data.params.idxName}/_close`);
    data.params.idx.status = 'close';
    useI18nOperateSuccessMsg();
};
const onIdxOpen = async (data: any) => {
    await useI18nConfirm('es.openIndexConfirm', { name: data.params.idxName });
    await esApi.proxyReq('post', data.params.instId, `/${data.params.idxName}/_open`);
    data.params.idx.status = 'open';
    useI18nOperateSuccessMsg();
};
const onIdxDelete = async (data: any) => {
    await useI18nDeleteConfirm(data.params.idxName);
    await esApi.proxyReq('delete', data.params.instId, data.params.idxName);
    useI18nDeleteSuccessMsg();
    await onRefreshIndices(data.params.instId, data.params.parentKey);
};
const onIdxBaseSearch = async (data: any) => {
    // 加载表数据
    let params = data.params;
    await loadIndexData(params.params.inst.id, params);
    // 弹出搜索窗口
    await onBasicSearch(getNowDataTab());
};

const loadIndexData = async (instId: any, params: any) => {
    let idxName = params.idxName;
    const label = `es.${instId}.index.${idxName}`;
    state.activeName = label;

    let dataTab = state.dataTabs[label];
    if (dataTab) {
        return;
    }
    dataTab = {
        tabType: EsNodeType.Index,
        params: params,
        key: label,
        label: idxName,
        name: label,
        instId,
        idxName,
        columns: [],
        fields: [],
        datas: [],
        total: [],
        searchRes: {} as any,
        rowHeight: 30,
        selectAll: false,
        selectKeys: [],
        columnsFilterText: '',
        checkAllColumns: true,
        loading: true,
        abortSearch: () => {},
        execTime: 0,
        currentFrom: 25,
        search: getDefaultSearch(),
        searchDialogVisible: false,
    };
    state.dataTabs[label] = dataTab;
    // 延时加载数据，避免卡顿
    setTimeout(fetchIndexData, 300);
};

// 选中的数据， key->rowIndex  value->primaryKeyValue
const selectionRowsMap = ref(new Map<number, any>());
/**
 * 判断当前行是否被选中
 * @param rowIndex
 */
const isSelection = (rowIndex: number): boolean => {
    return selectionRowsMap.value.get(rowIndex);
};

/**
 * 选中指定行
 * @param rowIndex
 * @param rowData
 * @param isMultiple 是否允许多选
 */
const selectionRow = (rowIndex: number, rowData: any, isMultiple = false) => {
    if (isMultiple) {
        // 如果重复点击，则取消改选中数据
        if (selectionRowsMap.value.get(rowIndex)) {
            selectionRowsMap.value.delete(rowIndex);
            return;
        }
    } else {
        selectionRowsMap.value.clear();
    }
    selectionRowsMap.value.set(rowIndex, rowData);
};

const copyCell = new ContextmenuItem('copyCell', 'common.copyCell').withIcon('CopyDocument').withOnClick(async (data: any) => {
    await copyToClipboard(data.rowData[data.column.dataKey]);
});

const copyLineJson = new ContextmenuItem('copyLineJson', 'es.contextmenu.index.copyLineJson').withIcon('CopyDocument').withOnClick(async (data: any) => {
    await copyToClipboard(data.rowData.src);
});

const copySelectLineJson = new ContextmenuItem('copySelectLineJson', 'es.contextmenu.index.copySelectLineJson')
    .withIcon('CopyDocument')
    .withHideFunc(() => {
        let dt = getNowDataTab();
        if (dt) {
            return dt.selectKeys?.length == 0;
        }
        return true;
    })
    .withOnClick(async () => {
        await copyToClipboard(
            JSON.stringify(
                getNowDataTab().selectKeys.map((a: any) => JSON.parse(a.src)),
                null,
                2
            )
        );
    });

const editLineJson = new ContextmenuItem('editLineJson', 'common.edit').withIcon('EditPen').withOnClick(async (data: any) => onEditDoc(data.rowData.src));

const deleteLine = new ContextmenuItem('deleteLine', 'common.delete').withIcon('Delete').withOnClick(async (data: any) => {
    // 二次确认后执行删除
    let ids = [data.rowData._id];
    await useI18nDeleteConfirm(ids.join(', '));
    await doDeleteDoc(ids);
});
const deleteSelectLine = new ContextmenuItem('deleteLine', 'es.contextmenu.index.DeleteSelectLine')
    .withIcon('Delete')
    .withHideFunc(() => {
        return getNowDataTab().selectKeys.length == 0;
    })
    .withOnClick(async () => {
        // 二次确认后执行删除
        let ids = getNowDataTab().selectKeys.map((a: any) => a._id);
        await useI18nDeleteConfirm(ids.join(', '));
        await doDeleteDoc(ids);
    });

const dataContextmenuClick = (event: any, rowIndex: number, column: any, data: any) => {
    event.preventDefault(); // 阻止默认的右击菜单行为
    // 当前行未选中，则单行选中该行
    if (!isSelection(rowIndex)) {
        selectionRow(rowIndex, data);
    }
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [copyCell, copyLineJson, copySelectLineJson, editLineJson, deleteLine, deleteSelectLine];
    contextmenuRef.value.openContextmenu({ column, rowData: data });
};

const reloadNode = async (nodeKey: string) => {
    state.reloadStatus = true;
    resourceOpCtx?.reloadTreeNode(nodeKey);
};

const onShowSysIndex = (data: any) => {
    state.showSysIndex = true;
    reloadNode(data.key);
};
const onShowTemplate = async (data: any) => {
    state.templateDialog.visible = true;
    state.templateDialog.version = data.params.inst.version;
    state.templateDialog.instId = data.params.instId;
    console.log(state.templateDialog);
};

const onChangePageSize = async (size: number, dt: any) => {
    dt.search.size = size;
    dt.search.from = 0;
    await fetchIndexData(); // 改变每页条数
};

const onFirstPage = async (dt: any) => {
    dt.search.from = 0;
    await fetchIndexData(); // 回到首页
};
const onNextPage = async (dt: any) => {
    dt.search.from = dt.search.from + dt.search.size;
    await fetchIndexData(); // 下一页
};

const onPrevPage = async (dt: any) => {
    dt.search.from = Math.max(0, dt.search.from - dt.search.size);
    await fetchIndexData(); // 上一页
};

const refreshIndex = async () => {
    const dataTab = getNowDataTab();
    await esApi.proxyReq('post', dataTab.instId, `/${dataTab.idxName}/_refresh`);
};

const onEditRowSuccess = async () => {
    await refreshIndex();
    await fetchIndexData(); // 编辑完成后刷新索引数据
};

// 获取索引数据
const fetchIndexData = async () => {
    const dt = getNowDataTab();

    dt.execTime = 0;
    // loading
    const { pause, resume } = useIntervalFn(() => {
        dt.execTime += 0.1;
    }, 100);
    resume();
    dt.loading = true;

    // 清空选中
    dt.selectAll = false;
    dt.selectKeys = [];

    let api = Api.newPost(`/es/instance/proxy/${dt.instId}/${dt.idxName}/_search`);

    const { execute: execSearch, data: searchRes, abort: abortSearch } = api.useApi<any>(dt.search, { esProxyReq: true });
    dt.abortSearch = () => {
        abortSearch();
        dt.loading = false;
        pause();
    };
    // 获取数据
    await execSearch();
    dt.searchRes = searchRes;
    let error = searchRes.value.error || (searchRes.value.failures && searchRes.value.failures.length > 0 && searchRes.value.failures[0]);
    if (error) {
        dt.loading = false;
        pause();
        return await esApi.alertError(error, t('es.execError'));
    }

    // 搜集字段信息
    let fieldMap = {} as any;
    fieldMap['_id'] = { width: 50 };

    // 处理数据
    dt.datas = dt.searchRes.hits.hits.map((a: any) => {
        let src = JSON.parse(JSON.stringify(a._source));
        src._id = a._id;
        let source = a._source;
        source._id = a._id;
        source._score = a._score;
        fieldMap['_score'] = { width: 40 };
        // 如果字段不是字符串或数字，则转换为json字符串
        for (let k in source) {
            if (typeof source[k] != 'string' && typeof source[k] != 'number' && source[k] !== null && typeof source[k] != 'boolean') {
                source[k] = JSON.stringify(source[k]);
            }
            // 表格长度为值长度的十倍px，最小50px，最大220px
            let column = fieldMap[k] || { width: 50 };
            try {
                let valLength = source[k] ? Math.max(source[k].length, k.length) : k.length;
                column.width = Math.max(Math.max(Math.min(220, (valLength || 10) * 10), 50), column.width);
            } catch (e) {
                console.log(e);
                column.width = 50;
            }
            fieldMap[k] = column;
        }
        // 缓存原始数据
        source.src = JSON.stringify(src, null, 2);
        source._selected = false;
        return source;
    });

    dt.total = dt.searchRes.hits?.total.value || 0;

    if (dt.datas.length > 0) {
        let keys = Object.keys(fieldMap).sort();
        dt.fields = keys.filter((k) => k != '_score');
        dt.columns = keys.map((k) => ({ title: k, width: fieldMap[k].width, key: k, dataKey: k, class: 'es-table-column', _filterd: true, _show: true }));
        dt.columns.unshift({
            title: '#',
            width: 50,
            key: '_table_index',
            class: 'es-table-column',
            align: 'center',
            _filterd: false,
        });

        dt.columns.unshift({
            title: 'checkbox',
            width: 30,
            key: '_selected',
            class: 'es-table-column',
            align: 'center',
            _filterd: false,
        });
    }
    pause();
    dt.loading = false;
    dt.currentFrom = dt.search.from;
};

const onRefreshData = async (dt: any) => {
    // dt.search = getDefaultSearch();
    await fetchIndexData(); // 刷新到首页
};

const onAddDoc = async (dt: any) => {
    state.docEditDialog.isAdd = true;
    state.docEditDialog.instId = dt.instId;
    state.docEditDialog._id = '';
    state.docEditDialog.idxName = dt.idxName;
    state.docEditDialog.visible = true;
};

const onEditDoc = async (src: any) => {
    state.docEditDialog.isAdd = false;
    state.docEditDialog.instId = getNowDataTab().instId;
    state.docEditDialog.idxName = getNowDataTab().idxName;
    // 删除_id字段
    let obj = JSON.parse(src);
    state.docEditDialog._id = obj._id;
    delete obj._id;
    state.docEditDialog.doc = JSON.stringify(obj, null, 2);
    state.docEditDialog.visible = true;
};

const onEditSelectDoc = async (dt: any) => {
    if (dt.selectKeys.length > 1 || dt.selectKeys.length == 0) {
        ElMessage.warning(t('common.pleaseSelectOne'));
        return;
    }
    await onEditDoc(dt.selectKeys[0].src);
};

const onSwitchTrackTotal = async (dt: any) => {
    if (!dt.search.track_total_hits && dt.total === 10000) {
        dt.search.track_total_hits = true;
    } else {
        delete dt.search.track_total_hits;
    }
    if (dt.total >= 10000) {
        await fetchIndexData(); // 切换查询总数
    }
};

const onDeleteDocs = async (dt: any) => {
    let ids = dt.selectKeys.map((d: any) => d._id);
    // 二次确认后执行删除
    await useI18nDeleteConfirm(ids.join(', '));
    await doDeleteDoc(ids);
};

const doDeleteDoc = async (ids: any[]) => {
    let dataTab = getNowDataTab();
    await esApi.proxyReq('post', dataTab.instId, `/${dataTab.idxName}/_delete_by_query`, {
        query: { terms: { _id: ids } },
    });
    useI18nDeleteSuccessMsg();
    await refreshIndex(); // 删除后刷新索引
    setTimeout(async () => {
        await fetchIndexData(); // 删除后刷新数据
    }, 500);
};
const onDownload = async (dt: any) => {
    let ids = dt.selectKeys.map((d: any) => d._id);
    console.log(ids);
};
const onIndexDetail = async (data: any) => {
    let param = {
        idxName: data.params.idxName,
        instId: data.params.instId,
    };
    esIndexDetailRef.value.open(param);
};
const onUpload = async (dt: any) => {
    let ids = dt.selectKeys.map((d: any) => d._id);
    console.log(ids);
};
const onReindex = async (instId: any, idxName: string) => {
    state.reIndexDialog.visible = true;
    // 弹出新增/修改索引窗口
    state.reIndexDialog.instId = instId;
    state.reIndexDialog.idxName = idxName;
    state.reIndexDialog.visible = true;
    let indices = await getIndicesByInstId(instId);
    state.reIndexDialog.idxNames = indices
        .map((x: any) => x.index)
        .filter((x: any) => x != idxName && x.indexOf('.') < 0)
        .sort();
    console.log(state.reIndexDialog);
};
const onBasicSearch = async (dt: any) => {
    dt.searchDialogVisible = true;
};

const onEsSearch = async (data: any) => {
    let dt = getNowDataTab();
    data.from = 0;
    data.size = dt.search.size;
    dt.search = data;

    await fetchIndexData();
    dt.searchDialogVisible = false;
};

const onCheckColumnFilter = (column: any) => {
    column.hidden = !column._show;
};

const onCheckAllColumns = (dt: any) => {
    dt.columns.forEach((c: any) => {
        if (c.key != '_table_index' && c.key != '_selected') {
            c._show = dt.checkAllColumns;
            c.hidden = !c._show;
        }
    });
};

const onFilterColumns = (dt: any) => {
    if (!dt.columnsFilterText) {
        dt.columns.forEach((c: any) => {
            if (c.key != '_table_index' && c.key != '_selected') {
                c._filterd = true;
            }
        });
    } else {
        dt.columns.forEach((c: any) => {
            if (c.key != '_table_index' && c.key != '_selected') {
                c._filterd = c.key.toLowerCase().indexOf(dt.columnsFilterText.toLowerCase()) > -1;
            }
        });
    }
};

const onSelectAll = async (dt: any) => {
    dt.datas.forEach((d: any) => (d._selected = dt.selectAll));
    if (!dt.selectAll) {
        dt.selectKeys = [];
    } else {
        dt.selectKeys = dt.datas;
    }
};

const onSelectRow = (dt: any, item: any) => {
    if (item._selected) {
        dt.selectKeys.push(item);
    } else {
        dt.selectKeys = dt.selectKeys.filter((d: any) => d._id != item._id);
    }
};

const onRefreshStats = async (dt: any) => {
    let stats = await esApi.proxyReq('get', dt.instId, `/${dt.idxName}/_stats`);
    dt.params.idx['docs.count'] = stats.indices[dt.idxName].primaries.docs.count;
    if (stats.indices[dt.idxName].health) {
        dt.params.idx.health = stats.indices[dt.idxName].health;
    }
    if (stats.indices[dt.idxName].status) {
        dt.params.idx.status = stats.indices[dt.idxName].status;
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

const getHealthTagType = (health: string) => {
    return health == 'green' ? 'success' : health == 'yellow' ? 'warning' : 'danger';
};

const onInstClick = (nodeData: TagTreeNode) => {
    const label = `${nodeData.key}.dashboard`;
    state.activeName = label;

    let dataTab = state.dataTabs[label];
    if (dataTab) {
        return;
    }
    dataTab = {
        tabType: EsNodeType.Dashboard,
        params: nodeData.params,
        key: label,
        label: nodeData.label,
        name: label,
        instId: nodeData.params.id,
    };

    state.dataTabs[label] = dataTab;
};

const loadIdxs = async (params: any) => {
    console.log('lodIdxs', params);
    // 展示索引列表，显示索引名，文档总数，
    let idxNodes = [];
    let indices = {} as any;
    let keys = [] as string[];

    // 加载索引列表
    let indicesRes = await getIndicesByInstId(params.instId);
    indicesRes.forEach((x: any) => {
        if (state.showSysIndex) {
            indices[x.index] = x;
            keys.push(x.index);
        } else if (!x.index.startsWith('.') && x.index.indexOf('.') < 0) {
            indices[x.index] = x;
            keys.push(x.index);
        }
    });

    keys = keys.sort();

    for (let idxName of keys) {
        const idx = indices[idxName];
        const key = `es.${params.inst.id}.index.${idxName}`;
        idxNodes.push({
            instId: params.instId,
            idxName,
            idx,
            params,
            key: key,
            size: idx['store.size'],
            docs: formatDocSize(idx['docs.count'] || 0, 1),
        });
    }

    state.showSysIndex = false;

    return idxNodes;
};

defineExpose({
    onInstClick,
    loadIdxs,
    reloadNode,
    onRefreshIndices,
    onAddIndex,
    onShowSysIndex,
    onShowTemplate,
    onIdxClose,
    onIdxOpen,
    onIdxDelete,
    onIdxBaseSearch,
    onIndexDetail,
    onFlushIdx,
    onRefreshIdx,
    onIdxCopyName,
    onClearIdxCache,
    onIdxReindex,
    loadIndexData,
});
</script>

<style lang="scss">
#es-op-tabs {
    .el-tabs {
        --el-tabs-header-height: 30px;
        .el-tabs__header {
            margin: 0 0 5px;
            .el-tabs__item {
                padding: 0 5px;
            }
        }
    }
}

.es-table-data {
    .es-table {
        border-left: var(--el-table-border);
        border-top: var(--el-table-border);
    }

    .es-table-column {
        padding: 0 2px;
        font-size: 12px;
        border-right: var(--el-table-border);
    }

    .header-column-title {
        height: 30px;
        display: flex;
        justify-content: center;
    }

    .table-data-cell {
        width: 100%;
        height: 100%;
        line-height: 30px;
        cursor: pointer;
    }

    .data-selection {
        background-color: var(--el-table-current-row-bg-color);
    }

    .update_field_active {
        background-color: var(--el-color-success-light-3);
    }

    .column-type {
        color: var(--el-color-info-light-3);
        font-weight: bold;
        position: absolute;
        top: -7px;
        padding: 2px;
    }

    .column-right {
        position: absolute;
        top: 2px;
        right: 0;
        padding: 2px;
    }
    .el-table-v2__overlay {
        z-index: 1;
    }
}

.el-drawer__header {
    padding: 0 15px !important;
    height: 50px;
    display: flex;
    align-items: center;
    margin-bottom: 0 !important;
    border-bottom: 1px solid var(--el-border-color);
}
</style>
