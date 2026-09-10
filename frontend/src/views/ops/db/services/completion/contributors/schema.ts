import { languages } from 'monaco-editor';
import type { SqlCompletionContext, SuggestionContributor } from '../types';

/**
 * 库名（schema）联想贡献者
 */
export const schemaContributor: SuggestionContributor = {
    name: 'schema',
    async contribute(ctx: SqlCompletionContext) {
        // 仅在表名期望位置（FROM/JOIN/INTO/UPDATE 后）产出库名建议；
        // `.` 触发为库/别名限定场景：命中库名时由 column 贡献者专属处理，
        // 命中别名时库名建议只会替换掉别名本身，属噪音，不产出
        if (ctx.clause !== 'table') {
            return undefined;
        }
        if (!ctx.dbs || ctx.dbs.length === 0) {
            return undefined;
        }
        const suggestions: languages.CompletionItem[] = ctx.dbs.map((a: string) => ({
            // 库名无附加元信息，label 用裸名即可（图标已表达类型，右侧不放类型词）
            label: a,
            kind: languages.CompletionItemKind.Folder,
            insertText: ctx.quoteIdentifier(a),
            range: ctx.range,
        }));
        return { suggestions };
    },
};
