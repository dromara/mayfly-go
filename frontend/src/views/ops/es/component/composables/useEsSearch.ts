/**
 * ES 索引数据搜索逻辑 composable
 * 职责：索引管理、数据查询、分页、列过滤、行选择、刷新
 */
import Api from '@/common/Api';
import { Msg } from '@/hooks/useI18n';
import { esApi } from '@/views/ops/es/api';
import type { EsColumn, EsDoc, EsHit, EsIndexInfo, EsIndexStatsRes, EsSearchParam, EsSearchRes } from '@/views/ops/es/types';
import { useIntervalFn } from '@vueuse/core';
import { onMounted, reactive, ref } from 'vue';
import type { Composer } from 'vue-i18n';

export interface UseEsSearchOptions {
    instId: number;
    i18n: Composer;
}

const getDefaultSearch = (): EsSearchParam => ({ sort: {}, query: { bool: { must: [], should: [], must_not: [] } }, aggs: {}, from: 0, size: 25 });

export function useEsSearch(options: UseEsSearchOptions) {
    const { instId, i18n } = options;
    const { t } = i18n;

    const state = reactive({
        columns: [] as EsColumn[],
        fields: [] as string[],
        datas: [] as EsDoc[],
        total: 0,
        searchRes: {} as EsSearchRes,
        rowHeight: 30,
        selectAll: false,
        selectKeys: [] as EsDoc[],
        lastSelectedIndex: -1,
        columnsFilterText: '',
        checkAllColumns: true,
        loading: true,
        abortSearch: () => {},
        execTime: 0,
        currentFrom: 25,
        search: getDefaultSearch(),
        searchDialogVisible: false,
    });

    // Index state for self-contained index switching
    const currentIdxName = ref('');
    const currentIdxInfo = ref<EsIndexInfo | null>(null);
    const indices = ref<EsIndexInfo[]>([]);

    onMounted(async () => {
        await fetchIndices();
        setTimeout(fetchIndexData, 300);
    });

    const fetchIndices = async () => {
        const res = await esApi.proxyReq<EsIndexInfo[]>('get', instId, `/_cat/indices/?h=index,health,status,uuid,pri,rep,docs.count,docs.deleted,store.size,sc,cd`);
        indices.value = (res || []).filter((idx) => !idx.index.startsWith('.')).sort((a, b) => a.index.localeCompare(b.index));
        // Auto-select first index if no idxName provided
        if (!currentIdxName.value && indices.value.length > 0) {
            currentIdxName.value = indices.value[0].index;
            currentIdxInfo.value = indices.value[0];
        } else if (currentIdxName.value) {
            currentIdxInfo.value = indices.value.find((idx: EsIndexInfo) => idx.index === currentIdxName.value) || currentIdxInfo.value;
        }
    };

    const onIndexChange = (name: string) => {
        currentIdxInfo.value = indices.value.find((idx: EsIndexInfo) => idx.index === name) || null;
        state.search = getDefaultSearch();
        fetchIndexData();
    };

    // ---- Data fetching ----

    const fetchIndexData = async () => {
        if (!currentIdxName.value) return;
        state.execTime = 0;
        const { pause, resume } = useIntervalFn(() => {
            state.execTime += 0.1;
        }, 100);
        resume();
        state.loading = true;

        state.selectAll = false;
        state.selectKeys = [];
        state.lastSelectedIndex = -1;

        let api = Api.newPost<EsSearchRes>(`/es/instance/proxy/${instId}/${currentIdxName.value}/_search`);

        const { execute: execSearch, data: searchRes, abort: abortSearch } = api.useApi(state.search, { esProxyReq: true });
        state.abortSearch = () => {
            abortSearch();
            state.loading = false;
            pause();
        };
        await execSearch();
        const searchResValue = searchRes.value ?? ({} as EsSearchRes);
        state.searchRes = searchResValue;
        let error = searchResValue.error || (searchResValue.failures && searchResValue.failures.length > 0 && searchResValue.failures[0]);
        if (error) {
            state.loading = false;
            pause();
            return await esApi.alertError(error, t('es.execError'));
        }

        let fieldMap: Record<string, { width: number }> = {} as Record<string, { width: number }>;
        fieldMap['_id'] = { width: 50 };

        state.datas = state.searchRes.hits.hits.map((a: EsHit) => {
            let src = JSON.parse(JSON.stringify(a._source));
            src._id = a._id;
            let source = a._source;
            source._id = a._id;
            source._score = a._score;
            fieldMap['_score'] = { width: 40 };
            for (let k in source) {
                if (typeof source[k] != 'string' && typeof source[k] != 'number' && source[k] !== null && typeof source[k] != 'boolean') {
                    source[k] = JSON.stringify(source[k]);
                }
                let column = fieldMap[k] || { width: 50 };
                try {
                    const val = source[k];
                    let valLength = val ? Math.max((val as string).length, k.length) : k.length;
                    column.width = Math.max(Math.max(Math.min(220, (valLength || 10) * 10), 50), column.width);
                } catch {
                    column.width = 50;
                }
                fieldMap[k] = column;
            }
            source.src = JSON.stringify(src, null, 2);
            source._selected = false;
            return source as EsDoc;
        });

        state.total = state.searchRes.hits?.total.value || 0;

        if (state.datas.length > 0) {
            let keys = Object.keys(fieldMap).sort();
            state.fields = keys.filter((k) => k != '_score');
            state.columns = keys.map((k) => ({ title: k, width: fieldMap[k].width, key: k, dataKey: k, class: 'es-table-column', _filterd: true, _show: true }));
            state.columns.unshift({
                title: '#',
                width: 50,
                key: '_table_index',
                class: 'es-table-column',
                align: 'center',
                _filterd: false,
            });
            state.columns.unshift({
                title: 'checkbox',
                width: 30,
                key: '_selected',
                class: 'es-table-column',
                align: 'center',
                _filterd: false,
            });
        }
        pause();
        state.loading = false;
        state.currentFrom = state.search.from;
    };

    // ---- Pagination ----

    const onChangePageSize = async (size: number) => {
        state.search.size = size;
        state.search.from = 0;
        await fetchIndexData();
    };

    const onFirstPage = async () => {
        state.search.from = 0;
        await fetchIndexData();
    };

    const onNextPage = async () => {
        state.search.from = state.search.from + state.search.size;
        await fetchIndexData();
    };

    const onPrevPage = async () => {
        state.search.from = Math.max(0, state.search.from - state.search.size);
        await fetchIndexData();
    };

    const onSwitchTrackTotal = async () => {
        if (!state.search.track_total_hits && state.total === 10000) {
            state.search.track_total_hits = true;
        } else {
            delete state.search.track_total_hits;
        }
        if (state.total >= 10000) {
            await fetchIndexData();
        }
    };

    // ---- Refresh ----

    const refreshIndex = async () => {
        await esApi.proxyReq('post', instId, `/${currentIdxName.value}/_refresh`);
    };

    const onRefreshData = async () => {
        await fetchIndexData();
    };

    const onRefreshStats = async () => {
        const name = currentIdxName.value;
        let stats = await esApi.proxyReq<EsIndexStatsRes>('get', instId, `/${name}/_stats`);
        if (currentIdxInfo.value) {
            currentIdxInfo.value['docs.count'] = stats.indices[name]?.primaries?.docs?.count;
            if (stats.indices[name]?.health) currentIdxInfo.value.health = stats.indices[name].health;
            if (stats.indices[name]?.status) currentIdxInfo.value.status = stats.indices[name].status;
        }
    };

    // ---- Search ----

    const onBasicSearch = () => {
        if (!currentIdxName.value) {
            Msg.warning('es.selectIndexFirst');
            return;
        }
        state.searchDialogVisible = true;
    };

    const onEsSearch = async (data: EsSearchParam) => {
        data.from = 0;
        data.size = state.search.size;
        state.search = data;
        await fetchIndexData();
        state.searchDialogVisible = false;
    };

    // ---- Column filter ----

    const onCheckColumnFilter = (column: EsColumn) => {
        column.hidden = !column._show;
    };

    const onCheckAllColumns = () => {
        state.columns.forEach((c: EsColumn) => {
            if (c.key != '_table_index' && c.key != '_selected') {
                c._show = state.checkAllColumns;
                c.hidden = !c._show;
            }
        });
    };

    const onFilterColumns = () => {
        if (!state.columnsFilterText) {
            state.columns.forEach((c: EsColumn) => {
                if (c.key != '_table_index' && c.key != '_selected') {
                    c._filterd = true;
                }
            });
        } else {
            state.columns.forEach((c: EsColumn) => {
                if (c.key != '_table_index' && c.key != '_selected') {
                    c._filterd = c.key.toLowerCase().indexOf(state.columnsFilterText.toLowerCase()) > -1;
                }
            });
        }
    };

    // ---- Row selection ----

    const updateSelectAll = () => {
        state.selectAll = state.datas.length > 0 && state.datas.every((d: EsDoc) => d._selected);
    };

    const onSelectAll = () => {
        state.lastSelectedIndex = -1;
        state.datas.forEach((d: EsDoc) => (d._selected = state.selectAll));
        if (!state.selectAll) {
            state.selectKeys = [];
        } else {
            state.selectKeys = state.datas;
        }
    };

    const onSelectRow = (item: EsDoc) => {
        if (item._selected) {
            state.selectKeys.push(item);
        } else {
            state.selectKeys = state.selectKeys.filter((d: EsDoc) => d._id != item._id);
        }
        state.lastSelectedIndex = state.datas.findIndex((d: EsDoc) => d._id === item._id);
        updateSelectAll();
    };

    const onRowClickHandler = ({ rowData, rowIndex, event }: { rowData: EsDoc; rowIndex: number; event: Event }) => {
        const mouseEvent = event as MouseEvent;
        // Ignore clicks on the checkbox column (checkbox has its own handler)
        const target = mouseEvent.target as HTMLElement;
        if (target.closest('.el-checkbox') || target.closest('.el-checkbox__input')) return;

        if (mouseEvent.shiftKey && state.lastSelectedIndex >= 0) {
            // Shift + click: range select
            const start = Math.min(state.lastSelectedIndex, rowIndex);
            const end = Math.max(state.lastSelectedIndex, rowIndex);
            for (let i = start; i <= end; i++) {
                if (!state.datas[i]._selected) {
                    state.datas[i]._selected = true;
                    state.selectKeys.push(state.datas[i]);
                }
            }
        } else if (mouseEvent.ctrlKey || mouseEvent.metaKey) {
            // Ctrl/Cmd + click: toggle single row
            state.datas[rowIndex]._selected = !state.datas[rowIndex]._selected;
            if (state.datas[rowIndex]._selected) {
                state.selectKeys.push(state.datas[rowIndex]);
            } else {
                state.selectKeys = state.selectKeys.filter((d: EsDoc) => d._id !== rowData._id);
            }
            state.lastSelectedIndex = rowIndex;
        } else {
            // Normal click: clear all, select this row only
            state.datas.forEach((d: EsDoc) => (d._selected = false));
            state.selectKeys = [];
            state.datas[rowIndex]._selected = true;
            state.selectKeys = [state.datas[rowIndex]];
            state.lastSelectedIndex = rowIndex;
        }
        updateSelectAll();
    };

    const rowEventHandlers = {
        onClick: onRowClickHandler,
    };

    // ---- Helpers ----

    const selectIndex = async (name: string) => {
        if (indices.value.length === 0) {
            await fetchIndices();
        }
        currentIdxName.value = name;
        onIndexChange(name);
    };

    return {
        state,
        currentIdxName,
        currentIdxInfo,
        indices,
        fetchIndices,
        onIndexChange,
        fetchIndexData,
        onChangePageSize,
        onFirstPage,
        onNextPage,
        onPrevPage,
        onSwitchTrackTotal,
        refreshIndex,
        onRefreshData,
        onRefreshStats,
        onBasicSearch,
        onEsSearch,
        onCheckColumnFilter,
        onCheckAllColumns,
        onFilterColumns,
        onSelectAll,
        onSelectRow,
        rowEventHandlers,
        selectIndex,
    };
}
