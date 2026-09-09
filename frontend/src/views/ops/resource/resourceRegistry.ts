import type { ResourceConfig } from './resource';

/**
 * 资源注册中心（纯模块，无副作用依赖，可独立单测）
 *
 * 资源模块通过 ops/**\/resource/index.ts 默认导出 ResourceConfig，
 * 由 resource.ts 的 import.meta.glob 自动收集后调用 registerResource 注册。
 * 新增资源类型只需新建资源模块，无需修改本模块（开闭原则）。
 */
const resources = new Map<number | string, ResourceConfig>();

export function registerResource(type: number | string, rc: ResourceConfig) {
    // 重复注册会静默覆盖已有配置（与 dbm 注册表同名覆盖同类问题），开发环境显式告警
    if (import.meta.env.DEV && resources.has(type)) {
        console.warn(`[resource] 资源类型 ${String(type)} 重复注册，将覆盖已有配置：`, resources.get(type));
    }
    resources.set(type, rc);
}

/**
 * 资源模块默认导出声明（类型即文档，替代 `as ResourceConfig` 断言）：
 * 纯函数无副作用，可安全被 glob 收集的资源模块值导入（不形成运行时循环）
 */
export function defineResourceConfig(conf: ResourceConfig): ResourceConfig {
    return conf;
}

export function getResourceTypes() {
    return Array.from(resources.keys());
}

export function getResourceConfigs(): ResourceConfig[] {
    return sortByOrder(Array.from(resources.values()));
}

export function getResourceConfig(type: number | string): ResourceConfig | undefined {
    return resources.get(type);
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
