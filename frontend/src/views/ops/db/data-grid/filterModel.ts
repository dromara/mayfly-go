/**
 * 数据网格可视化过滤条件模型：条件行定义、按列类型的操作符集合、WHERE 序列化。
 *
 * DbTableFilterBuilder（popover 内条件构建器）与 DbTableDataOp（chips / 应用 / SQL 双轨）
 * 共享本契约：构建器只负责编辑条件组，序列化统一收敛在此，保证 chips 展示、
 * 应用查询与 SQL 模式回填读同一份真源。
 */
import type { DbDialect } from '@/views/ops/db/dialect';
import { DataType } from '@/views/ops/db/dialect';
import type { TableColumnDef } from '@/views/ops/db/types';

/** 条件值形态：无值（IS NULL 类）/ 单值 / 区间（BETWEEN）/ 多值（IN） */
export type FilterValueKind = 'none' | 'single' | 'range' | 'multi';

/** 列类型归化维度：决定操作符集合与值编辑器形态 */
export type FilterColumnKind = 'string' | 'number' | 'date';

export type FilterOperator = '=' | '!=' | '>' | '>=' | '<' | '<=' | 'like' | 'notLike' | 'between' | 'in' | 'isNull' | 'isNotNull';

export interface FilterOperatorDef {
    value: FilterOperator;
    /** 操作符名称 i18n key */
    labelKey: string;
    /** chips 上的符号形态（=、≠、> 等）；词状操作符无符号，chips 回退 labelKey 文案 */
    symbol?: string;
    valueKind: FilterValueKind;
    kinds: FilterColumnKind[];
    /**
     * 序列化该条件的 SQL 片段（新增操作符只需在此加一项，无需改序列化循环）：
     * name 已含方言引号包裹，wrap 已做单引号转义与类型包装；返回 null 表示跳过。
     */
    sql: (cond: TableFilterCondition, name: string, wrap: (v: string) => string) => string | null;
}

export interface TableFilterCondition {
    id: number;
    columnName: string;
    operator: FilterOperator;
    /** 单值 / 区间首端点（日期类为格式化字符串） */
    value: string;
    /** 区间（BETWEEN）第二端点 */
    value2: string;
    /** 多值（IN） */
    inValues: string[];
}

export interface TableFilterGroup {
    /** 扁平组合逻辑：所有条件同用 AND 或 OR */
    logic: 'AND' | 'OR';
    conditions: TableFilterCondition[];
}

const ALL_KINDS: FilterColumnKind[] = ['string', 'number', 'date'];
const ORDERABLE: FilterColumnKind[] = ['number', 'date'];

export const FILTER_OPERATORS: FilterOperatorDef[] = [
    { value: '=', labelKey: 'db.opEquals', symbol: '=', valueKind: 'single', kinds: ALL_KINDS, sql: (c, n, w) => `${n} = ${w(c.value)}` },
    { value: '!=', labelKey: 'db.opNotEquals', symbol: '≠', valueKind: 'single', kinds: ALL_KINDS, sql: (c, n, w) => `${n} != ${w(c.value)}` },
    { value: 'like', labelKey: 'db.opContains', valueKind: 'single', kinds: ['string'], sql: (c, n, w) => `${n} LIKE ${w(`%${c.value}%`)}` },
    { value: 'notLike', labelKey: 'db.opNotContains', valueKind: 'single', kinds: ['string'], sql: (c, n, w) => `${n} NOT LIKE ${w(`%${c.value}%`)}` },
    { value: '>', labelKey: 'db.opGreaterThan', symbol: '>', valueKind: 'single', kinds: ORDERABLE, sql: (c, n, w) => `${n} > ${w(c.value)}` },
    { value: '>=', labelKey: 'db.opGreaterEquals', symbol: '≥', valueKind: 'single', kinds: ORDERABLE, sql: (c, n, w) => `${n} >= ${w(c.value)}` },
    { value: '<', labelKey: 'db.opLessThan', symbol: '<', valueKind: 'single', kinds: ORDERABLE, sql: (c, n, w) => `${n} < ${w(c.value)}` },
    { value: '<=', labelKey: 'db.opLessEquals', symbol: '≤', valueKind: 'single', kinds: ORDERABLE, sql: (c, n, w) => `${n} <= ${w(c.value)}` },
    { value: 'between', labelKey: 'db.opBetween', valueKind: 'range', kinds: ORDERABLE, sql: (c, n, w) => `${n} BETWEEN ${w(c.value)} AND ${w(c.value2)}` },
    { value: 'in', labelKey: 'db.opIn', valueKind: 'multi', kinds: ALL_KINDS, sql: (c, n, w) => `${n} IN (${c.inValues.map(w).join(', ')})` },
    { value: 'isNull', labelKey: 'db.opIsNull', valueKind: 'none', kinds: ALL_KINDS, sql: (c, n) => `${n} IS NULL` },
    { value: 'isNotNull', labelKey: 'db.opIsNotNull', valueKind: 'none', kinds: ALL_KINDS, sql: (c, n) => `${n} IS NOT NULL` },
];

export const getOperatorDef = (operator: FilterOperator): FilterOperatorDef | undefined => FILTER_OPERATORS.find((x) => x.value === operator);

export const operatorsForKind = (kind: FilterColumnKind): FilterOperatorDef[] => FILTER_OPERATORS.filter((x) => x.kinds.includes(kind));

let nextConditionId = 1;

export const createFilterCondition = (columnName = ''): TableFilterCondition => ({
    id: nextConditionId++,
    columnName,
    operator: '=',
    value: '',
    value2: '',
    inValues: [],
});

