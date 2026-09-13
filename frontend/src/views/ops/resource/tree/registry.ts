import type { Component } from 'vue';

import type { NodeKind, TreeNode, TreeNodeData } from './types';

/**
 * 贡献者注册表：每类节点资产一个 Contributor，声明 kind/层级/加载/渲染/选择能力。
 * 核心容器对具体资产零知识，新增资产类型 = 新增一个 Contributor 模块。
 */

export interface TreeContributor {
    kind: NodeKind;
    /** 声明式层级：本类型节点可挂在哪些父类型下（面包屑/分组/结构推导用） */
    parentKinds?: NodeKind[];
    /** 类型分组节点文案（i18n key），声明后容器可自动生成分组层 */
    groupLabel?: string;
    /**
     * 后端资源类型（TagResourceTypeEnum/ResourceTypeEnum 值）：资源根贡献者声明，
     * 注册时自动登记「资源类型 → 根节点 kind」映射（类型分组节点展开的路由依据），
     * 贡献者自身即根类型唯一事实源，无需在 ResourceConfig 重复声明
     */
    resourceType?: number | string;
    icon?: { name: string; color?: string };
    /** 默认 false；函数形态可按节点动态判定 */
    hasChildren?: boolean | ((node: TreeNode) => boolean);
    loadChildren?: (node: TreeNode) => Promise<TreeNodeData[]>;
    /** 根资源节点列表（类型分组节点展开时调用，仅资源根 kind 声明） */
    loadRoots?: (groupNode: TreeNode) => Promise<TreeNodeData[]>;
    /** 可选自定义节点渲染器，缺省用容器通用渲染器 */
    renderer?: Component;
    /**
     * 可选行内 label 渲染器（覆盖默认 i18n 文本，行组件的图标/操作按钮/角标不变）。
     * 标签路径等特殊 label 呈现用——kind 专属渲染知识归贡献者，通用行组件保持零知识
     */
    labelRenderer?: Component;
    /**
     * 选择目标：该节点是否携带完整可选的业务实体（如物理库/凭证，选中即终态）。
     * 选择器（ResourceSelect）与引用面板（ai）的统一判定依据；函数形态可按节点参数动态判定
     */
    selectable?: boolean | ((node: TreeNode) => boolean);
    /** 折叠时释放子节点（大子树内存回收） */
    releaseOnCollapse?: boolean;
    /**
     * 标签资源 code 提取器（定位用）：返回该节点自身对应的后端标签资源 code（即 codePath 资源段 `type|code` 的 code）。
     * 声明后该 kind 即可被 codePath 定位逐层解析命中——新增资产只需在自己的贡献者声明一行，无需改动任何定位逻辑（开闭原则）。
     * ⚠️ 必须返回「本节点自身」的资源 code：凭证类节点常把父资源参数一并展开进 params，
     * 直接取 params.code 会拿到父资源 code（如凭证取到机器 code）而导致定位错命中
     */
    locateCode?: (node: TreeNode) => string | undefined;
}

const contributors = new Map<NodeKind, TreeContributor>();
/** 后端资源类型 → 根节点 kind（类型分组节点的 children 路由依据） */
const resourceTypeIndex = new Map<string, NodeKind>();

export function registerContributor(contributor: TreeContributor) {
    if (import.meta.env.DEV && contributors.has(contributor.kind)) {
        console.warn(`[tree] 节点类型 ${contributor.kind} 重复注册，将覆盖已有贡献者`);
    }
    // 资源根贡献者声明 resourceType 即自动登记类型映射（根 kind 单一事实源在贡献者）
    if (contributor.resourceType !== undefined) {
        resourceTypeIndex.set(String(contributor.resourceType), contributor.kind);
    }
    contributors.set(contributor.kind, contributor);
}

export function getContributor(kind: NodeKind): TreeContributor | undefined {
    return contributors.get(kind);
}

export function getRootContributor(resourceType: number | string): TreeContributor | undefined {
    const kind = resourceTypeIndex.get(String(resourceType));
    return kind ? contributors.get(kind) : undefined;
}

export function resolveHasChildren(contributor: TreeContributor, node: TreeNode): boolean {
    if (typeof contributor.hasChildren === 'function') {
        return contributor.hasChildren(node);
    }
    return !!contributor.hasChildren;
}

/**
 * 按「贡献者声明的 locateCode」在父节点已加载的子节点中解析目标资源节点。
 * 通用能力：只依赖 locateCode 契约，不认识任何具体资源的 key 规则；
 * 未声明 locateCode 的 kind（占位行、表/索引等非标签资源）天然不参与匹配
 */
export function findChildByLocateCode(parent: TreeNode | undefined, code: string): TreeNode | undefined {
    if (!parent?.children) {
        return undefined;
    }
    return parent.children.find((child) => contributors.get(child.kind)?.locateCode?.(child) === code);
}

/**
 * 节点是否为有效的选择目标（禁用节点不可选；未注册贡献者的 kind 不可选）。
 * 选择器 change 守卫与引用面板共用的单源判定
 */
export function isNodeSelectable(node: TreeNodeData): boolean {
    if (node.disabled) {
        return false;
    }
    const contributor = contributors.get(node.kind);
    if (!contributor?.selectable) {
        return false;
    }
    return typeof contributor.selectable === 'function' ? contributor.selectable(node as TreeNode) : true;
}

/** 选择粒度声明形态：kind 清单或自定义谓词（选择器的 selectable prop） */
export type NodeMatcher = string[] | ((node: TreeNodeData) => boolean);

/** 匹配器解析：kind 清单数组转为判定函数，函数形态原样透传 */
export function toNodeMatcher(matcher: NodeMatcher): (node: TreeNodeData) => boolean {
    return Array.isArray(matcher) ? (node) => matcher.includes(node.kind) : matcher;
}
