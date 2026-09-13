/**
 * 方言层聚合出口
 *
 * 架构（依赖单向向下，无模块循环）：
 * ```
 * index.ts（本文件，仅聚合与触发加载）
 *    ↓ import.meta.glob
 * xxx_dialect.ts（各方言实现，模块体内自注册）
 *    ↓
 * registry.ts / types.ts / dbType.ts / shared/*（零依赖内核）
 * ```
 *
 * 内核各文件职责：
 * - types.ts       方言契约、能力声明、数据类型等纯类型（含 DuplicateStrategy 枚举）
 * - dbType.ts      DbType 常量（纯值、零依赖）
 * - registry.ts    方言注册表与查询
 * - shared/utils.ts       跨方言共享的纯函数工具
 * - shared/capabilities.ts 能力缺省值与引用符/切割语义预设
 * - shared/defaultRows.ts  默认审计字段预设
 *
 * **新增方言流程（开闭原则，只需 2 步、不改动本文件）**：
 * 1. 在 dbType.ts 中添加类型常量；
 * 2. 新建 `dialect/xxx_dialect.ts`，实现 DbDialect 接口、用 defineCapabilities() 声明能力，
 *    并在模块体末尾调用 registerDbDialect(DbType.xxx, new XxxDialect()) 自注册。
 *
 * 方言的全部差异（schema 支持、注释支持、引用符、SQL 切割语义、运维动作等）都由
 * getCapabilities() 自描述，调用方一律问方言、不硬编码 DbType 枚举。
 *
 * **体积约束**：方言文件禁止静态导入 monaco 语言定义（sql/mysql/pgsql 等），
 * 需要时一律在 getEditorCompletions() 内 `await import(...)` 按需加载，
 * 否则「只取图标」的页面也会被迫加载数十 KB 且与 monaco 主包重复的 tokenizer。
 */

// ==================== 触发全部方言加载（各自自注册） ====================

/**
 * 本目录下所有 `*_dialect.ts` 模块。
 *
 * eager 加载使其在模块图构建期即完成求值与自注册，故新增方言文件无需在任何地方登记。
 * 开发期据此核对文件数与注册数是否一致（见文末自检）。
 */
const dialectModules = import.meta.glob('./*_dialect.ts', { eager: true });

// ==================== 内核 re-export ====================

// 方言契约与能力声明等纯类型
export type {
    DbDialect,
    RowDefinition,
    IndexDefinition,
    DialectInfo,
    DialectCapabilities,
    QuotePair,
    SqlSplitOptions,
    SqlBlockMode,
    sqlColumnType,
    EditorCompletionItem,
    EditorCompletion,
    SqlSnippetTemplate,
    TableEditContext,
    TableInfoEditContext,
    ChangeDiff,
    SqlFormatterLanguage,
} from './types';

// 运行时值：数据类型枚举、列类型角标、通用关键字、冲突策略枚举
export { DataType, ColumnTypeSubscript, commonCustomKeywords, DuplicateStrategy } from './types';

// 数据库类型常量
export { DbType } from './dbType';

// 注册表
export { getDbDialect, getDbDialectMap, getDialectCapabilities } from './registry';

// 共享工具
export {
    appendLimitSql,
    matchType,
    matchNumericType,
    extractSchema,
    buildSchemaTable,
    getDefaultDataType,
    wrapValueDefault,
    wrapValueMssql,
    wrapValueOracle,
    QuoteEscape,
} from './shared/utils';

// 能力缺省值与词法预设
export {
    defineCapabilities,
    defaultCapabilities,
    standardQuotePairs,
    backtickQuotePairs,
    sqliteQuotePairs,
    mssqlQuotePairs,
    commonQuotePairs,
    defaultSplitOptions,
    standardSplitOptions,
    pgSplitOptions,
    mysqlSplitOptions,
    clickhouseSplitOptions,
    sqliteSplitOptions,
    mssqlSplitOptions,
    plsqlSplitOptions,
} from './shared/capabilities';

// 默认审计字段预设
export { createDefaultRows, defaultRowsConfigs, type DefaultRowsConfig } from './shared/defaultRows';

// 分页片段模板预设（方言的 getPageSnippet() 从中挑选）
export { limitCommaPageSnippet, limitOffsetPageSnippet, offsetFetchPageSnippet, rownumPageSnippet } from './shared/snippets';

// ==================== 开发期自检 ====================

import { getRegisteredDbTypes, getRegisteredDialectVersions } from './registry';

if (import.meta.env.DEV) {
    // 方言文件数与注册数不一致，说明有文件漏写自注册
    // （注册数 = 基础方言 + 版本方言，如 oracle11 只登记在版本表中）
    const moduleCount = Object.keys(dialectModules).length;
    const registeredCount = getRegisteredDbTypes().length + getRegisteredDialectVersions().length;
    if (moduleCount !== registeredCount) {
        console.warn(`[db-dialect] 方言模块 ${moduleCount} 个，已注册 ${registeredCount} 个，存在漏注册的方言文件`);
    }
}
