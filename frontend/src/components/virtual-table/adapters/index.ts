/**
 * 虚拟表格适配器统一导出
 *
 * 默认使用 el-table-v2 适配器。
 * 若需替换为其他表格引擎，修改下方 defaultAdapter 即可。
 */
export { elTableV2Adapter } from './elTableV2Adapter';
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
} from './types';
export { normalizeColumns } from './types';

import { elTableV2Adapter } from './elTableV2Adapter';

/**
 * 默认适配器：全局唯一，替换此值即可切换表格引擎。
 *
 * 示例：
 *   import { agGridAdapter } from './agGridAdapter';
 *   export const defaultAdapter = agGridAdapter;
 */
export const defaultAdapter = elTableV2Adapter;
