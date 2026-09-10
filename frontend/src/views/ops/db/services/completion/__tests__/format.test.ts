import { describe, expect, it, vi } from 'vitest';

// mock monaco-editor（format 仅使用 languages 常量）
vi.mock('monaco-editor', () => ({
    languages: {
        CompletionItemKind: {
            Property: 'property',
            File: 'file',
        },
    },
}));

import { buildColumnSuggestion, buildTableSuggestion, parseColumnHint } from '../format';

describe('parseColumnHint', () => {
    it('解析 字段名 + 类型 + 注释', () => {
        expect(parseColumnHint('create_time  [datetime][创建时间]')).toEqual({
            name: 'create_time',
            type: 'datetime',
            comment: '创建时间',
        });
    });

    it('解析仅含类型的字段', () => {
        expect(parseColumnHint('id  [bigint unsigned]')).toEqual({ name: 'id', type: 'bigint unsigned', comment: '' });
    });

    it('无类型无注释时整体视为字段名', () => {
        expect(parseColumnHint('remark')).toEqual({ name: 'remark', type: '', comment: '' });
    });

    it('类型含中括号内空格（如 varchar(32)）', () => {
        expect(parseColumnHint('creator  [varchar(32)][创建人]')).toEqual({
            name: 'creator',
            type: 'varchar(32)',
            comment: '创建人',
        });
    });

    it('空值兜底', () => {
        expect(parseColumnHint('')).toEqual({ name: '', type: '', comment: '' });
        expect(parseColumnHint(undefined as unknown as string)).toEqual({ name: '', type: '', comment: '' });
    });
});

describe('buildColumnSuggestion', () => {
    const range = { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 1 };

    it('label 为裸字段名，右侧灰显 类型 · 注释', () => {
        const item = buildColumnSuggestion('create_time  [datetime][创建时间]', 2, range);
        expect(item.label).toEqual({ label: 'create_time', description: 'datetime · 创建时间' });
        expect(item.insertText).toBe('create_time');
        expect(item.sortText).toBe('102');
        expect(item.range).toBe(range);
    });

    it('仅有类型时 description 不带分隔符', () => {
        expect((buildColumnSuggestion('id  [bigint]', 0, range).label as { description: string }).description).toBe('bigint');
    });

    it('无类型无注释时 description 为空（右侧不渲染任何内容）', () => {
        expect((buildColumnSuggestion('remark', 0, range).label as { description: string }).description).toBe('');
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

    it('可通过引用器包裹插入文本（方言幂等包裹）', () => {
        const item = buildTableSuggestion({ tableName: 't_user', tableComment: '' } as never, 0, range, (n) => '`' + n + '`');
        expect(item.insertText).toBe('`t_user`');
        // 无注释时 description 为空
        expect((item.label as { description: string }).description).toBe('');
    });
});
