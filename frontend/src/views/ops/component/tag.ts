import { ContextmenuItem } from '@/components/contextmenu';
import { markRaw, type Component } from 'vue';

export class TagTreeNode {
    /**
     * 节点id
     */
    key: string | number;

    /**
     * 节点名称
     */
    label: string;

    /**
     * 节点名称备注（用于元素title属性）
     */
    labelRemark: string;

    /**
     * 树节点类型
     */
    type: NodeType;

    /**
     * 是否为叶子节点
     */
    isLeaf: boolean = false;

    /**
     * 是否禁用状态
     */
    disabled: boolean = false;

    /**
     * 额外需要传递的参数
     */
    params: Record<string, unknown>;

    icon: { name: string; color?: string };

    // 节点组件
    nodeComponent?: Component;

    static TagPath = -1;

    constructor(key: string | number, label: string, type?: NodeType) {
        this.key = key;
        this.label = label;
        this.type = type || new NodeType(TagTreeNode.TagPath);
    }

    static new(parent: TagTreeNode, key: string | number, label: string, type?: NodeType) {
        return new TagTreeNode(key, label, type);
    }

    withLabelRemark(labelRemark: string) {
        this.labelRemark = labelRemark;
        return this;
    }

    withIsLeaf(isLeaf: boolean) {
        this.isLeaf = isLeaf;
        return this;
    }

    withDisabled(disabled: boolean) {
        this.disabled = disabled;
        return this;
    }

    withParams(params: Record<string, unknown>) {
        this.params = params;
        return this;
    }

    withIcon(icon: { name: string; color?: string }) {
        this.icon = icon;
        return this;
    }

    withNodeComponent(component: Component) {
        this.nodeComponent = markRaw(component);
        return this;
    }

    /**
     * 加载子节点，使用节点类型的loadNodesFunc去加载子节点
     * @returns 子节点信息
     */
    async loadChildren() {
        if (this.isLeaf) {
            return [];
        }
        if (this.type && this.type.loadNodesFunc) {
            return await this.type.loadNodesFunc(this);
        }
        return [];
    }
}

/**
 * 节点类型，用于加载子节点及点击事件等
 */
export class NodeType {
    /**
     * 节点类型值
     */
    value: number | string;

    contextMenuItems: ContextmenuItem[];

    loadNodesFunc: (parentNode: TagTreeNode) => Promise<TagTreeNode[]>;

    /**
     * 节点点击事件
     */
    nodeClickFunc: (node: TagTreeNode) => void;

    // 节点双击事件
    nodeDblclickFunc?: (node: TagTreeNode) => void;

    /**
     * 折叠时是否删除子节点以释放内存（展开时重新懒加载）
     */
    collapseRemoveChildren: boolean = false;

    constructor(value: number | string) {
        this.value = value;
    }

    /**
     * 赋值加载子节点回调函数
     * @param func 加载子节点回调函数
     * @returns this
     */
    withLoadNodesFunc(func: (parentNode: TagTreeNode) => Promise<TagTreeNode[]>) {
        this.loadNodesFunc = func;
        return this;
    }

    /**
     * 赋值节点点击事件回调函数
     * @param func 节点点击事件回调函数
     * @returns this
     */
    withNodeClickFunc(func: (node: TagTreeNode) => void) {
        this.nodeClickFunc = func;
        return this;
    }

    /**
     * 赋值节点双击事件回调函数
     * @param func 节点双击事件回调函数
     * @returns this
     */
    withNodeDblclickFunc(func: (node: TagTreeNode) => void) {
        this.nodeDblclickFunc = func;
        return this;
    }

    /**
     * 赋值右击菜单按钮选项
     * @param contextMenuItems 右击菜单按钮选项
     * @returns this
     */
    withContextMenuItems(contextMenuItems: ContextmenuItem[]) {
        this.contextMenuItems = contextMenuItems;
        return this;
    }

    /**
     * 设置折叠时是否删除子节点
     */
    withCollapseRemoveChildren(collapseRemoveChildren: boolean = true) {
        this.collapseRemoveChildren = collapseRemoveChildren;
        return this;
    }
}

export function expandCodePath(codePath: string) {
    const parts = codePath.split('/');
    const result = [];
    let currentPath = '';

    for (let i = 0; i < parts.length - 1; i++) {
        currentPath += parts[i] + '/';
        result.push(currentPath);
    }

    return result;
}
