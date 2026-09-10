import { languages, type IRange } from 'monaco-editor';
import type { DbTableInfo } from '../../types';

/**
 * 建议项统一格式化（db.ts 与 completion 贡献者共用，单一出处）。
 *
 * 呈现策略（对齐 DataGrip 等产品）：
 * - 类型信息由左侧图标表达，右侧灰显位（label.description）只放有用的元信息：
 *   表 → 注释；字段 → 类型 · 注释；其余（库/关键字/函数/片段）不放类型词。
 * - 注意：label.description 会被 Monaco 建议组件常驻右侧渲染，
 *   顶层 detail 仅在选择态/详情面板展示，二者分工不同。
 */

export interface ColumnHintParts {
    name: string;
    type: string;
    comment: string;
}

/**
 * 解析字段提示原始串，格式：`字段名  [类型][注释]`，如 `create_time  [datetime][创建时间]`。
 */
export function parseColumnHint(raw: string): ColumnHintParts {
    const text = (raw ?? '').trim();
    const sep = text.indexOf('  ');
    const name = sep > 0 ? text.slice(0, sep).trim() : text;
    const rest = sep > 0 ? text.slice(sep) : '';
    const groups: string[] = [];
    const regex = /\[([^\]]*)\]/g;
    let matched;
    while ((matched = regex.exec(rest)) !== null) {
        groups.push(matched[1].trim());
    }
    return { name, type: groups[0] ?? '', comment: groups[1] ?? '' };
}

/**
 * 字段建议项：label 为裸字段名，右侧灰显 `类型 · 注释`。
 */
export function buildColumnSuggestion(raw: string, index: number, range: IRange): languages.CompletionItem {
    const { name, type, comment } = parseColumnHint(raw);
    const description = [type, comment].filter(Boolean).join(' · ');
    return {
        label: {
            label: name,
            description,
        },
        kind: languages.CompletionItemKind.Property,
        insertText: name,
        range,
        // 使用表字段声明顺序排序，排序需为字符串类型
        sortText: 100 + index + '',
    };
}

/**
 * 表建议项：label 为表名，右侧灰显表注释，插入文本默认裸名（调用方可用补全引用器包裹）。
 */
export function buildTableSuggestion(tableMeta: DbTableInfo, index: number, range: IRange, quoter: (name: string) => string = (name) => name): languages.CompletionItem {
    const { tableName, tableComment } = tableMeta;
    return {
        label: {
            label: tableName,
            description: tableComment ?? '',
        },
        kind: languages.CompletionItemKind.File,
        detail: tableComment ?? '',
        insertText: quoter(tableName),
        range,
        // 表名排在字段之后，排序需为字符串类型
        sortText: 300 + index + '',
    };
}
