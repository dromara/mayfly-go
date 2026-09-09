import type { Component } from 'vue';

/**
 * 资源树 v2 统一节点协议（核心对具体资产零知识）：
 * 新增资产类型 = 一个 Contributor 模块（tree/registry），核心不 import 任何具体资源。
 */

/** 节点类型标识（协议级，全局唯一字符串） */
export type NodeKind = string;

/** 树节点数据（贡献者产出，容器渲染/水合消费） */
export interface TreeNodeData {
    /** 稳定身份（kind + 业务 code，禁用时间戳/下标） */
    key: string;
    kind: NodeKind;
    /** i18n key 或最终文本 */
    label: string;
    icon?: { name: string; color?: string };
    /** 角标（key 数量、表大小等） */
    badge?: string | number;
    /** label 的 title 提示 */
    labelRemark?: string;
    disabled?: boolean;
    params?: Record<string, unknown>;
    /** 展开能力覆盖提示：显式给出时优先生效（如引用面板将库节点叶子化），否则由贡献者判定 */
    hasChildren?: boolean;
    /** 预构子树（如标签骨架，内存中即可构建的层级）；贡献者懒加载产出一般不填 */
    children?: TreeNodeData[];
}

/** 容器内节点（含水合状态）；params 由水合层兜底非空 */
export interface TreeNode extends TreeNodeData {
    params: Record<string, unknown>;
    /** 是否存在子节点（决定展开箭头） */
    hasChildren: boolean;
    /** 子节点是否已加载（懒加载水合标记） */
    loaded?: boolean;
    /** 折叠时是否释放子节点（等价旧 collapseRemoveChildren） */
    releaseOnCollapse?: boolean;
    children?: TreeNode[];
    /** 内部维护：父节点 key */
    parentKey?: string;
    /** 内部维护：水合序号（折叠释放/刷新时递增，迟到响应经此作废，防旧数据覆盖释放后的占位态） */
    loadSeq?: number;
}

/** 提供给命令/贡献者的树操作 API */
export interface TreeApi {
    /** 定位节点：按需展开祖先路径并滚动选中（未就绪时挂起待水合后自动完成） */
    locate(key: string): Promise<void>;
    /** 重新加载该节点子树（key 传空重载根） */
    refresh(key?: string): void;
    getNode(key: string): TreeNode | undefined;
}

export interface TreeCommandCtx {
    node: TreeNode;
    tree: TreeApi;
}

/** 内置占位节点类型（懒加载水合层使用） */
export const LOADING_KIND = '__loading__';
export const ERROR_KIND = '__error__';
/** 类型分组节点（资源树中"机器/数据库"等中间层，children 按资源类型路由到对应贡献者） */
export const RES_GROUP_KIND = '__res_group__';

/** 树引擎适配器契约（见 TreeEngineV2.vue）：容器只依赖此接口，换树库 = 新建适配器组件实现同契约 */
export interface TreeEngineExpose {
    /** 选中并高亮节点（配合 highlight-current 样式） */
    setCurrentKey(key: string): void;
    /** 滚动到节点（strategy: auto/center/start/end/nearest） */
    scrollToNode(key: string, strategy?: 'auto' | 'center' | 'start' | 'end' | 'nearest'): void;
}
