import { getRootContributor } from './tree/registry';
import type { NodeVisibilityFilter } from './tree/filter';
import { ERROR_KIND, type TreeNodeData } from './tree/types';
import { getResourceConfigs } from './resourceRegistry';

/**
 * 标签层的资源取数策略（标签节点 loadChildren 的实现）。
 *
 * 单独成模块而非留在 resource.ts：resource.ts 是「资源模块 glob 收集 + 注册中心转发」的启动引导层，
 * 会把全部资源模块拉进依赖图；本文件只依赖两张注册表，可脱离资源模块本体独立单测。
 *
 * 三件事都在这里，且都不认识任何具体资产 kind（新增一类资产无需改动本文件）：
 * ① 标签展开的按类型并发取数；② 可筛选类型清单的推导；③ 类型筛选谓词的工厂。
 */

/**
 * 标签下按资源类型并发取回根资源，合并为一层平铺结果（无「机器/数据库」类型分组中间层）。
 *
 * 「类型」维度仍保留，但只作为合并顺序：同一类型内保持各贡献者自己的序，类型之间按骨架里
 * 出现的序——同类资源仍然相邻，去掉的只是那一行分组表头（icon 已足以分辨类型，少一层少一次点击）。
 *
 * 单个类型取数失败不拖垮整个标签：其余类型照常展示，末尾补一条错误行指向标签节点重试
 * （与 useLazyTree 的失败占位同一形状，搜索谓词也不会把它当资源命中）。
 */
export async function loadTagResources(tagPath: string, resourceTypes: (number | string)[]): Promise<TreeNodeData[]> {
    const settled = await Promise.allSettled(resourceTypes.map(async (type) => (await getRootContributor(type)?.loadRoots?.(tagPath)) ?? []));

    const rows: TreeNodeData[] = [];
    let failed = 0;
    for (const [index, result] of settled.entries()) {
        if (result.status === 'fulfilled') {
            rows.push(...result.value);
        } else {
            failed++;
            console.error(`[resource] 标签 ${tagPath} 下资源类型 ${resourceTypes[index]} 加载失败:`, result.reason);
        }
    }
    if (failed) {
        rows.push({ key: `${tagPath}__load_error`, kind: ERROR_KIND, label: 'common.retry', hasChildren: false, params: { retryTarget: tagPath } });
    }
    return rows;
}

/** 可筛选的资源类型选项（值即资源根节点 kind） */
export interface ResourceTypeOption {
    /** 资源根节点 kind：筛选判定按它比对 */
    kind: string;
    /** i18n key */
    label: string;
    icon?: { name: string; color?: string };
}

/**
 * 推导「可按哪几类筛选」= 注册了资源模块（带 manager 文案与图标）且声明了根贡献者的那些。
 *
 * 顺序取资源注册表的 order（与右侧资源管理页签同源），保证两处对「类型顺序」只有一个真源。
 * 没有根贡献者的类型不进选项——它在树上根本不占一层，选了也筛不出东西。
 */
export function deriveTypeOptions(): ResourceTypeOption[] {
    return getResourceConfigs().flatMap((conf) => {
        const kind = getRootContributor(conf.resourceType)?.kind;
        const manager = conf.manager?.componentConf;
        return kind && manager ? [{ kind, label: manager.name, icon: manager.icon }] : [];
    });
}

/** 祖先上溯深度护栏：树最深为 标签→实例→库→菜单→表（≤6 层），异常父子环不应把过滤变成死循环 */
const MAX_ANCESTOR_DEPTH = 12;

const isRecord = (value: unknown): value is Record<string, unknown> => !!value && typeof value === 'object';

/**
 * 生成「只看某一类资源」的视图级判定（与文本命中 AND 组合，作用于引擎过滤、计数与行序三处）。
 *
 * 两条规则，都零具体 kind：
 * - 标签层看骨架声明的 `params.resourceTypes`：未展开的标签自身 kind 是 'tag'，
 *   若也按「自身/祖先等于所选类型」判定，它会被引擎整批隐藏，选完类型树就空了；
 * - 资源层按 parentKey 上溯，命中所选根 kind 即通过，其下属行（凭证/库/表）随父可见。
 */
export function createTypeFilter(
    rootKind: string,
    lookupNode: (key: string) => { kind?: unknown; parentKey?: unknown } | undefined
): NodeVisibilityFilter {
    return (node) => {
        const params = node.params;
        if (isRecord(params) && Array.isArray(params.resourceTypes)) {
            return params.resourceTypes.some((type) => getRootContributor(type)?.kind === rootKind);
        }

        let kind: unknown = node.kind;
        let parentKey: unknown = node.parentKey;
        for (let depth = 0; kind !== undefined && depth < MAX_ANCESTOR_DEPTH; depth++) {
            if (kind === rootKind) {
                return true;
            }
            const parent = typeof parentKey === 'string' ? lookupNode(parentKey) : undefined;
            kind = parent?.kind;
            parentKey = parent?.parentKey;
        }
        return false;
    };
}
