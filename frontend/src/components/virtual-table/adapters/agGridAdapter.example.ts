/**
 * AG Grid 适配器示例（参考实现）
 *
 * 此文件展示了如何为新表格引擎编写适配器。
 * 实际使用时需安装 ag-grid-vue3 依赖并完善具体实现。
 *
 * 使用方式：
 *   import { agGridAdapter } from '@/components/virtual-table/adapters/agGridAdapter.example';
 *   <VirtualTable :adapter="agGridAdapter" ... />
 */
import type { Component } from 'vue';
import type {
    AdapterPropsInput,
    NormalizedColumn,
    NormalizedRowEventHandlers,
    TableInstance,
    VirtualTableAdapter,
} from './types';

/**
 * AG Grid 列定义转换
 */
function transformColumns(columns: NormalizedColumn[]): unknown[] {
    return columns.map((col) => ({
        field: col.key,
        headerName: col.title,
        width: col.width ?? 150,
        pinned: col.fixed ? ('left' as const) : false,
        hide: col.hidden,
        sortable: col.sortable,
    }));
}

/**
 * AG Grid 事件绑定转换
 */
function transformRowEventHandlers(_handlers: NormalizedRowEventHandlers): Record<string, unknown> {
    return {};
}

/**
 * AG Grid 组件 props 构建
 */
function getProps(input: AdapterPropsInput): Record<string, unknown> {
    return {
        columnDefs: transformColumns(input.columns),
        rowData: input.data,
        domLayout: 'normal',
        suppressRowClickSelection: true,
        rowHeight: input.rowHeight,
        headerHeight: input.headerHeight,
    };
}

/**
 * AG Grid 实例方法适配
 */
function getInstance(tableRef: { value: unknown }): TableInstance {
    const api = (tableRef.value as { api?: { ensureColumnVisible: (col: string) => void } })?.api;
    return {
        scrollTo: (_options) => {
            // AG Grid scrollTo implementation
        },
        scrollToLeft: (_scrollLeft) => {
            // AG Grid horizontal scroll
        },
    };
}

export const agGridAdapter: VirtualTableAdapter = {
    name: 'ag-grid',
    component: null as unknown as Component,
    transformColumns,
    transformRowEventHandlers,
    getProps,
    getInstance,
};
