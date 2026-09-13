import { languages } from '@/components/monaco/setup';
import type { EditorCompletionItem } from '../../dialect';
import type { SqlCompletionContext, SuggestionContributor } from '../types';

/**
 * 关键字/操作符/函数/变量联想贡献者（方言 getEditorCompletions() 驱动）
 */
export const keywordContributor: SuggestionContributor = {
    name: 'keyword',
    async contribute(ctx: SqlCompletionContext) {
        // 按需加载：首次补全才拉取 monaco 语言定义，之后由方言自身缓存
        const { keywords, operators, functions, variables } = await ctx.dialect.getEditorCompletions();
        const suggestions: languages.CompletionItem[] = [];

        // 空值兜底：方言漏配任一类建议时降级为空列表，不阻断整个补全。
        // label 用裸名（图标已表达类型，右侧不放 keyword/operator 等类型词）
        const push = (completions: EditorCompletionItem[] | undefined, kind: languages.CompletionItemKind) => {
            (completions ?? []).forEach((item: EditorCompletionItem) => {
                const { label, insertText } = item;
                suggestions.push({
                    label,
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
