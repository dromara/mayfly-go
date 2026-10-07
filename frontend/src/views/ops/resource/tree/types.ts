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
    /**
     * 随节点透传给子级的数据载体（如 VO 字段、tagPath、库名）。
     * 写入整具名 VO 时需展开（`params: { ...vo }`）：interface / class 无隐式索引签名，不能直接赋给本类型；
     * 子级读取时用 `node.params as XxxNodeParams` 收窄回具体形状。
     */
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
    /** 程序化展开节点（含水合）：供节点渲染器在自身出现时自动展开并列子节点（如搜索框自动展开结果容器） */
    expandNode(key: string): Promise<void>;
    getNode(key: string): TreeNode | undefined;
    /**
     * 增量追加子节点（分页「加载更多」）：不重建既有子树、不重拉前缀，虚拟列表滚动位置不受影响。
     * replaceKey 先摘除指定直接子节点（如旧的「加载更多」占位行）再追加，保证新节点始终落在尾部。
     */
    appendChildren(key: string, nodes: TreeNodeData[], options?: { replaceKey?: string }): void;
}

/**
 * 定位解析结果（三态显式化，避免「重试 / 放弃」靠控制流隐式表达）：
 * - resolved：命中节点 key；
 * - pending：依赖的层级尚未水合，可在水合完成后重试；
 * - missing：目标不存在 / 无权限访问，应放弃并清除挂起态
 */
export type LocateOutcome = { status: 'resolved'; key: string } | { status: 'pending' } | { status: 'missing' };

/** 定位解析器可用的树能力（由容器注入，解析器无需持有树实例） */
export interface LocateAccess {
    getNode(key: string): TreeNode | undefined;
    expandNode(key: string): Promise<void>;
    /** 根是否已加载完成：为 true 时仍缺失层级即可判定 missing，避免永久挂起重试 */
    rootLoaded: boolean;
}

/**
 * 「后端资源标识 → 树节点 key」解析器契约。
 * 由资产域实现、经 prop 注入容器：容器只负责展开/高亮，不认识任何具体协议（codePath 等），
 * 新增定位协议或新增资产类型均无需改动容器（开闭原则）
 */
export type LocateResolver = (identity: string, access: LocateAccess) => Promise<LocateOutcome>;

export interface TreeCommandCtx {
    node: TreeNode;
    tree: TreeApi;
}

/** 内置占位节点类型（懒加载水合层使用） */
export const LOADING_KIND = '__loading__';
export const ERROR_KIND = '__error__';
/** 类型分组节点（资源树中"机器/数据库"等中间层，children 按资源类型路由到对应贡献者） */
export const RES_GROUP_KIND = '__res_group__';
