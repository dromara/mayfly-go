import { languages } from 'monaco-editor';
import type { DbTableInfo } from '../../../types';
import type { SqlCompletionContext, SuggestionContributor } from '../types';

/**
 * 当前库表名联想贡献者
 */
export const tableContributor: SuggestionContributor = {
    name: 'table',
    async contribute(ctx: SqlCompletionContext) {
        const tables = await ctx.dbInst.loadTables(ctx.db);
        const suggestions: languages.CompletionItem[] = (tables ?? []).map((tableMeta: DbTableInfo, index: number) => {
            const { tableName, tableComment } = tableMeta;
            return {
                label: {
                    label: tableName + ' - ' + tableComment,
                    description: 'table',
                },
                kind: languages.CompletionItemKind.File,
                detail: tableComment,
                insertText: ctx.quoteIdentifier(tableName),
                range: ctx.range,
                sortText: 300 + index + '', // 表名排在字段之后，排序需为字符串类型
            };
        });
        return { suggestions };
    },
};
