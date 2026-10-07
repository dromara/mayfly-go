/**
 * 虚拟树适配器统一导出
 *
 * 默认使用 el-tree-v2 适配器。
 * 若需替换为其他树引擎，修改下方 defaultAdapter 即可（引擎 DOM 皮肤随所选适配器导入）。
 */
export { elTreeV2Adapter } from './elTreeV2Adapter';
export type { VirtualTreeAdapter, VirtualTreeFieldNames, VirtualTreeInstance, NormalizedTreeEvents, TreeAdapterPropsInput } from './types';
export { defaultFieldNames } from './types';

import { elTreeV2Adapter } from './elTreeV2Adapter';
// 默认适配器引擎 DOM 的动效皮肤：皮肤归属引擎选择，而非容器
import './elTreeV2Style.scss';

/**
 * 默认适配器：全局唯一，替换此值即可切换树引擎。
 *
 * 示例：
 *   import { naiveTreeAdapter } from './naiveTreeAdapter';
 *   export const defaultAdapter = naiveTreeAdapter;
 */
export const defaultAdapter = elTreeV2Adapter;
