import type { SqlCompletionContext, SuggestionContributor } from '../types';
import { quoteSuggestions } from '../quoter';
import { tableSuggestions } from '../suggestions';

/**
 * 当前库表名联想贡献者
 *
 * 数据读取与形状统一在 suggestions.ts，本模块只负责场景判定与方言包裹（与 column 贡献者同一套约定）。
 */
export const tableContributor: SuggestionContributor = {
    name: 'table',
    async contribute(ctx: SqlCompletionContext) {
        // 字段/表达式位置（SELECT/WHERE/ON/SET 等）不产出表名建议，避免噪音
        if (ctx.clause === 'column') {
            return undefined;
        }
        const suggestions = await tableSuggestions(ctx.dbInst, ctx.db, ctx.range);
        return { suggestions: quoteSuggestions(ctx.quoteIdentifier, suggestions) };
    },
};
