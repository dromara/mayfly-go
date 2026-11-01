/**
 * 列元数据 → 表编辑表单的列定义（单一真源）。
 *
 * 长度/小数位必须由本模块直接推导，而不是读取 `initColumns` 事后写上的 showLength：
 * 表单字段在 `initColumns` 之前就已构建（且 ER 图等路径根本不调用它），那样长度会恒为空，
 * 保存时 `MODIFY COLUMN` 丢掉参数——`varchar`/`decimal` 这类必须带参数的类型直接报 1064，
 * 用户看到的现象就是「长度、可为空都改了却没生效」。
 */
import type { ColumnDefinition } from '../dialect/types';
import type { ColumnMetadata } from '../types';

/**
 * 列类型声明里真正带 `(长度[,精度])` 参数的类型族（按小写子串匹配）：
 * char 覆盖 char/varchar/nvarchar/bpchar，binary 覆盖 binary/varbinary，
 * time 覆盖 time/datetime/timestamp/datetime2，bit 覆盖 bit/bit varying，
 * number/numeric/decimal 覆盖定点数（oracle NUMBER、MySQL decimal、pg numeric）。
 *
 * 整数与浮点不在其中：元数据给的 NUMERIC_PRECISION 对整数是显示位宽、对 float/double 是位精度，
 * 写回 DDL 会得到 `tinyint(3)` 这类无意义显示宽度，甚至 `double(22)`（MySQL 直接语法错误）。
 * 枚举/集合也不在其中（其长度是最长枚举值的字符数，不是类型参数；注意不能用 'num' 前缀匹配，
 * 否则 `enum` 会被误命中）。
 */
const LENGTH_ARG_TYPE_TOKENS = ['char', 'binary', 'time', 'bit', 'number', 'numeric', 'decimal'];

/**
 * 推导某列的「声明的」长度与小数位，无参数的类型一律返回空串（表单里即留空、DDL 里即不写）。
 */
export function declaredColumnArgs(column: ColumnMetadata): { length: string | number; numScale: string | number } {
    const type = (column.dataType || '').toLowerCase();
    if (!LENGTH_ARG_TYPE_TOKENS.some((token) => type.includes(token))) {
        return { length: '', numScale: '' };
    }

    const charLength = column.charMaxLength && column.charMaxLength > 0 ? column.charMaxLength : 0;
    const precision = column.numPrecision && column.numPrecision > 0 ? column.numPrecision : 0;
    const length = charLength || precision;
    if (!length) {
        return { length: '', numScale: '' };
    }
    const numScale = column.numScale && column.numScale > 0 ? column.numScale : '';
    return { length, numScale };
}

/** PG 默认值的类型转换后缀，如 `'abc'::text`、`0::numeric`、`'x'::character varying(10)` */
const DEFAULT_CAST_SUFFIX = /\s*::\s*[A-Za-z_][\w." ]*(\(\s*\d+\s*(,\s*\d+\s*)?\))?\s*$/;

/**
 * 元数据里的默认值文本 → 表单里可编辑的「值」。
 *
 * - 表达式默认值（后端已按方言判定）原样保留，不能去掉引号/加引号，否则函数名会变成字面量；
 * - 字面量默认值要剥掉方言的呈现外壳：pg 恒为 `'abc'::text` 形态（带类型转换后缀），
 *   MySQL 5.7 带引号、8.0 不带引号，统一还原成裸值，再由各方言的 DDL 生成器按类型加引号。
 */
export function normalizeDefaultValue(raw: string, isExpression: boolean): string {
    const text = (raw ?? '').trim();
    if (!text || isExpression) {
        return text;
    }
    const withoutCast = text.replace(DEFAULT_CAST_SUFFIX, '').trim();
    const quoted = /^'(.*)'$/s.exec(withoutCast);
    return quoted ? quoted[1].replace(/''/g, "'") : withoutCast;
}

/** 元数据里的「自动更新」子句原文 → 规范化书写形态（`on update CURRENT_TIMESTAMP` → `ON UPDATE CURRENT_TIMESTAMP`） */
export function normalizeOnUpdate(raw: unknown): string | undefined {
    const text = typeof raw === 'string' ? raw.trim() : '';
    if (!text) {
        return undefined;
    }
    // 只把关键字规范成大写，表达式原样保留（CURRENT_TIMESTAMP(3) 之类不得改写）
    return text.replace(/^on\s+update\s+/i, 'ON UPDATE ');
}

/**
 * 列元数据数组 → 表单列定义数组（编辑表、新建表回退路径共用）。
 */
export function buildColumnDefinitions(columns: ColumnMetadata[]): ColumnDefinition[] {
    return columns.map((column) => {
        const { length, numScale } = declaredColumnArgs(column);
        const isExpression = column.isExprDefault === true;
        return {
            name: column.columnName,
            oldName: column.columnName,
            type: column.dataType,
            value: normalizeDefaultValue(column.columnDefault ?? '', isExpression),
            length,
            numScale,
            nullable: column.nullable ?? false,
            isPrimaryKey: column.isPrimaryKey ?? false,
            autoIncrement: column.autoIncrement ?? false,
            comment: column.columnComment ?? '',
            valueIsExpression: isExpression,
            onUpdate: normalizeOnUpdate(column.extra?.['onUpdate']),
        };
    });
}
