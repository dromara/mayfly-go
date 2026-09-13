/**
 * 虚拟表格全局组件
 *
 * 使用方式：
 *   import { VirtualTable } from '@/components/virtual-table';
 *
 * 替换引擎：
 *   import { agGridAdapter } from '@/components/virtual-table/adapters/agGridAdapter.example';
 *   <VirtualTable :adapter="agGridAdapter" ... />
 *
 * 职责边界：
 *   VirtualTable 仅负责容器尺寸管理、适配器渲染、加载覆盖层透传、空数据占位行。
 *   行选择、右键菜单、列头渲染等均由消费者通过 slots 自行实现。
 */
export { default as VirtualTable } from './VirtualTable.vue';

// 适配器
export { defaultAdapter, elTableV2Adapter, normalizeColumns } from './adapters';
export type {
    VirtualTableAdapter,
    VirtualTableColumn,
    NormalizedColumn,
    NormalizedRowClickEvent,
    NormalizedCellEvent,
    NormalizedScrollEvent,
    NormalizedRowEventHandlers,
    AdapterPropsInput,
    TableInstance,
} from './adapters';
