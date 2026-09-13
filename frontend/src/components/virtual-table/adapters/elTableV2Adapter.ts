/**
 * el-table-v2 适配器
 *
 * 将 Element Plus el-table-v2 的 API 差异封装在适配器内部，
 * 对外暴露统一的 VirtualTableAdapter 接口。
 */
import { ElTableV2 } from 'element-plus';
import type {
    AdapterPropsInput,
    NormalizedColumn,
    NormalizedRowEventHandlers,
    TableInstance,
    VirtualTableAdapter,
} from './types';

function transformRowEventHandlers(handlers: NormalizedRowEventHandlers): Record<string, unknown> {
    const result: Record<string, (params: { event: MouseEvent; rowIndex: number; rowData: Record<string, unknown> }) => void> = {};

    if (handlers.onClick) {
        const fn = handlers.onClick;
        result.onClick = (p: { event: MouseEvent; rowIndex: number; rowData: Record<string, unknown> }) => fn({ event: p.event, rowIndex: p.rowIndex, rowData: p.rowData });
    }
    if (handlers.onDblclick) {
        const fn = handlers.onDblclick;
        result.onDblclick = (p: { event: MouseEvent; rowIndex: number; rowData: Record<string, unknown> }) => fn({ event: p.event, rowIndex: p.rowIndex, rowData: p.rowData });
    }
    if (handlers.onContextmenu) {
        const fn = handlers.onContextmenu;
        result.onContextmenu = (p: { event: MouseEvent; rowIndex: number; rowData: Record<string, unknown> }) => fn({ event: p.event, rowIndex: p.rowIndex, rowData: p.rowData });
    }

    return result;
}

function transformColumns(columns: NormalizedColumn[]): unknown[] {
    return columns.map((col) => col.raw);
}

function getProps(input: AdapterPropsInput): Record<string, unknown> {
    return {
        columns: transformColumns(input.columns),
        data: input.data,
        width: input.width,
        height: input.height,
        headerHeight: input.headerHeight,
        rowHeight: input.rowHeight,
        rowClass: input.rowClass,
        rowKey: null,
        fixed: true,
        rowEventHandlers: input.rowEventHandlers ? transformRowEventHandlers(input.rowEventHandlers) : undefined,
    };
}

function getInstance(tableRef: { value: unknown }): TableInstance {
    const ref = tableRef.value as { scrollTo: (opts: { scrollLeft?: number; scrollTop?: number }) => void; scrollToLeft: (left: number) => void } | null;
    return {
        scrollTo: (options) => ref?.scrollTo(options),
        scrollToLeft: (scrollLeft) => ref?.scrollToLeft(scrollLeft),
    };
}

export const elTableV2Adapter: VirtualTableAdapter = {
    name: 'el-table-v2',
    component: ElTableV2 as unknown as import('vue').Component,
    transformColumns,
    transformRowEventHandlers,
    getProps,
    getInstance,
};
