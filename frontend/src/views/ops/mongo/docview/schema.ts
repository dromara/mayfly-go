/**
 * 表格视图的动态列模型。
 *
 * Mongo 没有固定 schema，列只能从本次返回的文档里推断；推断结果与用户手动固定的列合并后
 * 才是最终列集。这里只做纯计算，不碰请求与状态，因此规则可以完全单测。
 */
import type { MongoDoc } from '../types';
import { displayCell, wrapperTypeOf } from './extjson';

/** 字段出现率低于该值的列默认不显示（异构集合动辄上百个字段，全列等于没有列） */
export const DEFAULT_MIN_COVERAGE = 0.5;

/** 默认最多显示多少列，超出时靠「显示更多列」按需取回 */
export const DEFAULT_MAX_COLUMNS = 8;

/** 主键列一定保留：它是定位与跳转的依据，与出现率无关 */
export const ID_FIELD = '_id';

export interface InferredColumn {
    /** 顶层字段名（点号路径用于嵌套展示，不参与推断列） */
    key: string;
    /** 该列上出现过的 BSON/JSON 类型名，用于列头标注与排序可行性判断 */
    kinds: string[];
    /** 出现率，0~1 */
    coverage: number;
}

export interface ColumnInference {
    /** 默认展示的列（含 _id） */
    columns: InferredColumn[];
    /** 被覆盖率/列数上限筛掉、但可通过「显示更多列」取回的字段 */
    optional: InferredColumn[];
}

/** 值的类型名：BSON 类型包装优先，其次 JSON 原生类型（详情视图与列头共用同一套判据） */
export function typeKindOf(value: unknown): string {
    const wrapper = wrapperTypeOf(value);
    if (wrapper) {
        return wrapper;
    }
    if (value === null || value === undefined) {
        return 'Null';
    }
    if (Array.isArray(value)) {
        return 'Array';
    }
    switch (typeof value) {
        case 'string':
            return 'String';
        case 'number':
            return 'Number';
        case 'boolean':
            return 'Boolean';
        case 'object':
            return 'Document';
        default:
            return 'Other';
    }
}

/**
 * 从一页文档推断列。
 *
 * 规则：按字段首次出现顺序统计出现率；`_id` 无条件保留；覆盖率低于阈值或超出列数上限的
 * 进入 optional 而不是丢弃 —— 数据还在，只是默认不显示，用户要看得自己选。
 */
export function inferColumns(docs: MongoDoc[], options: { minCoverage?: number; maxColumns?: number; pinned?: string[] } = {}): ColumnInference {
    const minCoverage = options.minCoverage ?? DEFAULT_MIN_COVERAGE;
    const maxColumns = options.maxColumns ?? DEFAULT_MAX_COLUMNS;
    const pinned = options.pinned ?? [];

    const order: string[] = [];
    const counts = new Map<string, number>();
    const kinds = new Map<string, string[]>();

    docs.forEach((doc) => {
        const source = doc?.doc ?? {};
        Object.keys(source).forEach((key) => {
            if (!counts.has(key)) {
                counts.set(key, 0);
                kinds.set(key, []);
                order.push(key);
            }
            counts.set(key, (counts.get(key) ?? 0) + 1);

            const kind = typeKindOf(source[key]);
            const list = kinds.get(key) ?? [];
            if (!list.includes(kind)) {
                list.push(kind);
                kinds.set(key, list);
            }
        });
    });

    const total = docs.length || 1;
    const all: InferredColumn[] = order.map((key) => ({
        key,
        kinds: kinds.get(key) ?? [],
        coverage: (counts.get(key) ?? 0) / total,
    }));

    const columns: InferredColumn[] = [];
    const optional: InferredColumn[] = [];
    const keep = (column: InferredColumn) => {
        if (!columns.some((item) => item.key === column.key)) {
            columns.push(column);
        }
    };

    for (const column of all) {
        const isId = column.key === ID_FIELD;
        const isPinned = pinned.includes(column.key);
        if (isId || isPinned || (column.coverage >= minCoverage && columns.length < maxColumns)) {
            keep(column);
        } else {
            optional.push(column);
        }
    }

    // 固定的列可能没被上面的分支纳入（覆盖率低但用户要看），这里按 pinned 顺序补齐
    pinned.forEach((key) => {
        const found = all.find((item) => item.key === key);
        if (!found) {
            return;
        }
        keep(found);
        const index = optional.findIndex((item) => item.key === key);
        if (index >= 0) {
            optional.splice(index, 1);
        }
    });

    return { columns, optional };
}

/** 取某个文档在指定列上的原始值（缺失返回 undefined，交给 displayCell 显示空） */
export function columnValue(doc: MongoDoc, key: string): unknown {
    return doc?.doc?.[key];
}

/**
 * 表头排序的三态循环：升序 → 降序 → 不排序。
 *
 * 返回的是完整的新排序文档（键序即优先级）。排序字段换掉时保留其他字段不动，
 * 否则点一下列头就会把用户配好的复合排序清掉。
 */
