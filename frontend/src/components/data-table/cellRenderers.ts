import type { Component } from 'vue';
import type { TableColumn } from './table-column';
import TagCell from './cells/TagCell.vue';

/**
 * 内置「单元格值渲染器」的统一契约：注册表按 TableColumn.type 选中一个组件，DataTable 用这套 props 喂它。
 * 各渲染器只取自己关心的字段（如 TagCell 读 item.typeParam 当枚举源）。
 */
export interface CellRendererProps {
    /** 列模型（渲染器从中读取自身需要的配置，如 typeParam） */
    item: TableColumn;
    /** 当前行数据 */
    row: Record<string, unknown>;
    /** 该列在本行的展示值（已由 getValueByData 取好，含 formatFunc） */
    value: unknown;
    /** 表格尺寸，透传给底层控件 */
    size?: string;
}

/**
 * type → 内置渲染器。新增一种内置展示形态（进度条、图片、开关只读态等）= 在此登记一个组件，
 * 无需改动 DataTable 的模板分支（开闭原则）。
 *
 * 边界：`actions`（行内操作列）与 `slot`（页面自定义插槽）是正交于 type 的「结构模式」，
 * 由 DataTable 先行判定、不走本表——它们不是「按 type 选渲染器」的语义。普通文本列 type 为空，
 * 命中不到本表，回落 DataTable 的默认文本渲染。
 */
const cellRenderers: Record<string, Component> = {
    tag: TagCell,
};

/** 解析某列的内置值渲染器；未登记（普通文本列）返回 undefined，由 DataTable 回落默认文本渲染 */
export function resolveCellRenderer(item: TableColumn): Component | undefined {
    return item.type ? cellRenderers[item.type] : undefined;
}
