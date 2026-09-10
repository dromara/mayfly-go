import { languages } from 'monaco-editor';
import type { SqlCompletionContext, SuggestionContributor } from '../types';
import { getSnippets } from '../snippets';

/**
 * SQL 片段模板联想贡献者：按方言产出 JOIN/INSERT/分页等高频模板。
 * 仅空格触发时产出（`.` 触发为库/别名限定场景，不混入片段）；
 * sortText 使用 'z' 前缀保证片段排在关键字/表/字段建议之后（sortText 为字符串序）。
 */
export const snippetContributor: SuggestionContributor = {
    name: 'snippet',
    async contribute(ctx: SqlCompletionContext) {
        if (ctx.isDotTrigger) {
            return undefined;
        }
        // 字段/表达式位置（SELECT/WHERE/ON 等）不产出模板，避免噪音
        if (ctx.clause === 'column') {
            return undefined;
        }

        const suggestions: languages.CompletionItem[] = getSnippets(ctx.dbType).map((snippet, index) => ({
            // label 用模板名，detail 仅在选择态/详情面板展示模板说明
            label: snippet.label,
            kind: languages.CompletionItemKind.Snippet,
            detail: snippet.description,
            insertText: snippet.body,
            insertTextRules: languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range: ctx.range,
            sortText: 'z' + index + '',
        }));

        return { suggestions };
    },
};
