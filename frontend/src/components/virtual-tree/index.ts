/**
 * 虚拟树统一导出：容器组件 + 适配器契约。
 * 消费方只应 import 本入口，不直接触碰 adapters 内部实现。
 */
export { default as VirtualTree } from './VirtualTree.vue';
export { defaultAdapter, elTreeV2Adapter, defaultFieldNames } from './adapters';
export type { VirtualTreeAdapter, VirtualTreeFieldNames, VirtualTreeInstance, NormalizedTreeEvents, TreeAdapterPropsInput } from './adapters';
