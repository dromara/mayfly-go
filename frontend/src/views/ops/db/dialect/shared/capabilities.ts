/**
 * 方言能力声明的缺省值与共享预设
 *
 * 本文件零依赖（仅引用 ../types 的纯类型），可被任意方言安全导入而不引入模块循环。
 *
 * 设计意图：能力声明的字段较多，若每个方言都逐字段全写，新增方言时极易漏项
 * （历史上 clickhouse 就因漏登记外部的 dbType 白名单，导致表编辑入口整体消失）。
 * 这里给出「取众数 + 保守」的缺省值，方言通过 defineCapabilities() 只描述自身差异项，
 * 漏项时回落到缺省值而非静默丢失功能。
 */

import type { DialectCapabilities, QuotePair, SqlSplitOptions } from '../types';

// ==================== 标识符引用符预设 ====================

/** 标准 SQL 引用符（双引号）：postgres 系、oracle、dm 等未特化方言的缺省值 */
export const standardQuotePairs: QuotePair[] = [{ open: '"', close: '"' }];

/** 反引号引用符：mysql 系、clickhouse */
export const backtickQuotePairs: QuotePair[] = [{ open: '`', close: '`' }];

/** sqlite：同时接受双引号与反引号（兼容 MySQL 语法） */
export const sqliteQuotePairs: QuotePair[] = [
    { open: '"', close: '"' },
    { open: '`', close: '`' },
];

/** mssql：[ ] 为主引用符，QUOTED_IDENTIFIER 开启时亦接受双引号 */
export const mssqlQuotePairs: QuotePair[] = [
    { open: '[', close: ']' },
    { open: '"', close: '"' },
];

/**
 * 各方言常见的引用符全集（反引号/双引号/方括号），
 * 用于解析用户输入中的引用形态（如带引号的别名点触发 `users`.），与具体方言无关
 */
export const commonQuotePairs: QuotePair[] = [
    { open: '`', close: '`' },
    { open: '"', close: '"' },
    { open: '[', close: ']' },
];

// ==================== SQL 切割语义预设 ====================

/** 切割语义基线（全字段显式）：仅 mysql 反斜杠转义为历史默认行为，其余能力关闭 */
export const defaultSplitOptions: Required<SqlSplitOptions> = {
    backslashEscape: true,
    backtickQuote: false,
    hashComment: false,
    dollarQuote: false,
    escapeStringPrefix: false,
    nestedBlockComment: false,
    bracketQuote: false,
    altQuoteLiteral: false,
    doubleQuoteAsIdentifier: false,
    lineCommentNeedsWhitespace: false,
    executableComment: false,
    blockMode: 'none',
};

/** 标准 SQL 语义：反斜杠为普通字符、双引号引用标识符，无过程块 */
export const standardSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: false,
};

/** postgres 系（含 gauss/人大金仓/vastbase 等兼容方言）共同切割语义 */
export const pgSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: false,
    dollarQuote: true,
    escapeStringPrefix: true,
    nestedBlockComment: true,
    doubleQuoteAsIdentifier: true,
    blockMode: 'sql',
};

/** mysql 系（含 mariadb）共同切割语义 */
export const mysqlSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: true,
    backtickQuote: true,
    hashComment: true,
    lineCommentNeedsWhitespace: true,
    executableComment: true,
    blockMode: 'procedural',
};

/** clickhouse：反引号标识符 + 反斜杠转义，# 与 -- 均为行注释（-- 不要求后随空白） */
export const clickhouseSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: true,
    backtickQuote: true,
    hashComment: true,
};

/** sqlite：兼容 MySQL 的反引号与方括号引用符，但反斜杠为普通字符 */
export const sqliteSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: false,
    backtickQuote: true,
    bracketQuote: true,
    doubleQuoteAsIdentifier: true,
};

/** T-SQL：方括号标识符 + BEGIN..END/BEGIN TRY，IF、WHILE 无 END 闭合且 DECLARE 为独立语句，故不纳入过程关键字 */
export const mssqlSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: false,
    bracketQuote: true,
    doubleQuoteAsIdentifier: true,
    blockMode: 'sql',
};

/** Oracle/达梦：q- quote 字面量 + PL-SQL 过程块（IF/LOOP/WHILE 均以 END <kw> 闭合） */
export const plsqlSplitOptions: SqlSplitOptions = {
    ...defaultSplitOptions,
    backslashEscape: false,
    altQuoteLiteral: true,
    doubleQuoteAsIdentifier: true,
    blockMode: 'procedural',
};

// ==================== 能力缺省值 ====================

/**
 * 能力声明缺省值。
 *
 * 取值原则：
 * - 语法能力项取「多数方言的真实值」，减少各方言的重复声明；
 * - 功能开关项（supportsTableEdit/supportsDuplicateStrategy）默认开启，
 *   使漏声明表现为「多出一个入口」而非「功能静默丢失」，便于及早暴露；
 * - MySQL 协议兼容等少数派特性默认关闭。
 */
export const defaultCapabilities: DialectCapabilities = {
    // SQL 语法能力
    supportsSchema: true,
    supportsTableComment: true,
    supportsColumnComment: true,
    supportsIndexComment: false,
    supportsAutoIncrement: true,
    defaultIndexType: 'BTREE',
    // 协议/生态兼容
    mysqlCompatible: false,
    // 功能开关
    supportsTableEdit: true,
    supportsDuplicateStrategy: true,
    // 表编辑器交互约束
    canEditAutoIncrementOnCreate: true,
    canEditAutoIncrementOnEdit: true,
    // 连接元信息
    connectionMode: 'host_port',
    connectDescriptor: 'none',
    // 词法特征
    quotePairs: standardQuotePairs,
    sqlSplitOptions: {},
};

/**
 * 声明方言能力：合并缺省值与本方言差异项。
 *
 * 用法（方言内）：
 * ```ts
 * getCapabilities(): DialectCapabilities {
 *     return defineCapabilities({
 *         supportsSchema: false,
 *         mysqlCompatible: true,
 *         quotePairs: backtickQuotePairs,
 *         sqlSplitOptions: mysqlSplitOptions,
 *     });
 * }
 * ```
 *
 * 注意：合并基准是全局缺省值，而非父类方言的声明。
 * 继承型方言（mariadb←mysql、gauss/kingbaseEs/vastbase←postgres、oracle11←oracle）
 * 因此**不要**覆盖 getCapabilities()，否则会静默丢掉父类的全部差异项；
 * 确有差异时应覆盖具体的行为方法，而非能力声明。
 *
 * @param overrides 与缺省值不同的能力项
 */
export function defineCapabilities(overrides: Partial<DialectCapabilities>): DialectCapabilities {
    const capabilities: DialectCapabilities = { ...defaultCapabilities, ...overrides };

    // 不变式派生：无自增概念的方言（如 clickhouse），其自增列必然不可编辑。
    // 统一在此派生而不是要求方言各自声明，是为了避免 supportsAutoIncrement 与
    // canEditAutoIncrementOn* 两处并存而漂移——消费方若只看后者，就会放开非法编辑并生成错误 DDL。
    if (!capabilities.supportsAutoIncrement) {
        capabilities.canEditAutoIncrementOnCreate = false;
        capabilities.canEditAutoIncrementOnEdit = false;
    }

    return capabilities;
}
