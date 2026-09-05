import { languages } from 'monaco-editor';
import type { EditorCompletionItem } from '../../../dialect';
import type { SqlCompletionContext, SuggestionContributor } from '../types';

/**
 * 关键字/操作符/函数/变量联想贡献者（方言 editorCompletions 配置驱动）
 */
export const keywordContributor: SuggestionContributor = {
    name: 'keyword',
    async contribute(ctx: SqlCompletionContext) {
        const { keywords, operators, functions, variables } = ctx.dialect.getInfo().editorCompletions;
        const suggestions: languages.CompletionItem[] = [];

        // 空值兜底：方言漏配任一类建议时降级为空列表，不阻断整个补全
        const push = (completions: EditorCompletionItem[] | undefined, kind: languages.CompletionItemKind) => {
            (completions ?? []).forEach((item: EditorCompletionItem) => {
                const { label, insertText, description } = item;
                suggestions.push({
                    label: { label, description },
                    kind,
                    insertText: insertText || label,
                    range: ctx.range,
                });
            });
        };

        push(keywords, languages.CompletionItemKind.Keyword);
        push(operators, languages.CompletionItemKind.Operator);
        push(functions, languages.CompletionItemKind.Function);
        push(variables, languages.CompletionItemKind.Variable);

        return { suggestions };
    },
};
