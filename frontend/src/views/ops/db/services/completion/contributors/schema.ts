import { languages } from 'monaco-editor';
import type { SqlCompletionContext, SuggestionContributor } from '../types';

/**
 * 库名（schema）联想贡献者
 */
export const schemaContributor: SuggestionContributor = {
    name: 'schema',
    async contribute(ctx: SqlCompletionContext) {
        if (!ctx.dbs || ctx.dbs.length === 0) {
            return undefined;
        }
        const suggestions: languages.CompletionItem[] = ctx.dbs.map((a: string) => ({
            label: {
                label: a,
                description: 'schema',
            },
            kind: languages.CompletionItemKind.Folder,
            insertText: ctx.quoteIdentifier(a),
            range: ctx.range,
        }));
        return { suggestions };
    },
};
