/**
 * 虚拟树引擎适配器接口（全局通用）
 *
 * 将业务逻辑（分组、筛选、勾选语义、右键菜单、行渲染）与底层虚拟树框架解耦。
 * 替换树引擎时，仅需实现新的 VirtualTreeAdapter 并设为 defaultAdapter。
 *
 * 契约边界：
 *   - 容器（VirtualTree.vue）只面向本文件的语义化 props、归一化事件与归一化实例方法；
 *   - 引擎 prop 名、事件签名、实例方法名的差异全部收口在适配器内部；
 *   - 叶子判定交给引擎（children 为空即叶子），数据层负责给「可展开但未加载」的节点占位 children；
 *   - 插槽按名透传：适配器所选引擎组件须暴露 default（行渲染，槽参含 { data } 原始节点）与
 *     empty（空态）插槽，容器不做插槽名映射（详见 VirtualTreeAdapter.component）。
 */
import type { Component } from 'vue';

/** 字段映射：消费方节点结构的字段名声明，引擎配置由适配器翻译 */
export interface VirtualTreeFieldNames {
    value: string;
    label: string;
    children: string;
    disabled?: string;
}

export const defaultFieldNames: VirtualTreeFieldNames = {
    value: 'key',
    label: 'label',
    children: 'children',
    disabled: 'disabled',
};

/** 归一化实例方法：容器 expose 的语义契约，适配器翻译到引擎实例 */
export interface VirtualTreeInstance {
    /** 选中并高亮节点 */
    setCurrentKey(key: string): void;
    /** 滚动到节点（strategy: auto/center/start/end/nearest） */
    scrollToNode(key: string, strategy?: 'auto' | 'center' | 'start' | 'end' | 'nearest'): void;
    /** 滚动到像素偏移 */
    scrollTo(offset: number): void;
    getCheckedKeys(leafOnly?: boolean): string[];
    setCheckedKeys(keys: string[]): void;
    /** 全量设置展开集（受控展开的兜底通道） */
    setExpandedKeys(keys: string[]): void;
    /** 命令式过滤入口（立即生效、不走防抖）；声明式过滤请用容器的 filterText prop */
    filter(query: string): void;
}

/** 归一化事件：容器只认这些签名，引擎事件口径在适配器消化 */
export interface NormalizedTreeEvents {
    onNodeClick?: (node: Record<string, unknown>) => void;
    onNodeExpand?: (node: Record<string, unknown>) => void;
    onNodeCollapse?: (node: Record<string, unknown>) => void;
    /** 勾选变化：叶子 key 列表（父节点/半选口径在适配器归一） */
    onCheck?: (checkedLeafKeys: string[]) => void;
}

export interface TreeAdapterPropsInput {
    data: Record<string, unknown>[];
    fieldNames: VirtualTreeFieldNames;
    height: number;
    /** 行高（像素）：虚拟引擎按此定位每一行，缺省则用引擎默认值 */
    rowHeight?: number;
    expandedKeys: string[];
    checkable: boolean;
    checkOnClickLeaf: boolean;
    highlightCurrent: boolean;
    expandOnClickNode: boolean;
    /** 本地过滤谓词（引擎不支持过滤时适配器可 no-op） */
    filterMethod?: (query: string, node: Record<string, unknown>) => boolean;
    events: NormalizedTreeEvents;
}

export interface VirtualTreeAdapter {
    readonly name: string;
    /**
     * 引擎组件。适配义务：须暴露 default（行渲染，槽参 { data } 为原始节点对象）与 empty
     * （空数据占位）两个插槽——容器 VirtualTree.vue 直接按名透传消费方插槽，不做映射。
     * 换用插槽命名不同的引擎时，在适配器内包一层组件对齐该约定即可，容器无需改动。
     */
    readonly component: Component;
    getProps(input: TreeAdapterPropsInput): Record<string, unknown>;
    getInstance(treeRef: { value: unknown }): VirtualTreeInstance;
}
