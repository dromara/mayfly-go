import type { SqlCompletionContext, SuggestionContributor } from '../types';
import { buildValueSuggestion } from '../format';
import { resolveTableContext } from '../sqlContext';
import { matchesValuePartial, resolveValueTarget, valueInsertText } from '../valueContext';

/**
 * 列值联想贡献者：在 `列 = ` / `列 IN (` / `列 LIKE ` 这类值位置给出该列出现过的值。
 *
 * 三条边界：
 * - `.` 触发（`t.` 之后）让位给字段联想，那是列名位置不是值位置；
 * - 取不到值（方言不支持、表被删、无权限）时**不接管补全**，返回 undefined 让列名等候选照常给出，
 *   绝不能因为值补全失败就把整个补全列表清空；
 * - 命中并拿到候选时 exclusive=true，避免关键字/片段候选把值淹没。
 */
export const valueContributor: SuggestionContributor = {
    name: 'value',
    async contribute(ctx: SqlCompletionContext) {
        if (ctx.isDotTrigger) {
            return undefined;
        }
        const target = resolveValueTarget(ctx.statement.slice(0, ctx.statementCursorOffset));
        if (!target) {
            return undefined;
        }

        // 限定符按别名/表名解析归属表；未限定则退回语句里的第一个表（与字段联想同一口径）
        const tableCtx = resolveTableContext(ctx.statement, target.qualifier, ctx.db);
        if (!tableCtx?.tableName) {
            return undefined;
        }

        const values = await ctx.dbInst.loadColumnValues(tableCtx.db || ctx.db, tableCtx.tableName, target.column);
        const matched = values.filter((value) => matchesValuePartial(value, target.partial));
        if (!matched.length) {
            return undefined;
        }

        return {
            exclusive: true,
            // 引号形态全部交给 valueInsertText 判定（数字裸写、已有开引号时不重复加）
            suggestions: matched.map((value, index) => buildValueSuggestion(value, target.column, index, ctx.range, valueInsertText(value, target.quoted))),
        };
    },
};
