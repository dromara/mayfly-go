import { OptionsApi, SearchItem } from '@/components/SearchForm';
import { ContextmenuItem } from '@/components/contextmenu';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { tagApi } from '../tag/api';

export class TagTreeNode {
    /**
     * 节点id
     */
    key: any;

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
    params: any;

    icon: any;

    static TagPath = -1;

    constructor(key: any, label: string, type?: NodeType) {
        this.key = key;
        this.label = label;
        this.type = type || new NodeType(TagTreeNode.TagPath);
    }

    withLabelRemark(labelRemark: any) {
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

    withParams(params: any) {
        this.params = params;
        return this;
    }

    withIcon(icon: any) {
        this.icon = icon;
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
    value: number;

    contextMenuItems: ContextmenuItem[];

    loadNodesFunc: (parentNode: TagTreeNode) => Promise<TagTreeNode[]>;

    /**
     * 节点点击事件
     */
    nodeClickFunc: (node: TagTreeNode) => void;

    // 节点双击事件
    nodeDblclickFunc: (node: TagTreeNode) => void;

    constructor(value: number) {
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
}

/**
 * 获取标签搜索项配置
 * @param resourceType 资源类型
 * @returns
 */
export function getTagPathSearchItem(resourceType: any) {
    return SearchItem.select('tagPath', 'common.tag').withOptionsApi(
        OptionsApi.new(tagApi.getResourceTagPaths, { resourceType }).withConvertFn((res: any) => {
            return res.map((x: any) => {
                return {
                    label: x,
                    value: x,
                };
            });
        })
    );
}

/**
 * 根据标签路径获取对应的类型与编号数组
 * @param codePath 编号路径  tag1/tag2/1|xxx/11|yyy/
 * @returns {1: ['xxx'], 11: ['yyy']}
 */
export function getTagTypeCodeByPath(codePath: string) {
    const result: any = {};
    if (!codePath) return result;
    const parts = codePath.split('/'); // 切分字符串并保留数字和对应的值部分

    for (let part of parts) {
        if (!part) {
            continue;
        }
        let [key, value] = part.split('|'); // 分割数字和值部分
        // 如果不存在第二个参数，则说明为标签类型
        if (!value) {
            value = key;
            key = '-1';
        }
        if (!result[key]) {
            result[key] = [];
        }
        result[key].push(value);
    }

    return result;
}

/**
 * 完善标签路径信息
 * @param codePaths 标签路径
 * @returns
 */
export async function getAllTagInfoByCodePaths(codePaths: string[]) {
    if (!codePaths) return;
    const allTypeAndCode: any = {};

    for (let codePath of codePaths) {
        const typeAndCode = getTagTypeCodeByPath(codePath);
        for (let type in typeAndCode) {
            allTypeAndCode[type] = [...new Set(typeAndCode[type].concat(allTypeAndCode[type] || []))];
        }
    }

    for (let type in allTypeAndCode) {
        if (type == TagResourceTypeEnum.Tag.value) {
            continue;
        }
        const tagInfo = await tagApi.listByQuery.request({ type: type, codes: allTypeAndCode[type] });
        allTypeAndCode[type] = tagInfo;
    }

    const code2CodeInfo: any = {};
    for (let type in allTypeAndCode) {
        for (let code of allTypeAndCode[type]) {
            code2CodeInfo[`${type}|${code.code}`] = code;
        }
    }

    return code2CodeInfo;
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
