/**
 * el-tree-v2 适配器
 *
 * 将 Element Plus el-tree-v2 的 API 差异封装在适配器内部：字段映射、
 * 勾选口径归一（check 事件的全量节点 → 叶子 key 列表）、实例方法翻译。
 * 引擎 DOM 的动效皮肤见同目录 elTreeV2Style.scss，随 defaultAdapter 的选择生效。
 */
import { ElTreeV2 } from 'element-plus';
import type { Component } from 'vue';
import type { NormalizedTreeEvents, TreeAdapterPropsInput, VirtualTreeAdapter, VirtualTreeFieldNames, VirtualTreeInstance } from './types';

/** v2 的 check 事件给全量勾选节点，归一化只保留叶子（children 为空），与 getCheckedKeys(true) 同口径 */
function leafKeysOf(checkedNodes: Record<string, unknown>[], fieldNames: VirtualTreeFieldNames): string[] {
    return checkedNodes.filter((node) => !(node[fieldNames.children] as unknown[] | undefined)?.length).map((node) => String(node[fieldNames.value]));
}

function transformEvents(events: NormalizedTreeEvents, fieldNames: VirtualTreeFieldNames): Record<string, unknown> {
    const result: Record<string, unknown> = {};

    if (events.onNodeClick) {
        const fn = events.onNodeClick;
        result.onNodeClick = (data: Record<string, unknown>) => fn(data);
    }
    if (events.onNodeExpand) {
        const fn = events.onNodeExpand;
        result.onNodeExpand = (data: Record<string, unknown>) => fn(data);
    }
    if (events.onNodeCollapse) {
        const fn = events.onNodeCollapse;
        result.onNodeCollapse = (data: Record<string, unknown>) => fn(data);
    }
    if (events.onCheck) {
        const fn = events.onCheck;
        result.onCheck = (_data: Record<string, unknown>, checkedInfo: { checkedNodes?: Record<string, unknown>[] }) =>
            fn(leafKeysOf(checkedInfo?.checkedNodes ?? [], fieldNames));
    }

    return result;
}

function getProps(input: TreeAdapterPropsInput): Record<string, unknown> {
    return {
        data: input.data,
        props: {
            value: input.fieldNames.value,
            label: input.fieldNames.label,
            children: input.fieldNames.children,
            disabled: input.fieldNames.disabled,
        },
        height: input.height,
        itemSize: input.rowHeight,
        indent: 10,
        showCheckbox: input.checkable,
        checkOnClickLeaf: input.checkOnClickLeaf,
        highlightCurrent: input.highlightCurrent,
        expandOnClickNode: input.expandOnClickNode,
        defaultExpandedKeys: input.expandedKeys,
        filterMethod: input.filterMethod ?? undefined,
        ...transformEvents(input.events, input.fieldNames),
    };
}

/** el-tree-v2 实例方法的最小声明（仅收口本适配器用到的成员） */
interface ElTreeV2Expose {
    setCurrentKey: (key: string) => void;
    scrollToNode: (key: string, strategy?: 'auto' | 'center' | 'start' | 'end' | 'nearest') => void;
    scrollTo: (offset: number) => void;
    getCheckedKeys: (leafOnly?: boolean) => (string | number)[];
    setCheckedKeys: (keys: string[]) => void;
    setExpandedKeys: (keys: string[]) => void;
    filter: (query: string) => void;
}

function getInstance(treeRef: { value: unknown }): VirtualTreeInstance {
    const engine = () => treeRef.value as ElTreeV2Expose | null;
    return {
        setCurrentKey: (key) => engine()?.setCurrentKey(key),
        scrollToNode: (key, strategy) => engine()?.scrollToNode(key, strategy),
        scrollTo: (offset) => engine()?.scrollTo(offset),
        getCheckedKeys: (leafOnly) => (engine()?.getCheckedKeys(leafOnly) ?? []).map(String),
        setCheckedKeys: (keys) => engine()?.setCheckedKeys(keys),
        setExpandedKeys: (keys) => engine()?.setExpandedKeys(keys),
        filter: (query) => engine()?.filter(query),
    };
}

export const elTreeV2Adapter: VirtualTreeAdapter = {
    name: 'el-tree-v2',
    component: ElTreeV2 as Component,
    getProps,
    getInstance,
};
