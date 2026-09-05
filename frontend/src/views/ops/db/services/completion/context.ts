import type { editor, Position, IRange } from 'monaco-editor';
import { DbInst } from '../../db';
import { getDbDialect } from '../../dialect';
import { createCompletionQuoter, findWrappingQuote, getDialectQuotePairs } from './quoter';
import { extractStatementAt, parseCursorTokens } from './sqlContext';
import type { SqlCompletionContext } from './types';

/**
 * 构建 SQL 补全上下文：
 * 1. 解析光标单词、替换范围、令牌与别名（`.` 触发识别）；
 * 2. 提取光标所在完整 SQL 语句（感知字符串/注释中的分号）；
 * 3. 检测光标单词是否已被方言引用符包裹，创建幂等标识符引用器（修复二次包裹）。
 *
 * @param model 编辑器模型
 * @param position 光标位置
 * @param dbId 数据库实例 id
 * @param db 当前库名
 * @param dbs 可切换的所有库名
 * @param dbType 数据库类型
 */
export async function buildCompletionContext(model: editor.ITextModel, position: Position, dbId: number, db: string, dbs: string[] = [], dbType: string): Promise<SqlCompletionContext> {
    const dbInst = await DbInst.getInstA(dbId);
    const dialect = getDbDialect(dbType);
    const { lineNumber, column } = position;

    // 光标所在单词及替换范围
    const word = model.getWordUntilPosition(position);
    const range: IRange = {
        startLineNumber: lineNumber,
        endLineNumber: lineNumber,
        startColumn: word.startColumn,
        endColumn: word.endColumn,
    };

    // 当前行内容
    const lineContent = model.getLineContent(lineNumber);

    // 当前行光标前文本（用于令牌与 `.` 触发解析）
    const lineTextBeforeCursor = model.getValueInRange({
        startLineNumber: lineNumber,
        startColumn: 0,
        endLineNumber: lineNumber,
        endColumn: column,
    });
    const { lastToken, secondToken, isDotTrigger, dotAlias } = parseCursorTokens(lineTextBeforeCursor);

    // 光标所在完整 SQL 语句
    const statement = extractStatementAt(model.getValue(), model.getOffsetAt(position));

    // 方言引用符与光标单词包裹状态
    const quotePairs = getDialectQuotePairs(dbType);
    const wrappingQuote = findWrappingQuote(lineContent, word.startColumn, word.endColumn, quotePairs);

    return {
        model,
        position,
        word,
        range,
        lineContent,
        statement,
        lastToken,
        secondToken,
        isDotTrigger,
        dotAlias,
        dbInst,
        db,
        dbs,
        dialect,
        quotePairs,
        isWordQuoted: !!wrappingQuote,
        quoteIdentifier: createCompletionQuoter(dialect, quotePairs, !!wrappingQuote),
    };
}
