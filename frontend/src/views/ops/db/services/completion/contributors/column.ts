import type { SqlCompletionContext, SuggestionContributor } from '../types';
import { resolveScopeTableContexts, resolveTableContext } from '../sqlContext';
import { quoteSuggestions } from '../quoter';

/**
 * 解析 `.` 触发的库名部分为实际库名（兼容 mssql 等库名带路径前缀形态）。
 * 未命中 dbs 时返回空串。
 */
function matchDbName(ctx: SqlCompletionContext, dbPart: string): string {
    const db = ctx.dbs?.find((a) => a?.toLowerCase() === dbPart);
    if (!db) {
        return '';
    }
    if (ctx.db.indexOf('/') > 0) {
        return ctx.db.substring(0, ctx.db.indexOf('/') + 1) + db;
    }
    return db;
}

/**
 * 表字段联想贡献者，覆盖以下场景（上下文感知，对齐 DataGrip 等产品行为）：
 * 1. `.` 触发（专属场景，命中后收敛贡献者链）：
 *    - 【库.表名联想】`.` 前是库名；
 *    - 【库.表.字段联想】`.` 前是 `db.table` 两级限定；
 *    - 【表别名.字段联想】`.` 前是表名或表别名。
 * 2. 列上下文（SELECT/WHERE/ON/SET/ORDER BY 等及括号列清单）：
 *    提示语句作用域内所有表的字段（按表出现顺序去重）。
 * 3. 表名期望位置（FROM/JOIN/INTO/UPDATE 后）不产出，由表名/库名贡献者负责。
 *
 * DbInst 返回的 insertText 为裸名，统一经 quoteSuggestions 按方言包裹，
 * 与库名/表名建议共用同一引用策略（已包裹场景插入裸名，避免二次包裹）。
 */
export const columnContributor: SuggestionContributor = {
    name: 'column',
    async contribute(ctx: SqlCompletionContext) {
        if (ctx.isDotTrigger) {
            return await contributeDotTrigger(ctx);
        }

        // 表名期望位置（FROM/JOIN/INTO/UPDATE 后）不提示字段，避免与表名建议混杂
        if (ctx.clause === 'table') {
            return undefined;
        }

        // 列上下文：提示作用域内所有表的字段（多表 JOIN/WHERE 场景对齐全作用域联想）
        const scopeTables = resolveScopeTableContexts(ctx.statement, ctx.db);
        const suggestions = [];
        const seen = new Set<string>();
        for (const tableInfo of scopeTables) {
            const res = await ctx.dbInst.loadTableColumnSuggestions(tableInfo.db, tableInfo.tableName, ctx.range);
            for (const suggestion of res.suggestions) {
                const insertText = (suggestion.insertText as string) ?? '';
                // 多表同名字段只保留首个（按表声明顺序）
                if (seen.has(insertText)) {
                    continue;
                }
                seen.add(insertText);
                suggestions.push(suggestion);
            }
        }
        if (suggestions.length === 0) {
            return undefined;
        }
        return { suggestions: quoteSuggestions(ctx.quoteIdentifier, suggestions) };
    },
};

/**
 * `.` 触发的专属场景处理：库.表.字段 > 库.表名 > 表别名.字段，逐级尝试。
 */
async function contributeDotTrigger(ctx: SqlCompletionContext) {
    // 【库.表.字段联想】`.` 前是 `db.table` 两级限定（如 db1.orders.）
    const dotIdx = ctx.dotAlias.indexOf('.');
    if (dotIdx > 0) {
        const dbName = matchDbName(ctx, ctx.dotAlias.substring(0, dotIdx));
        if (dbName) {
            const tableName = ctx.dotAlias.substring(dotIdx + 1);
            const res = await ctx.dbInst.loadTableColumnSuggestions(dbName, tableName, ctx.range);
            return { suggestions: quoteSuggestions(ctx.quoteIdentifier, res.suggestions), exclusive: true };
        }
    }

    // 【库.表名联想】`.` 前的字符串是库名
    const dbName = matchDbName(ctx, ctx.dotAlias);
    if (dbName) {
        const res = await ctx.dbInst.loadTableSuggestions(dbName, ctx.range);
        return { suggestions: quoteSuggestions(ctx.quoteIdentifier, res.suggestions), exclusive: true };
    }

    // 【表别名.字段联想】`.` 前的字符串是表名或表别名
    const tableInfo = resolveTableContext(ctx.statement, ctx.dotAlias, ctx.db);
    if (tableInfo) {
        const res = await ctx.dbInst.loadTableColumnSuggestions(tableInfo.db, tableInfo.tableName, ctx.range);
        return { suggestions: quoteSuggestions(ctx.quoteIdentifier, res.suggestions), exclusive: true };
    }

    // 未命中别名/库名，交给后续贡献者（表名、关键字等）
    return undefined;
}
