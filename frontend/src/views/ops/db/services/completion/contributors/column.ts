import type { SqlCompletionContext, SuggestionContributor } from '../types';
import { resolveTableContext } from '../sqlContext';
import { quoteSuggestions } from '../quoter';

/**
 * 表字段联想贡献者，覆盖两类场景：
 * 1. `.` 触发（专属场景，命中后收敛贡献者链）：
 *    - 【库.表名联想】`.` 前是库名；
 *    - 【表别名.字段联想】`.` 前是表名或表别名；
 * 2. 空格触发（非专属，与表名/关键字建议合并）：提示当前语句第一个表的字段。
 *
 * DbInst 返回的 insertText 为裸名，统一经 quoteSuggestions 按方言包裹，
 * 与库名/表名建议共用同一引用策略（已包裹场景插入裸名，避免二次包裹）。
 */
export const columnContributor: SuggestionContributor = {
    name: 'column',
    async contribute(ctx: SqlCompletionContext) {
        if (ctx.isDotTrigger) {
            // 【库.表名联想】. 前的字符串是库名
            if (ctx.dbs?.some((a) => ctx.dotAlias === a?.toLowerCase())) {
                let dbName = ctx.dotAlias;
                if (ctx.db.indexOf('/') > 0) {
                    dbName = ctx.db.substring(0, ctx.db.indexOf('/') + 1) + ctx.dotAlias;
                }
                const res = await ctx.dbInst.loadTableSuggestions(dbName, ctx.range);
                return { suggestions: quoteSuggestions(ctx.quoteIdentifier, res.suggestions), exclusive: true };
            }
            // 【表别名.字段联想】. 前的字符串是表名或表别名
            const tableInfo = resolveTableContext(ctx.statement, ctx.dotAlias, ctx.db);
            if (tableInfo) {
                const res = await ctx.dbInst.loadTableColumnSuggestions(tableInfo.db, tableInfo.tableName, ctx.range);
                return { suggestions: quoteSuggestions(ctx.quoteIdentifier, res.suggestions), exclusive: true };
            }
            // 未命中别名/库名，交给后续贡献者（表名、关键字等）
            return undefined;
        }

        // 空格触发也会提示字段信息
        const tableInfo = resolveTableContext(ctx.statement, '', ctx.db);
        if (!tableInfo) {
            return undefined;
        }
        const res = await ctx.dbInst.loadTableColumnSuggestions(tableInfo.db, tableInfo.tableName, ctx.range);
        return { suggestions: quoteSuggestions(ctx.quoteIdentifier, res.suggestions) };
    },
};
