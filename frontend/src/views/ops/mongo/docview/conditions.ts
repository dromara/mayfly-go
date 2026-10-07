/**
 * 条件区的纯计算：把三段 JSON 文本翻译成「一眼看完」的摘要，并由现有文本派生新文本。
 *
 * 单独成模块而不是写在组件里，是因为条件区有一条硬约束：
 * chip 上显示的必须就是即将发出去的那份条件。写在组件里就会与请求拼装各写一遍，
 * 改一处漏一处，用户看到 `status: paid` 却查出来全集合。
 *
 * 摘要里的符号（`↑`/`↓`/`+`/`-`）不参与翻译：它们是排序方向与包含/排除的通用记号。
 */
import type { MongoDoc } from '../types';
import { parseDocText } from './extjson';
import { nextSort, withEquality } from './schema';

/** 条件名，取值与 API 的 filter/sort/projection 参数一致 */
export type ConditionName = 'filter' | 'sort' | 'projection';

/** tab 上承载三段条件的字段 */
export interface ConditionTexts {
    filterText: string;
    sortText: string;
    projectionText: string;
}

export interface ConditionChip {
    name: ConditionName;
    /** 生效值摘要；空串表示该条件当前未设置 */
    value: string;
}

/** 条件名 → tab 字段名 */
export const CONDITION_TEXT_KEY: Record<ConditionName, keyof ConditionTexts> = {
    filter: 'filterText',
    sort: 'sortText',
    projection: 'projectionText',
};

/** 条件名 → 语言包键（逐条列出而不是拼接，避免契约测试扫不到动态 key） */
export const CONDITION_LABEL_KEY: Record<ConditionName, string> = {
    filter: 'mongo.filter',
    sort: 'mongo.sort',
    projection: 'mongo.projection',
};

/** chip 里最多显示多少个字符：再多也读不出结构，只会把条件条挤成三行 */
export const MAX_CHIP_VALUE = 40;

const ASC_MARK = '↑';
const DESC_MARK = '↓';

function isPlainObject(value: unknown): value is Record<string, unknown> {
    return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** 空条件判定：空串、纯空白、以及等价于「不过滤」的 `{}`（`{ }` 也算） */
export function isEmptyCondition(text: string): boolean {
    const compact = (text ?? '').replace(/\s/g, '');
    return !compact || compact === '{}';
}

/** 压成单行并截断：chip 放不下整段 JSON，但必须看得出过滤的是什么 */
export function compactValue(text: string, max = MAX_CHIP_VALUE): string {
    const oneLine = (text ?? '').replace(/\s+/g, '');
    return oneLine.length > max ? `${oneLine.slice(0, max)}…` : oneLine;
}

/** 单个排序键的显示：数字与 asc/desc 用箭头，2dsphere/text 这类命名方向带冒号原样透出 */
function sortEntry(field: string, direction: unknown): string {
    if (direction === 1 || direction === 'asc' || direction === 'ascending') {
        return `${field} ${ASC_MARK}`;
    }
    if (direction === -1 || direction === 'desc' || direction === 'descending') {
        return `${field} ${DESC_MARK}`;
    }
    return `${field}: ${String(direction)}`;
}

/** 排序摘要：`qty ↑, createdAt ↓`，键序即优先级，所以不排序输出 */
export function sortSummary(text: string): string {
    if (isEmptyCondition(text)) {
        return '';
    }
    const parsed = parseDocText(text);
    // 文本非法时退回压缩原文：chip 宁可难看，也不能显示一个与请求不同的条件
    if (!isPlainObject(parsed)) {
        return compactValue(text);
    }
    return Object.entries(parsed)
        .map(([field, direction]) => sortEntry(field, direction))
        .join(', ');
}

/** 取列摘要：`+amount, buyer` / `-_id`；包含与排除混用时退回压缩原文 */
export function projectionSummary(text: string): string {
    if (isEmptyCondition(text)) {
        return '';
    }
    const parsed = parseDocText(text);
    if (!isPlainObject(parsed)) {
        return compactValue(text);
    }

    const included: string[] = [];
    const excluded: string[] = [];
    Object.entries(parsed).forEach(([field, flag]) => {
        (flag === 1 || flag === true ? included : excluded).push(field);
    });

    if (included.length && excluded.length) {
        return compactValue(text);
    }
    const parts: string[] = [];
    if (included.length) {
        parts.push(`+${included.join(', ')}`);
    }
    if (excluded.length) {
        parts.push(`-${excluded.join(', ')}`);
    }
    return parts.join(' ');
}

/** 单个条件的摘要文本 */
export function chipValue(name: ConditionName, text: string): string {
    if (name === 'sort') {
        return sortSummary(text);
    }
    if (name === 'projection') {
        return projectionSummary(text);
    }
    return isEmptyCondition(text) ? '' : compactValue(text);
}

/** 生效中的条件 chips（按「先筛后序再取列」的顺序） */
export function activeChips(texts: ConditionTexts): ConditionChip[] {
    return (Object.keys(CONDITION_TEXT_KEY) as ConditionName[])
        .map((name) => ({ name, value: chipValue(name, String(texts[CONDITION_TEXT_KEY[name]] ?? '')) }))
        .filter((chip) => chip.value !== '');
}

/** 是否设置了任何条件：空结果态据此决定要不要给「清空条件」 */
export function hasCondition(texts: ConditionTexts): boolean {
    return activeChips(texts).length > 0;
}

/** 排序文本 → 排序文档（非法或非对象时给空文档）：列头方向标记与即将发出的条件是同一份文本解出来的 */
export function parseSortDoc(text: string): Record<string, unknown> {
    const parsed = parseDocText(text);
    return isPlainObject(parsed) ? parsed : {};
}

/** 点列头后的新排序文本（三态循环由 schema.nextSort 决定，取消排序时回空串） */
export function applySort(text: string, field: string): string {
    const next = nextSort(parseSortDoc(text), field);
    return Object.keys(next).length ? JSON.stringify(next) : '';
}

/** 单元格「按此值过滤」后的新 filter 文本 */
export function applyEquality(text: string, field: string, value: unknown): string {
    const current = parseDocText(text);
    const base = isPlainObject(current) ? current : {};
    return JSON.stringify(withEquality(base, field, value));
}

/** 清空单个条件：filter 回到 `{}`（后端以此为「无过滤」的默认值），另两回空串 */
export function clearedText(name: ConditionName): string {
    return name === 'filter' ? '{}' : '';
}

/**
 * 选中行的主键过滤器。
 *
 * 取的是文档里的 `_id` 原值：含类型包装的（`{"$oid": "..."}`）会连同包装一起进数组，
 * 后端按 Extended JSON 解码，因此 ObjectId 与「长得一样的十六进制字符串」不会被混为一谈。
 */
export function idInFilter(docs: MongoDoc[]): Record<string, unknown> | null {
    const ids = docs.map((doc) => doc?.doc?._id).filter((id) => id !== undefined);
    if (!ids.length) {
        return null;
    }
    return { _id: { $in: ids } };
}
