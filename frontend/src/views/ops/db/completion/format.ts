import { languages, type IRange } from '@/components/monaco/setup';
import type { ColumnMetadata, DbTableInfo } from '../types';

/**
 * 建议项统一格式化（表名/字段联想共用，单一出处）。
 *
 * 呈现策略：
 * - 类型信息由左侧图标表达，右侧灰显位（label.description）只放有用的元信息：
 *   表 → 注释；字段 → 类型 · 注释；其余（库/关键字/函数/片段）不放类型词。
 * - 注意：label.description 会被 Monaco 建议组件常驻右侧渲染，
 *   顶层 detail 仅在选择态/详情面板展示，二者分工不同。
 */

/**
 * 字段建议项：label 为裸字段名，右侧灰显 `类型 · 注释`。
 * 数据来自按表结构化列元数据（c-metadata）：类型优先取完整列类型（含长度/精度），回退基础数据类型。
 */
export function buildColumnSuggestion(col: ColumnMetadata, index: number, range: IRange): languages.CompletionItem {
    const description = [col.columnType || col.dataType, col.columnComment].filter(Boolean).join(' · ');
    return {
        label: {
            label: col.columnName,
            description,
        },
        kind: languages.CompletionItemKind.Property,
        insertText: col.columnName,
        range,
        // 使用表字段声明顺序排序，排序需为字符串类型
        sortText: 100 + index + '',
    };
}

/**
 * 表建议项：label 为表名，右侧灰显表注释，插入文本为裸表名。
 * 引用符包裹由贡献者统一经 quoteSuggestions(ctx.quoteIdentifier) 完成，此处不做二次决策（见 quoter.ts）。
 */
export function buildTableSuggestion(tableMeta: DbTableInfo, index: number, range: IRange): languages.CompletionItem {
    const { tableName, tableComment } = tableMeta;
    return {
        label: {
            label: tableName,
            description: tableComment ?? '',
        },
        kind: languages.CompletionItemKind.File,
        detail: tableComment ?? '',
        insertText: tableName,
        range,
        // 表名排在字段之后，排序需为字符串类型
        sortText: 300 + index + '',
    };
}

/**
 * 视图建议项：与表同形（右侧灰显视图注释、插入裸名），但用不同图标区分——视图是虚拟表。
 * 引用符包裹同表，由贡献者统一经 quoteSuggestions 完成。
 */
export function buildViewSuggestion(view: { name: string; comment?: string }, index: number, range: IRange): languages.CompletionItem {
    return {
        label: {
            label: view.name,
            description: view.comment ?? '',
        },
        kind: languages.CompletionItemKind.Class,
        detail: view.comment ?? '',
        insertText: view.name,
        range,
        // 视图与表同档排序（表名之后、字段之后），排序需为字符串类型
        sortText: 300 + index + '',
    };
}
