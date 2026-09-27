import { describe, expect, it, vi } from 'vitest';

// mock monaco 装配入口（format 仅使用 languages 常量）
vi.mock('@/components/monaco/setup', () => ({
    languages: {
        CompletionItemKind: {
            Property: 'property',
            File: 'file',
        },
    },
}));

import { buildColumnSuggestion, buildTableSuggestion } from '../format';

describe('buildColumnSuggestion', () => {
    const range = { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 };

    it('label 为裸字段名，右侧灰显 类型 · 注释', () => {
        const item = buildColumnSuggestion({ columnName: 'create_time', columnType: 'datetime', dataType: 'datetime', columnComment: '创建时间' }, 2, range);
        expect(item.label).toEqual({ label: 'create_time', description: 'datetime · 创建时间' });
        expect(item.insertText).toBe('create_time');
        expect(item.sortText).toBe('102');
        expect(item.range).toBe(range);
    });

    it('仅有类型时 description 不带分隔符', () => {
        expect(
            (buildColumnSuggestion({ columnName: 'id', columnType: 'bigint', dataType: 'bigint' }, 0, range).label as { description: string }).description
        ).toBe('bigint');
    });

    it('无类型无注释时 description 为空（右侧不渲染任何内容）', () => {
        expect((buildColumnSuggestion({ columnName: 'remark', dataType: '' }, 0, range).label as { description: string }).description).toBe('');
    });

    it('columnType 缺省时回退 dataType', () => {
        expect((buildColumnSuggestion({ columnName: 'price', dataType: 'decimal' }, 0, range).label as { description: string }).description).toBe('decimal');
    });
});

describe('buildTableSuggestion', () => {
    const range = { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 };

    it('label 为裸表名，注释移至右侧灰显 description，不再拼接 " - "', () => {
        const item = buildTableSuggestion({ tableName: 't_user', tableComment: '用户表' } as never, 3, range);
        expect(item.label).toEqual({ label: 't_user', description: '用户表' });
        expect(item.insertText).toBe('t_user');
        expect(item.sortText).toBe('303');
        expect(item.detail).toBe('用户表');
    });

    it('insertText 为裸表名（引用符包裹由贡献者统一经 quoteSuggestions 完成）', () => {
        const item = buildTableSuggestion({ tableName: 't_user', tableComment: '' } as never, 0, range);
        expect(item.insertText).toBe('t_user');
        // 无注释时 description 为空
        expect((item.label as { description: string }).description).toBe('');
    });
});
