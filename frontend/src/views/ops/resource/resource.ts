import { NodeType, ResourceConfig } from '@/views/ops/component/tag';

export const ResourceOpCtxKey = 'ResourceOpCtx';

// 加载目录下所有资源操作组件信息
const allResources: Record<string, any> = import.meta.glob('../**/resource/index.ts', { eager: true });

const resources = new Map<number, ResourceConfig>();

export function registerResource(type: number, rc: ResourceConfig) {
    resources.set(type, rc);
}

export function getResourceNodeType(type: number): NodeType | undefined {
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

export function getResourceConfig(type: number): ResourceConfig | undefined {
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

function sortByOrder(items: any[]) {
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