export function nextSort(current: Record<string, unknown>, field: string): Record<string, unknown> {
    const direction = current?.[field];
    const next: Record<string, unknown> = { ...current };

    if (direction === 1) {
        next[field] = -1;
    } else if (direction === -1) {
        delete next[field];
    } else {
        next[field] = 1;
    }
    return next;
}

/** 把点号路径的字段过滤条件写进 filter：等值匹配，沿用 Mongo 的字段路径语义 */
export function withEquality(filter: Record<string, unknown>, field: string, value: unknown): Record<string, unknown> {
    return { ...filter, [field]: value };
}

/**
 * 列宽按「这一列实际渲染出来的最长文本」计算，而不是给一个固定宽度。
 *
 * 固定宽度的问题是双向的：短值列（状态、数量）被撑成一片空白，长值列（主键、嵌套文档）
 * 又被裁掉，只能靠横向滚动条看全。思路同数据库表数据页（`views/ops/db/core/columnWidth.ts`），
 * 差别在于 Mongo 的单元格值必须先经 displayCell 变成可读形态再量——直接 `value + ''`
 * 会把嵌套文档量成 `[object Object]`，宽度就完全不对了。
 *
 * 本模块不碰 DOM：量宽由调用方注入（表头是粗体、正文是等宽，两套字体必须分别量，
 * 用同一份或用语义无关的默认字体会把「其实放得下」的值画成省略号）。
 */
/** 列宽下限：再短也要能容纳常见表头与点击区域 */
export const COLUMN_MIN_WIDTH = 88;

/** 列宽上限：一条嵌套文档可能几千字符，不设上限会把整表挤成一列可见 */
export const COLUMN_MAX_WIDTH = 420;

/** 单元格左右内间距与列边框 */
const CELL_CHROME = 20;

/** 表头除文字外的额外占位：排序方向记号、异构类型徽标、内间距 */
const HEADER_CHROME = 40;

/** 数值类 BSON/JSON 类型：这些列右对齐，位数对得整齐 */
const NUMERIC_KINDS = ['Int32', 'Number', 'NumberLong', 'Double', 'Decimal128'];

/** 参与计宽的行数上限：一屏也就几十行，全量测量只会让翻页变卡 */
const MEASURE_ROWS = 30;

/** 量宽回调 */
export type TextMeasurer = (text: string) => number;

export interface ColumnWidthMetrics {
    /** 正文（等宽字体）量宽 */
    measureCell: TextMeasurer;
    /** 表头量宽（表头字重更大，比正文宽，用正文字体量会偏窄） */
    measureHeader?: TextMeasurer;
    /** 主键列 BSON 徽标与其与文本的间距；只在真有 extjson 文档时计入 */
    idBadgeWidth?: number;
}

/**
 * 计算某一列的宽度（px）。
 *
 * 测量返回 0 视为「量不了」（字体未就绪、节点不存在），此时退回保守估算而不是把列压到下限——
 * 隐藏 tab 里的表格也会重算，量成 0 会让整表塌成一排省略号。
 */
export function columnWidthOf(column: InferredColumn, docs: MongoDoc[], metrics: ColumnWidthMetrics): number {
    const measureCell = metrics.measureCell;
    const measureHeader = metrics.measureHeader ?? measureCell;
    const isIdColumn = column.key === ID_FIELD;

    let contentWidth = 0;
    for (const doc of docs.slice(0, MEASURE_ROWS)) {
        const text = displayCell(columnValue(doc, column.key));
        // 空占位不参与：缺字段与 null 的显示宽度不代表这一列的真实内容
        if (!text || text === 'null') {
            continue;
        }
        const measured = measureCell(text);
        contentWidth = measured > 0 ? Math.max(contentWidth, measured) : contentWidth;
    }

    // 主键列的 BSON 徽标只画在 extjson 文档上，但宽度必须提前留出，否则徽标会把值挤成省略号
    if (isIdColumn && metrics.idBadgeWidth && docs.some((doc) => doc.mode === 'extjson')) {
        contentWidth += metrics.idBadgeWidth;
    }

    const headerWidth = measureHeader(column.key);
    const content = contentWidth > 0 ? contentWidth + CELL_CHROME : COLUMN_MIN_WIDTH;
    const width = Math.max(content, headerWidth > 0 ? headerWidth + HEADER_CHROME : 0, COLUMN_MIN_WIDTH);
    return Math.min(width, COLUMN_MAX_WIDTH);
}

/**
 * 该列是否整体右对齐。
 *
 * 判据是「这一列出现过的类型全是数值」：混合列（如 String 与 Number 同列）里
 * 只有部分是数字，右对齐反而对不齐。
 */
export function isNumericColumn(column: InferredColumn): boolean {
    return column.kinds.length > 0 && column.kinds.every((kind) => NUMERIC_KINDS.includes(kind));
}
