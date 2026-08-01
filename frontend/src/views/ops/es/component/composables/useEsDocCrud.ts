/**
 * ES 文档 CRUD + 导出逻辑 composable
 * 职责：文档新增/编辑/删除、导出对话框状态管理、大数据量后端导出 + 进度轮询
 */
import Api from '@/common/Api';
import { exportCsv, exportExcel, exportFile } from '@/common/utils/export';
import { getClientId, getToken } from '@/common/utils/storage';
import { Msg, useI18nDeleteConfirm } from '@/hooks/useI18n';
import { esApi } from '@/views/ops/es/api';
import type { EsCountRes, EsDoc, EsExportProgress, EsSearchParam } from '@/views/ops/es/types';
import { computed, reactive, watch } from 'vue';

export interface UseEsDocCrudOptions {
    instId: number;
    currentIdxName: () => string;
    state: {
        search: EsSearchParam;
        selectKeys: EsDoc[];
        datas: EsDoc[];
        fields: string[];
    };
    refreshIndex: () => Promise<void>;
    fetchIndexData: () => Promise<void>;
}

export function useEsDocCrud(options: UseEsDocCrudOptions) {
    const { instId, state, refreshIndex, fetchIndexData } = options;
    const currentIdxName = options.currentIdxName;

    // Doc edit dialog state
    const docEditDialog = reactive({
        isAdd: true,
        instId: 0 as number,
        doc: '',
        idxName: '',
        _id: '',
        visible: false,
    });

    // Export dialog state
    const exportDialog = reactive({
        visible: false,
        scope: 'selected' as 'selected' | 'query' | 'all',
        type: 'csv' as 'csv' | 'excel' | 'json',
        fields: [] as string[],
        allFields: true,
        loading: false,
        queryTotal: -1, // -1 means not queried yet
        queryTotalLoading: false,
        progress: null as EsExportProgress | null,
        progressTimer: null as ReturnType<typeof setInterval> | null,
    });

    // ---- Doc CRUD ----

    const onAddDoc = () => {
        docEditDialog.isAdd = true;
        docEditDialog.instId = instId;
        docEditDialog.idxName = currentIdxName();
        docEditDialog._id = '';
        docEditDialog.doc = '';
        docEditDialog.visible = true;
    };

    const onEditDoc = (src: string) => {
        docEditDialog.isAdd = false;
        docEditDialog.instId = instId;
        docEditDialog.idxName = currentIdxName();
        const obj = JSON.parse(src);
        docEditDialog._id = obj._id;
        delete obj._id;
        docEditDialog.doc = JSON.stringify(obj, null, 2);
        docEditDialog.visible = true;
    };

    const onEditSelectDoc = () => {
        if (state.selectKeys.length > 1 || state.selectKeys.length == 0) {
            Msg.warning('common.pleaseSelectOne');
            return;
        }
        onEditDoc(state.selectKeys[0].src || '');
    };

    const onEditRowSuccess = async () => {
        docEditDialog.visible = false;
        await refreshIndex();
        await fetchIndexData();
    };

    const onDeleteDocs = async () => {
        let ids = state.selectKeys.map((d: EsDoc) => d._id);
        await useI18nDeleteConfirm(ids.join(', '));
        await doDeleteDoc(ids);
    };

    const doDeleteDoc = async (ids: string[]) => {
        await esApi.proxyReq('post', instId, `/${currentIdxName()}/_delete_by_query`, {
            query: { terms: { _id: ids } },
        });
        Msg.deleteSuccess();
        await refreshIndex();
        setTimeout(async () => {
            await fetchIndexData();
        }, 500);
    };

    // ---- Export ----

    const hasCustomQuery = computed(() => {
        const query = state.search.query;
        if (!query) return false;
        const bool = query.bool;
        if (!bool) return Object.keys(query).length > 0;
        return (bool.must?.length ?? 0) > 0 || (bool.should?.length ?? 0) > 0 || (bool.must_not?.length ?? 0) > 0 || ((bool as Record<string, unknown>).filter as unknown[] | undefined)?.length;
    });

    const onOpenExportDialog = () => {
        exportDialog.scope = state.selectKeys.length > 0 ? 'selected' : hasCustomQuery.value ? 'query' : 'all';
        exportDialog.type = 'csv';
        exportDialog.fields = [...state.fields];
        exportDialog.allFields = true;
        exportDialog.loading = false;
        exportDialog.queryTotal = -1;
        exportDialog.queryTotalLoading = false;
        exportDialog.visible = true;
        if (exportDialog.scope === 'all' || exportDialog.scope === 'query') {
            fetchQueryTotal();
        }
    };

    const onExportFieldsToggle = () => {
        exportDialog.fields = exportDialog.allFields ? [...state.fields] : [];
    };

    const onExportFieldsChange = () => {
        exportDialog.allFields = exportDialog.fields.length === state.fields.length;
    };

    let queryTotalAbort: (() => void) | null = null;
    const fetchQueryTotal = async () => {
        if (!currentIdxName()) return;
        exportDialog.queryTotalLoading = true;
        exportDialog.queryTotal = -1;
        const api = Api.newPost<EsCountRes>(`/es/instance/proxy/${instId}/${currentIdxName()}/_count`);
        const body = state.search.query ? { query: state.search.query } : {};
        const { execute, data, abort } = api.useApi(body, { esProxyReq: true });
        queryTotalAbort = abort;
        await execute();
        if (data.value && typeof data.value.count === 'number') {
            exportDialog.queryTotal = data.value.count;
        }
        exportDialog.queryTotalLoading = false;
    };

    watch(
        () => exportDialog.scope,
        (scope: 'selected' | 'query' | 'all') => {
            if (scope === 'all' || scope === 'query') {
                fetchQueryTotal();
            } else {
                if (queryTotalAbort) {
                    queryTotalAbort();
                    queryTotalAbort = null;
                }
                exportDialog.queryTotal = -1;
                exportDialog.queryTotalLoading = false;
            }
        }
    );

    const onConfirmExport = async () => {
        exportDialog.loading = true;
        exportDialog.progress = null;
        try {
            if (exportDialog.scope === 'selected' && state.selectKeys.length <= 10000) {
                await exportSelectedData();
            } else {
                await exportAllData();
            }
            exportDialog.visible = false;
        } catch (e: unknown) {
            Msg.error((e instanceof Error ? e.message : '') || 'es.export.title');
        } finally {
            exportDialog.loading = false;
            stopProgressPolling();
        }
    };

    const getExportData = (rows: EsDoc[]) => {
        const columns = exportDialog.fields;
        return { rows, columns };
    };

    const exportSelectedData = async () => {
        const selectedRows = state.selectKeys.length > 0 ? state.selectKeys : state.datas;
        if (!selectedRows || selectedRows.length === 0) {
            Msg.warning('es.export.noData');
            return;
        }

        const { rows, columns } = getExportData(selectedRows);
        const filename = `${currentIdxName()}-${Date.now()}`;

        switch (exportDialog.type) {
            case 'csv':
                exportCsv(filename, columns, rows as Record<string, unknown>[]);
                break;
            case 'excel':
                await exportExcel(filename, [{ name: currentIdxName(), columns, datas: rows }]);
                break;
            case 'json':
                exportFile(
                    `${filename}.json`,
                    JSON.stringify(
                        rows.map((r: EsDoc) => {
                            const obj: Record<string, unknown> = {};
                            columns.forEach((col: string) => {
                                obj[col] = r[col];
                            });
                            return obj;
                        }),
                        null,
                        2
                    )
                );
                break;
        }
    };

    const exportAllData = async () => {
        // Build download URL for backend export
        const exportUrl = esApi.exportData.getUrl().replace('{instanceId}', String(instId));

        // Generate UUID for progress tracking
        const exportId = crypto.randomUUID();

        // If "selected" scope with large dataset, pass selected IDs as ES terms query
        // If "query" scope, pass the current search query
        let searchQuery: Record<string, unknown> | null = null;
        if (exportDialog.scope === 'selected' && state.selectKeys.length > 0) {
            searchQuery = {
                query: { terms: { _id: state.selectKeys.map((d: EsDoc) => d._id) } },
            };
        } else if (exportDialog.scope === 'query') {
            searchQuery = state.search;
        }

        // Build fields: nil means all, otherwise pass selected fields
        const fields = exportDialog.allFields ? null : exportDialog.fields;

        const body = {
            idxName: currentIdxName(),
            searchQuery,
            exportType: exportDialog.type,
            fields,
            exportId,
        };

        // Start progress polling
        startProgressPolling(exportId);

        const response = await fetch(exportUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                Authorization: getToken() || '',
                ClientId: getClientId() || '',
            },
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            throw new Error(`Export failed: HTTP ${response.status}`);
        }

        const blob = await response.blob();
        const disposition = response.headers.get('Content-Disposition');
        let downloadFilename = `${currentIdxName()}.zip`;
        if (disposition) {
            const match = disposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
            if (match && match[1]) {
                downloadFilename = match[1].replace(/['"]/g, '');
            }
        }

        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = downloadFilename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
    };

    const startProgressPolling = (exportId: string) => {
        stopProgressPolling();
        exportDialog.progress = { total: 0, processed: 0, phase: 'querying', done: false };
        exportDialog.progressTimer = setInterval(async () => {
            try {
                const res = await esApi.exportProgress.request({ exportId });
                if (res) {
                    exportDialog.progress = res;
                    if (res.done) {
                        stopProgressPolling();
                    }
                }
            } catch {
                stopProgressPolling();
            }
        }, 1000);
    };

    const stopProgressPolling = () => {
        if (exportDialog.progressTimer) {
            clearInterval(exportDialog.progressTimer);
            exportDialog.progressTimer = null;
        }
    };

    return {
        docEditDialog,
        exportDialog,
        hasCustomQuery,
        onAddDoc,
        onEditDoc,
        onEditSelectDoc,
        onEditRowSuccess,
        onDeleteDocs,
        doDeleteDoc,
        onOpenExportDialog,
        onExportFieldsToggle,
        onExportFieldsChange,
        onConfirmExport,
    };
}
