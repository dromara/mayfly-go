/**
 * data-table 对外入口（barrel）。公共面只从这里出，内部按职责分层：
 * - constants：宽度量常量（内部件，不对外暴露）；
 * - table-column：TableColumn 列模型（链式声明 + 自适应列宽算法）；
 * - actions：行内操作域（ActionItem 声明 + 折叠方案 + 操作列计宽，渲染与计宽同源）；
 * - columnWidthMeasure / useColumnAutoWidth：DOM 测量 / 生命周期编排（内部件）；
 * - cellRenderers / searchItemTypes：type→内置单元格渲染器、type→搜索项元数据（两处注册表）；
 * - RowActions / RowActionEntry / rowAction：行内操作渲染与执行。
 *
 * 消费方多数只需 `import { TableColumn } from '@/components/data-table'`；不走 DataTable 的裸表格
 * 用 actionColumnProps + RowActions 成对声明操作列，故二者一并从 barrel 导出，免页面深导入。
 */
export * from './actions';
export { TableColumn } from './table-column';
export { default as RowActions } from './RowActions.vue';
