import type { SuggestionContributor } from '../types';
import { columnContributor } from './column';
import { keywordContributor } from './keyword';
import { schemaContributor } from './schema';
import { snippetContributor } from './snippet';
import { tableContributor } from './table';

/**
 * 补全贡献者注册表
 *
 * 与 ops/resource/tree/registry.ts 采用同一套扩展约定：新增建议类型 = 新增一个贡献者模块
 * 并调用 registerContributor，无需修改补全核心逻辑（开闭原则）。
 */

/** 内置贡献者的执行顺序权重，间隔 10 便于在两者之间插入自定义贡献者 */
const BUILTIN_ORDER = {
    schema: 10,
    column: 20,
    table: 30,
    snippet: 40,
    keyword: 50,
} as const;

/** 缺省权重：自定义贡献者排在内置链之后 */
const DEFAULT_ORDER = 100;

/**
 * 贡献者链（模块级稳定数组）。
 * 注册即按权重就位，调用方直接遍历本引用，无需每次补全都重新分配数组。
 */
const chain: SuggestionContributor[] = [];

/** 贡献者名 → 权重，用于插入定位 */
const orderMap = new Map<string, number>();

function orderOf(name: string): number {
    return orderMap.get(name) ?? DEFAULT_ORDER;
}

/**
 * 注册补全贡献者。
 *
 * 同名贡献者重复注册时覆盖旧实现（便于测试与方言定制场景替换内置行为）。
 *
 * @param contributor 贡献者实现，name 作为唯一标识
 * @param order 执行顺序权重，越小越先执行。命中 exclusive 的贡献者会提前收敛整条链，
 *              故专属场景贡献者（如 `.` 触发的字段联想）须排在通用贡献者之前
 */
export function registerContributor(contributor: SuggestionContributor, order: number = DEFAULT_ORDER) {
    const existing = chain.findIndex((c) => c.name === contributor.name);
    if (existing > -1) {
        if (import.meta.env.DEV) {
            console.warn(`[db-completion] 贡献者 ${contributor.name} 重复注册，将覆盖已有实现`);
        }
        chain.splice(existing, 1);
    }

    // 按权重升序找到首个权重大于 order 的位置插入；同权重时后注册者排在后面（稳定序）
    let insertAt = chain.length;
    for (let i = 0; i < chain.length; i++) {
        if (orderOf(chain[i].name) > order) {
            insertAt = i;
            break;
        }
    }
    orderMap.set(contributor.name, order);
    chain.splice(insertAt, 0, contributor);
}

// 内置链：schema（库名）-> column（字段，含 `.` 触发的专属场景）-> table（表名）-> snippet（片段模板）-> keyword（关键字等）
registerContributor(schemaContributor, BUILTIN_ORDER.schema);
registerContributor(columnContributor, BUILTIN_ORDER.column);
registerContributor(tableContributor, BUILTIN_ORDER.table);
registerContributor(snippetContributor, BUILTIN_ORDER.snippet);
registerContributor(keywordContributor, BUILTIN_ORDER.keyword);

/**
 * 获取贡献者链（按权重升序）。
 *
 * 返回的是模块级稳定数组引用而非副本：后续 registerContributor 会自动反映到已取到的链上，
 * 同时避免每次按键补全都重新分配数组。调用方只应遍历，不得修改。
 */
export function createDefaultContributors(): readonly SuggestionContributor[] {
    return chain;
}
