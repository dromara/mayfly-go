import { TagResourceTypeEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';
import { getRootContributor, registerContributor, validateMenuCommands, RES_GROUP_KIND } from './tree';

import type { TreeNode, TreeNodeData } from './tree/types';
import { tagApi } from '@/views/ops/tag/api';
import type { Component } from 'vue';
import { getResourceTypes, registerResource } from './resourceRegistry';
import TagLabel from './TagLabel.vue';

interface TagTreeData {
    type: number;
    code: string;
    codePath: string;
    name: string;
    children?: TagTreeData[];
    [key: string]: unknown;
}

// 资源配置（资源管理面板维度；树侧的根节点 kind 由资源根贡献者自声明 resourceType，不在此重复）
export interface ResourceConfig {
    order?: number;
    resourceType: number | string; // 资源类型

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

// 加载目录下所有资源操作组件信息
const allResources: Record<string, { default: ResourceConfig }> = import.meta.glob('../**/resource/index.ts', { eager: true });

// 模块加载即把 glob 收集到的资源配置注册进注册中心（注册中心逻辑见 resourceRegistry.ts）
// 注：各资源模块的贡献者/命令/菜单已在模块导入时自注册（import.meta.glob eager 先于本循环执行）
for (const path in allResources) {
    // path => ../xxx/resource/index.ts，获取默认导出的资源组件配置信息
    registerResource(allResources[path].default.resourceType, allResources[path].default);
}

/**
 * 注册表完整性自检（DEV 启动期，对标 VS Code contribution validation）：
 * 拼接断点显式暴露——资源类型无根贡献者/根贡献者缺 loadRoots，都会导致该资产在树中静默不可见
 */
function validateResourceTreeLinks(): string[] {
    const problems: string[] = [];
    for (const type of getResourceTypes()) {
        const root = getRootContributor(type);
        if (!root) {
            problems.push(`资源类型 ${type} 无根贡献者（资源模块 registerContributor 需声明 resourceType）`);
        } else if (!root.loadRoots) {
            problems.push(`资源类型 ${type} 的根贡献者 ${root.kind} 缺 loadRoots，类型分组节点无法展开出资源列表`);
        }
    }
    if (import.meta.env.DEV) {
        problems.forEach((p) => console.warn(`[resource] ${p}`));
    }
    return problems;
}

if (import.meta.env.DEV) {
    validateResourceTreeLinks();
    validateMenuCommands();
}

// 注册中心转发导出，外部使用方式不变
export { defineResourceConfig, getResourceConfig, getResourceConfigs, getResourceTypes, registerResource } from './resourceRegistry';

/**
 * 类型分组节点贡献者（"机器/数据库"等中间层）：
 * 展开时按 params.resourceType 分发到对应资源根贡献者的 loadRoots（核心协议见 tree/registry.ts）
 */
registerContributor({
    kind: RES_GROUP_KIND,
    hasChildren: true,
    loadChildren: async (node: TreeNode) => {
        const resourceType = node.params?.resourceType as number | string;
        const rootContributor = getRootContributor(resourceType);
        return (await rootContributor?.loadRoots?.(node)) ?? [];
    },
});

/**
 * 标签节点贡献者：仅声明行内 label 渲染器（'/' 路径分段高亮）。
 * 标签骨架由 loadResourceTags 内存预构（children 分支即展开能力成立），无需加载器
 */
registerContributor({
    kind: 'tag',
    labelRenderer: TagLabel,
});

/**
 * 递归拍平标签树并重写 name 为完整路径，转为统一 TreeNodeData 骨架（children 预构，内存即建）：
 * - 标签层级存在资源子节点时保留该层级，资源子节点按类型归入分组节点
 * - 纯标签子层级被拍平，其下资源子树作为独立标签层级平铺到结果中
 */
const flatten = (node: TagTreeData, namePath: string[], result: TreeNodeData[]) => {
    const currentNamePath = [...namePath, node.name];

    if (node.type !== TagResourceTypeEnum.Tag.value) {
        return;
    }

    const children = node.children || [];
    const resourceChildren = children.filter((child) => child.type !== TagResourceTypeEnum.Tag.value);
    if (resourceChildren.length > 0) {
        result.push({
            key: node.codePath,
            kind: 'tag',
            label: currentNamePath.join('/'),
            children: groupByResourceType(node.codePath, resourceChildren),
        });
    }

    // 纯标签子层级继续向下递归拍平
    children
        .filter((child) => child.type === TagResourceTypeEnum.Tag.value)
        .forEach((child) => flatten(child, currentNamePath, result));
};

/** 资源子节点按资源类型分组为类型分组节点（展开时经分组贡献者分发到对应资源模块加载） */
const groupByResourceType = (codePath: string, resourceChildren: TagTreeData[]): TreeNodeData[] => {
    const type2Children = new Map<number, TagTreeData[]>();
    resourceChildren.forEach((child) => {
        if (!type2Children.has(child.type)) {
            type2Children.set(child.type, [child]);
            return;
        }
        type2Children.get(child.type)!.push(child);
    });

    const groupNodes: TreeNodeData[] = [];
    for (const [type, children] of type2Children) {
        const typeEnum = EnumValue.getEnumByValue(TagResourceTypeEnum, type);
        groupNodes.push({
            key: `${codePath}-${type}`,
            kind: RES_GROUP_KIND,
            label: typeEnum?.label || 'common.unknown',
            icon: typeEnum?.extra.icon ? { name: typeEnum.extra.icon, color: typeEnum.extra.iconColor } : undefined,
            params: { resourceType: type, resourceCodes: children.map((c) => c.code), tagPath: codePath },
        });
    }
    return groupNodes;
};

export const loadResourceTags = async (resourceType: (number | string)[]): Promise<TreeNodeData[]> => {
    const tags = await tagApi.getTagTrees.request({
        type: resourceType.join(','),
    });

    const result: TreeNodeData[] = [];
    for (const tree of tags) {
        flatten(tree as unknown as TagTreeData, [], result);
    }
    return result;
};