export const createFilterGroup = (): TableFilterGroup => ({ logic: 'AND', conditions: [] });

export const cloneFilterGroup = (group: TableFilterGroup): TableFilterGroup => JSON.parse(JSON.stringify(group)) as TableFilterGroup;

/**
 * 解析列类型：展示类型常带长度/unsigned 修饰（如 `unsigned bigint(20)`），
 * 方言整串锚定匹配会将其判为 string；去括号后按空白分词逐 token 匹配取首个非 string 结果，
 * 并返回命中的 token 作为值包装类型（wrapType），保证数值值不被加引号。
 */
export const resolveColumnMeta = (column: TableColumnDef | undefined, dialect: DbDialect): { dataType: DataType; wrapType: string } => {
    const columnType = column?.columnType || column?.dataType || '';
    if (!columnType) {
        return { dataType: DataType.String, wrapType: '' };
    }
    const direct = dialect.getDataType(columnType);
    if (direct !== DataType.String) {
        return { dataType: direct, wrapType: columnType };
    }
    const stripped = columnType.replace(/\([^)]*\)/g, '').trim();
    for (const token of stripped.split(/\s+/)) {
        const tokenType = dialect.getDataType(token);
        if (tokenType !== DataType.String) {
            return { dataType: tokenType, wrapType: token };
        }
    }
    return { dataType: DataType.String, wrapType: columnType };
};

/** 列类型归化：方言 DataType → 操作符/编辑器维度 */
export const columnKind = (column: TableColumnDef | undefined, dialect: DbDialect): FilterColumnKind => {
    switch (resolveColumnMeta(column, dialect).dataType) {
        case DataType.Number:
            return 'number';
        case DataType.Date:
        case DataType.DateTime:
        case DataType.Time:
            return 'date';
        default:
            return 'string';
    }
};

/** 日期族列的值编辑器形态：date / datetime / time 三种 picker */
export const datePickerType = (column: TableColumnDef | undefined, dialect: DbDialect): 'date' | 'datetime' | 'time' | null => {
    switch (resolveColumnMeta(column, dialect).dataType) {
        case DataType.Date:
            return 'date';
        case DataType.DateTime:
            return 'datetime';
        case DataType.Time:
            return 'time';
        default:
            return null;
    }
};

export const DATE_VALUE_FORMAT: Record<'date' | 'datetime' | 'time', string> = {
    date: 'YYYY-MM-DD',
    datetime: 'YYYY-MM-DD HH:mm:ss',
    time: 'HH:mm:ss',
};

const isBlank = (value: string) => value === '' || value == null;

/** 数值列的值必须为合法数字，避免 `id = abc` 这类非法 SQL 进查询 */
const NUMERIC_VALUE_RE = /^\s*[+-]?(\d+(\.\d+)?|\.\d+)\s*$/;

const valuesValid = (kind: FilterColumnKind, values: string[]) => kind !== 'number' || values.every((v) => NUMERIC_VALUE_RE.test(v));

/**
 * 条件是否完整（可参与序列化/chips）：缺列或缺值的行视为无效，应用时剪除以免 chip 与实际 SQL 不一致；
 * kind 传入时按列类型做值合法性校验（数值列拒绝非数字文本），不传则只做结构判定。
 */
export const isConditionComplete = (cond: TableFilterCondition, kind: FilterColumnKind = 'string'): boolean => {
    if (!cond.columnName) {
        return false;
    }
    switch (getOperatorDef(cond.operator)?.valueKind) {
        case 'none':
            return true;
        case 'range':
            return !isBlank(cond.value) && !isBlank(cond.value2) && valuesValid(kind, [cond.value, cond.value2]);
        case 'multi':
            return cond.inValues.length > 0 && valuesValid(kind, cond.inValues);
        default:
            return !isBlank(cond.value) && valuesValid(kind, [cond.value]);
    }
};

/** 条件是否可用（列存在且完整）：chips 展示与序列化共用同一谓词，避免两套判定靠调用纪律维持同步 */
export const isConditionUsable = (cond: TableFilterCondition, columns: TableColumnDef[], dialect: DbDialect): boolean => {
    const column = columns.find((x) => x.columnName === cond.columnName);
    return !!column && isConditionComplete(cond, columnKind(column, dialect));
};

/**
 * 条件组 → SQL WHERE 片段（不含 WHERE 关键字）。
 * 不完整条件（缺列、缺值）静默跳过；列名经方言 quoteIdentifier 包裹避免关键字冲突。
 */
export const serializeFilterGroup = (group: TableFilterGroup, columns: TableColumnDef[], dialect: DbDialect): string => {
    const clauses: string[] = [];
    for (const cond of group.conditions) {
        const column = columns.find((x) => x.columnName === cond.columnName);
        if (!column || !isConditionComplete(cond, columnKind(column, dialect))) {
            continue;
        }
        const def = getOperatorDef(cond.operator);
        if (!def) {
            continue;
        }
        const columnType = resolveColumnMeta(column, dialect).wrapType;
        const name = dialect.quoteIdentifier(cond.columnName);
        // 单引号转义（''）：wrapValue 各方言只加引号不转义，含撇号值（如 O'Brien）会拼出非法 SQL
        const wrap = (v: string) => String(dialect.wrapValue(columnType, v.replace(/'/g, "''")));
        const clause = def.sql(cond, name, wrap);
        if (clause) {
            clauses.push(clause);
        }
    }
    return clauses.join(` ${group.logic} `);
};
