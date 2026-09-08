import type { languages } from 'monaco-editor';
import type { DbDialect } from '../../dialect';
// DbType 为纯常量，从独立文件导入，避免依赖 dialect/index 的模块循环（TDZ）
import { DbType } from '../../dialect/dbType';

/**
 * 标识符引用符对，如 mysql 的 ` 与 mssql 的 [ ]
 */
export interface QuotePair {
    open: string;
    close: string;
}

/**
 * 各方言接受的标识符引用符（首项为主引用符，其余为该方言兼容形式）。
 * 新增方言时在此注册即可，无需改动引用检测与包裹逻辑（开闭原则）。
 */
const dialectQuotePairs: Record<string, QuotePair[]> = {
    [DbType.mysql]: [{ open: '`', close: '`' }],
    [DbType.mariadb]: [{ open: '`', close: '`' }],
    [DbType.clickhouse]: [{ open: '`', close: '`' }],
    // sqlite 同时接受双引号与反引号
    [DbType.sqlite]: [
        { open: '"', close: '"' },
        { open: '`', close: '`' },
    ],
    // mssql 支持 [ ] 与 QUOTED_IDENTIFIER 下的双引号
    [DbType.mssql]: [
        { open: '[', close: ']' },
        { open: '"', close: '"' },
    ],
};

/** 标准 SQL 引用符（双引号），未注册方言（postgres 系、oracle、dm 等）的默认值 */
const defaultQuotePairs: QuotePair[] = [{ open: '"', close: '"' }];

/**
 * 各方言常见的引用符全集（反引号/双引号/方括号），
 * 用于解析用户输入中的引用形态（如带引号的别名点触发 `users`.），与具体方言无关
 */
export const commonQuotePairs: QuotePair[] = [
    { open: '`', close: '`' },
    { open: '"', close: '"' },
    { open: '[', close: ']' },
];

/**
 * 获取指定方言接受的标识符引用符列表
 * @param dbType 数据库类型
 */
export function getDialectQuotePairs(dbType: string): QuotePair[] {
    return dialectQuotePairs[dbType] ?? defaultQuotePairs;
}

/**
 * 判断光标所在单词 [startColumn, endColumn)（monaco 1-based 列）两侧是否已被引用符包裹。
 * 用于修复补全二次包裹问题：如 `` `id` `` 中双击 id 触发补全时，插入文本应使用裸名而非再包裹一次。
 *
 * @param lineContent 当前行内容
 * @param startColumn 单词起始列（1-based，含）
 * @param endColumn 单词结束列（1-based，不含）
 * @param pairs 方言引用符列表
 * @returns 命中的引用符，未包裹时返回 undefined
 */
export function findWrappingQuote(lineContent: string, startColumn: number, endColumn: number, pairs: QuotePair[]): QuotePair | undefined {
    const charBefore = lineContent[startColumn - 2];
    const charAfter = lineContent[endColumn - 1];
    return pairs.find((p) => charBefore === p.open && charAfter === p.close);
}

/**
 * 去除名称首尾的引用符（幂等保护），
 * 防止对已被包裹的名称再次包裹产生 `` ``id`` `` 等双层引用
 *
 * @param name 表名、字段名等
 * @param pairs 方言引用符列表
 */
export function stripIdentifierQuotes(name: string, pairs: QuotePair[]): string {
    let result = (name ?? '').trim();
    for (const p of pairs) {
        if (result.length > 1 && result.startsWith(p.open) && result.endsWith(p.close)) {
            result = result.slice(1, -1);
            break;
        }
    }
    return result;
}

/**
 * 创建补全场景专用的标识符引用器，统一引用策略（单一出口，便于扩展与维护）：
 * 1. 光标处标识符已被引用符包裹时插入裸名，避免 `` `id` `` -> `` ``id`` `` 的二次包裹；
 * 2. 包裹前先幂等去包裹，兼容名称自带引用符的情况。
 *
 * @param dialect 数据库方言，提供 quoteIdentifier 能力
 * @param quotePairs 方言引用符列表
 * @param wordQuoted 光标处单词是否已被引用符包裹
 */
export function createCompletionQuoter(dialect: DbDialect, quotePairs: QuotePair[], wordQuoted: boolean): (name: string) => string {
    return (name: string): string => {
        const bare = stripIdentifierQuotes(name, quotePairs);
        return wordQuoted ? bare : dialect.quoteIdentifier(bare);
    };
}

/**
 * 对透传的标识符类建议列表（表/字段/库名）统一应用引用器。
 * 所有标识符建议路径（含 DbInst 返回的裸名列表）都必须经过本函数或 ctx.quoteIdentifier，
 * 确保引用策略唯一出口：既不遗漏包裹，也不会在已包裹场景二次包裹。
 * 注意：仅可用于标识符类建议，关键字/函数等建议不可套用。
 *
 * @param quoter 补全引用器（ctx.quoteIdentifier）
 * @param suggestions 标识符类建议列表
 */
export function quoteSuggestions(quoter: (name: string) => string, suggestions: languages.CompletionItem[]): languages.CompletionItem[] {
    return suggestions.map((s) => (s.insertText ? { ...s, insertText: quoter(s.insertText) } : s));
}
