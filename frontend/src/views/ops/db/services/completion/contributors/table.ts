import { languages } from 'monaco-editor';
import type { DbTableInfo } from '../../../types';
import type { SqlCompletionContext, SuggestionContributor } from '../types';
import { buildTableSuggestion } from '../format';

/**
 * 当前库表名联想贡献者
 */
export const tableContributor: SuggestionContributor = {
    name: 'table',
    async contribute(ctx: SqlCompletionContext) {
        // 字段/表达式位置（SELECT/WHERE/ON/SET 等）不产出表名建议，避免噪音
        if (ctx.clause === 'column') {
            return undefined;
        }
        const tables = await ctx.dbInst.loadTables(ctx.db);
        const suggestions: languages.CompletionItem[] = (tables ?? []).map((tableMeta: DbTableInfo, index: number) =>
            buildTableSuggestion(tableMeta, index, ctx.range, ctx.quoteIdentifier)
        );
        return { suggestions };
    },
};
