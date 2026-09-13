/**
 * 虚拟表格引擎适配器接口（全局通用）
 *
 * 将业务逻辑（选择、编辑、导出、右键菜单）与底层虚拟表格框架解耦。
 * 替换表格引擎时，仅需实现新的 VirtualTableAdapter 并设为 defaultAdapter。
 *
 * 架构示意：
 *
 *   ┌──────────────────────────────────────────────────┐
 *   │  VirtualTable.vue (全局通用组件)                   │
 *   │  ┌─────────────────┐  ┌────────────────────────┐ │
 *   │  │ Composables     │  │ Adapters               │ │
 *   │  │ - Selection     │  │ - ElTableV2Adapter     │ │
 *   │  │ - Export        │  │ - AgGridAdapter        │ │
 *   │  │ - ContextMenu   │  │ - VxeTableAdapter      │ │
 *   │  └────────┬────────┘  └──────────┬─────────────┘ │
 *   │           │     框架无关数据       │  框架适配      │
 *   │           └────────┬─────────────┘               │
 *   │                    ▼                             │
 *   │           ┌──────────────────┐                   │
 *   │           │ <component       │                   │
 *   │           │  :is="adapter" />│                   │
 *   │           └──────────────────┘                   │
 *   └──────────────────────────────────────────────────┘
 *
 * 使用方（DB / ES / Mongo ...）仅通过 slots 注入业务特有的单元格渲染逻辑。
 */
import type { Component } from 'vue';

// ==================== 通用列定义 ====================

/**
 * 虚拟表格列定义（全局通用，不依赖任何业务模块类型）
 * 各业务模块可在此基础上扩展自有字段。
 */
export interface VirtualTableColumn {
    /** 唯一标识（通常为 columnName / field name） */
    key: string;
    /** 显示标题 */
    title: string;
    /** 列宽 (px) */
    width?: number;
    /** 是否固定（左侧） */
    fixed?: boolean;
    /** 对齐方式 */
    align?: 'left' | 'center' | 'right';
    /** 是否隐藏 */
    hidden?: boolean;
    /** 是否可排序 */
    sortable?: boolean;
    /** 列名（业务字段名，可选） */
    columnName?: string;
    /** 数据类型（可选，业务方自行定义枚举） */
    dataType?: string;
    /** 数据类型角标（可选，用于列头展示） */
    dataTypeSubscript?: string;
    /** 列备注 / 注释（可选，用于列头展示） */
    remark?: string;
    /** 列备注文本（可选，用于列头副标题展示） */
    columnComment?: string;
    /** 表头 CSS class */
    headerClass?: string;
    /** 单元格 CSS class */
    class?: string;
    /** 扩展字段：业务模块可挂载任意自定义属性 */
    [extra: string]: unknown;
}

// ==================== 框架无关的列定义 ====================

/** 框架无关的列定义：由各适配器将 VirtualTableColumn 转换而来 */
export interface NormalizedColumn {
    key: string;
    title: string;
    width?: number;
    fixed?: boolean;
    align?: 'left' | 'center' | 'right';
    hidden?: boolean;
    sortable?: boolean;
    /** 原始列（透传给适配器） */
    raw: VirtualTableColumn;
}

// ==================== 框架无关的事件 ====================

export interface NormalizedRowClickEvent {
    event: MouseEvent;
    rowIndex: number;
    rowData: Record<string, unknown>;
}

export interface NormalizedCellEvent {
    event: MouseEvent | Event;
    rowIndex: number;
    columnIndex: number;
    rowData: Record<string, unknown>;
    column: NormalizedColumn;
}

export interface NormalizedScrollEvent {
    scrollLeft: number;
    scrollTop: number;
}

// ==================== 适配器接口 ====================

export interface VirtualTableAdapter {
    readonly name: string;
    readonly component: Component;
    transformColumns(columns: NormalizedColumn[]): unknown[];
    transformRowEventHandlers(handlers: NormalizedRowEventHandlers): unknown;
    getProps(input: AdapterPropsInput): Record<string, unknown>;
    getInstance(tableRef: { value: unknown }): TableInstance;
}

export interface TableInstance {
    scrollTo(options: { scrollLeft?: number; scrollTop?: number }): void;
    scrollToLeft(scrollLeft: number): void;
}

export interface NormalizedRowEventHandlers {
    onClick?: (e: NormalizedRowClickEvent) => void;
    onDblclick?: (e: NormalizedRowClickEvent) => void;
    onContextmenu?: (e: NormalizedRowClickEvent) => void;
}

export interface AdapterPropsInput {
    columns: NormalizedColumn[];
    data: Record<string, unknown>[];
    width: number;
    height: number;
    headerHeight: number;
    rowHeight: number;
    rowClass?: (row: { rowIndex: number }) => string;
    rowEventHandlers?: NormalizedRowEventHandlers;
    onScroll?: (e: NormalizedScrollEvent) => void;
}

// ==================== 列转换工具 ====================

export function normalizeColumns(columns: VirtualTableColumn[]): NormalizedColumn[] {
    return columns.map((col) => ({
        key: col.key,
        title: col.title,
        width: col.width,
        fixed: col.fixed,
        align: (col.align as 'left' | 'center' | 'right') ?? 'left',
        hidden: col.hidden,
        sortable: col.sortable,
        raw: col,
    }));
}
