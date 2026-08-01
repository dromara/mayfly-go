import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { i18n } from '@/i18n';
import { NodeType, TagTreeNode } from '@/views/ops/component/tag';
import type { ResourceOpCtx } from '@/views/ops/resource/resourceOp';
import { tagApi } from '@/views/ops/tag/api';
import type { Component } from 'vue';

interface TagTreeData {
    type: number;
    code: string;
    codePath: string;
    name: string;
    children?: TagTreeData[];
    [key: string]: unknown;
}

// 资源配置
export interface ResourceConfig {
    order?: number;
    resourceType: number | string; // 资源类型
    rootNodeType: NodeType; // 资源根节点类型

    // 资源管理组件配置
    manager?: {
        componentConf: {
            name: string; // 名称
            component?: Component; // 组件
            icon?: {
                name: string;
                color?: string;
            };
        }; // 组件
        countKey?: string; // 统计数key，tab展示的数字对象key
        permCode?: string; // 权限码
    };
}

// 注入 isShowActions 的 Symbol key
export const IsShowActionsKey = Symbol('isShowActions');

// 注入叶子节点类型的 Symbol key
export const LeafNodeTypesKey = Symbol('leafNodeTypes');

// 加载目录下所有资源操作组件信息
const allResources: Record<string, { default: ResourceConfig }> = import.meta.glob('../**/resource/index.ts', { eager: true });

const resources = new Map<number | string, ResourceConfig>();

export function registerResource(type: number | string, rc: ResourceConfig) {
    resources.set(type, rc);
}

export function getResourceNodeType(type: number | string): NodeType | undefined {
    init();
    return resources.get(type)?.rootNodeType;
}

export function getResourceTypes() {
    init();
    return Array.from(resources.keys());
}

export function getResourceConfigs(): ResourceConfig[] {
    init();
    return sortByOrder(Array.from(resources.values()));
}

export function getResourceConfig(type: number | string): ResourceConfig | undefined {
    init();
    return resources.get(type);
}

function init() {
    if (resources.size == 0) {
        for (const path in allResources) {
            // path => ../xxx/resource/index.ts
            // 获取默认导出的资源组件配置信息
            const resourceConf: ResourceConfig = allResources[path].default;
            registerResource(resourceConf.resourceType, resourceConf);
        }
    }
}

function sortByOrder(items: ResourceConfig[]) {
    return items.sort((a, b) => {
        if (a.order !== undefined && b.order !== undefined) {
            return a.order - b.order; // 按order字段排序
        } else if (a.order !== undefined) {
            return -1; // a有order字段，排在前面
        } else if (b.order !== undefined) {
            return 1; // b有order字段，排在前面
        } else {
            return 0; // 两个都没有order字段，保持原顺序
        }
    });
}

/**
 * 加载相关资源树节点
 */
export const loadResourceTags = async (resourceType: (number | string)[], ctx: ResourceOpCtx | null = null) => {
    const tags = await tagApi.getTagTrees.request({
        type: resourceType.join(','),
    });

    const result: TagTreeData[] = [];
    const flatten = (node: TagTreeData, namePath: string[]) => {
        const currentNamePath = [...namePath, node.name];

        if (node.type !== TagResourceTypeEnum.Tag.value) {
            return;
        }

        let hasNonMinus1Child = false;
        for (const child of node.children || []) {
            if (child.type !== TagResourceTypeEnum.Tag.value) {
                hasNonMinus1Child = true;
                break;
            }
        }

        if (hasNonMinus1Child) {
            const newNode = {
                ...node,
                children: [] as TagTreeData[],
            };
            newNode.name = currentNamePath.join('/');

            for (const child of node.children || []) {
                if (child.type !== TagResourceTypeEnum.Tag.value) {
                    const childCopy = {
                        ...child,
                        children: [] as TagTreeData[],
                    };
                    childCopy.name = [...currentNamePath, child.name].join('/');

                    for (const grandchild of child.children || []) {
                        const grandchildCopy = {
                            ...grandchild,
                        };
                        grandchildCopy.name = [...currentNamePath, child.name, grandchild.name].join('/');
                        childCopy.children.push(grandchildCopy);
                    }

                    newNode.children.push(childCopy);
                } else {
                    flatten(child, currentNamePath);
                }
            }

            result.push(newNode);
            return;
        }

        for (const child of node.children || []) {
            flatten(child, currentNamePath);
        }
    };

    for (const tree of tags) {
        flatten(tree as unknown as TagTreeData, []);
    }

    const tagNodes = [];
    for (let tag of result) {
        const tagNode = processTagNode(ctx, tag);
        tagNodes.push(tagNode);
    }
    return tagNodes;
};

const processTagNode = (ctx: ResourceOpCtx | null, tag: TagTreeData): TagTreeNode => {
    const tagNode = new TagTreeNode(tag.codePath, tag.name, tag.type as unknown as NodeType);

    if (!tag.children || !Array.isArray(tag.children) || tag.children.length == 0) {
        return tagNode;
    }

    // 子节点还是tag类型，则直接默认加载children即可
    if (tag.children[0].type == TagResourceTypeEnum.Tag.value) {
        tagNode.loadChildren = async () => {
            const childNodes = [];
            for (let child of tag.children!) {
                const childNode = processTagNode(ctx, child);
                childNodes.push(childNode);
            }
            return childNodes;
        };
        return tagNode;
    }

    // 创建中间节点， 按类型分组
    const type2Tags = new Map<number, TagTreeData[]>();
    tag.children.forEach((child: TagTreeData) => {
        if (!type2Tags.has(child.type)) {
            type2Tags.set(child.type, [child]);
            return;
        }
        type2Tags.get(child.type)!.push(child);
    });

    tagNode.loadChildren = async () => {
        const childNodes = [];

        for (let [type, children] of type2Tags) {
            // 创建中间节点
            const typeEnum = EnumValue.getEnumByValue(TagResourceTypeEnum, type);
            const intermediateNode = new TagTreeNode(`${tag.codePath}-${type}`, i18n.global.t(typeEnum?.label || '未知'), getResourceNodeType(type))
                .withIcon({
                    name: typeEnum?.extra.icon,
                    color: typeEnum?.extra.iconColor,
                })
                .withIsLeaf(false)
                .withParams({ resourceCodes: children.map((c: TagTreeData) => c.code), tagPath: tag.codePath });

            childNodes.push(intermediateNode);
        }
        return childNodes;
    };

    return tagNode;
};
