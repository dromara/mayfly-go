import type { languages } from 'monaco-editor';
import { getDbDialect, getDialectCapabilities } from '../dialect';
import type { DbDialect, QuotePair } from '../dialect/types';

/**
 * 获取指定方言接受的标识符引用符列表（首项为主引用符，其余为该方言兼容形式）。
 *
 * 直接读取方言能力声明，不再维护 dbType → 引用符 的平行映射表：
 * 新增方言只需在自身文件声明 quotePairs，引用检测与包裹逻辑零修改（开闭原则）。
 *
 * @param dbType 数据库类型，见 DbType
 */
export function getDialectQuotePairs(dbType: string): QuotePair[] {
    return getDialectCapabilities(getDbDialect(dbType)).quotePairs;
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
