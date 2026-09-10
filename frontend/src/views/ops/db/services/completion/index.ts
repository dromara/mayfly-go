import { registerCompletionItemProvider } from '@/components/monaco/completionItemProvider';
import type { editor, languages, Position } from 'monaco-editor';
import { getSqlSplitOptions } from '../../component/sqleditor/utils/sqlParser';
import { buildCompletionContext } from './context';
import { createDefaultContributors } from './contributors';
import { resolveCursorZone } from './sqlContext';

export type { SqlCompletionContext, SuggestionContributor, ContributorResult } from './types';
export { createDefaultContributors } from './contributors';
export * from './quoter';
export * from './sqlContext';

/**
 * 注册数据库 SQL 补全提供者（库名、表、字段、关键字等联想）。
 *
 * 架构说明（对齐开闭原则）：
 * - context.ts   统一构建补全上下文（语句提取、令牌/别名解析、标识符引用状态检测）；
 * - contributors 贡献者链按序产出建议，每类建议一个贡献者，新增类型零修改核心；
 * - quoter.ts    标识符引用策略唯一出口：光标处已被方言引用符包裹时插入裸名，
 *                修复 `` `id` `` 双击补全后变为 `` ``id`` `` 的二次包裹问题。
 *
 * @param dbId 数据库实例 id
 * @param db 当前库名
 * @param dbs 该实例所有库名
 * @param dbType 数据库类型
 */
export function registerDbCompletionItemProvider(dbId: number, db: string, dbs: string[] = [], dbType: string) {
    registerCompletionItemProvider('sql', {
        // '.' 别名/库名限定；空格为通用触发；'(' 用于 INSERT INTO t ( 列清单等场景
        triggerCharacters: ['.', ' ', '('],
        provideCompletionItems: async (model: editor.ITextModel, position: Position): Promise<languages.CompletionList | null | undefined> => {
            // 字符串字面量与注释内不提供代码提示（含行中尾注释，比仅判断行首更精确）
            if (resolveCursorZone(model.getValue(), model.getOffsetAt(position), getSqlSplitOptions(dbType)) !== 'code') {
                return { suggestions: [] };
            }

            const ctx = await buildCompletionContext(model, position, dbId, db, dbs, dbType);

            // 贡献者链：命中专属场景（exclusive）时提前收敛，仅返回该结果
            let suggestions: languages.CompletionItem[] = [];
            for (const contributor of createDefaultContributors()) {
                const result = await contributor.contribute(ctx);
                if (!result) {
                    continue;
                }
                if (result.exclusive) {
                    return { suggestions: result.suggestions };
                }
                suggestions.push(...result.suggestions);
            }
            return { suggestions };
        },
    });
}
