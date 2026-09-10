import { describe, expect, it, vi } from 'vitest';
import { languages } from 'monaco-editor';

// mock monaco-editor（贡献者仅使用 languages 常量）
vi.mock('monaco-editor', () => ({
    languages: {
        CompletionItemKind: {
            Property: 'property',
            File: 'file',
            Folder: 'folder',
            Keyword: 'keyword',
            Operator: 'operator',
            Function: 'function',
            Variable: 'variable',
            Snippet: 'snippet',
        },
        CompletionItemInsertTextRule: {
            InsertAsSnippet: 4,
        },
    },
}));

import { createDefaultContributors } from '../contributors';
import { columnContributor } from '../contributors/column';
import { keywordContributor } from '../contributors/keyword';
import { schemaContributor } from '../contributors/schema';
import { snippetContributor } from '../contributors/snippet';
import { tableContributor } from '../contributors/table';
import { getSnippets } from '../snippets';
import type { SqlCompletionContext } from '../types';

/** 读取建议项 label 文本（兼容 string 与 { label, description } 两种形态） */
const labelText = (s: languages.CompletionItem): string => (typeof s.label === 'string' ? s.label : s.label.label);

/** 构建最小可用的补全上下文，dbInst 各加载方法均可按用例覆写 */
const createCtx = (overrides: Partial<SqlCompletionContext> = {}): SqlCompletionContext => {
    const dbInst = {
        loadTables: vi.fn().mockResolvedValue([]),
        loadTableSuggestions: vi.fn().mockResolvedValue({ suggestions: [] }),
        loadTableColumnSuggestions: vi.fn().mockResolvedValue({ suggestions: [] }),
    };
    const dialect = {
        getInfo: () => ({
            editorCompletions: {
                keywords: [{ label: 'SELECT', description: 'keyword' }],
                operators: [{ label: '=', description: 'operator' }],
                functions: [{ label: 'NOW()', description: 'func' }],
                variables: [{ label: '@@version', description: 'var' }],
            },
            quoteIdentifier: (name: string) => '`' + name + '`',
        }),
    };
    return {
        model: {} as SqlCompletionContext['model'],
        position: { lineNumber: 1, column: 1 } as SqlCompletionContext['position'],
        word: { startColumn: 1, endColumn: 1 } as SqlCompletionContext['word'],
        range: { startLineNumber: 1, startColumn: 1, endLineNumber: 1, endColumn: 5 },
        lineContent: '',
        statement: '',
        statementCursorOffset: 0,
        lastToken: '',
        secondToken: '',
        isDotTrigger: false,
        dotAlias: '',
        clause: 'free',
        dbInst: dbInst as unknown as SqlCompletionContext['dbInst'],
        db: 'test',
        dbs: ['test', 'db1'],
        dialect: dialect as unknown as SqlCompletionContext['dialect'],
        dbType: 'mysql',
        quotePairs: [{ open: '`', close: '`' }],
        isWordQuoted: false,
        quoteIdentifier: (name: string) => '`' + name + '`',
        ...overrides,
    } as SqlCompletionContext;
};

describe('columnContributor', () => {
    it('`.` 别名触发：按别名解析表并返回字段建议（insertText 按方言包裹）', async () => {
        const loadTableColumnSuggestions = vi.fn().mockResolvedValue({ suggestions: [{ insertText: 'id' }] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'u',
            statement: 'SELECT * FROM users u WHERE u.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableColumnSuggestions = loadTableColumnSuggestions;

        const result = await columnContributor.contribute(ctx);
        expect(result?.exclusive).toBe(true);
        expect(loadTableColumnSuggestions).toHaveBeenCalledWith('test', 'users', ctx.range);
        expect(result?.suggestions[0].insertText).toBe('`id`');
    });

    it('`.` 库名触发：返回该库的表建议', async () => {
        const loadTableSuggestions = vi.fn().mockResolvedValue({ suggestions: [{ insertText: 'orders' }] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'db1',
            statement: 'SELECT * FROM db1.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableSuggestions = loadTableSuggestions;

        const result = await columnContributor.contribute(ctx);
        expect(result?.exclusive).toBe(true);
        expect(loadTableSuggestions).toHaveBeenCalledWith('db1', ctx.range);
    });

    it('`.` 库名触发（mssql 库名前缀形态 db 带路径）', async () => {
        const loadTableSuggestions = vi.fn().mockResolvedValue({ suggestions: [] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'db1',
            db: 'id_1/root',
            dbs: ['db1'],
            statement: 'SELECT * FROM db1.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableSuggestions = loadTableSuggestions;

        await columnContributor.contribute(ctx);
        expect(loadTableSuggestions).toHaveBeenCalledWith('id_1/db1', ctx.range);
    });

    it('`.` 未命中别名/库名时交还贡献者链（返回 undefined）', async () => {
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'unknownAlias',
            statement: 'SELECT * FROM users WHERE unknownAlias.',
        });
        expect(await columnContributor.contribute(ctx)).toBeUndefined();
    });

    it('空格触发：JOIN 多表场景提示作用域内全部表字段（按表顺序去重）', async () => {
        const loadTableColumnSuggestions = vi.fn().mockImplementation((db: string, table: string) =>
            Promise.resolve({ suggestions: [{ insertText: table === 'a' ? 'id' : 'name' }, { insertText: 'id' }] })
        );
        const statement = 'SELECT * FROM a x JOIN b y ON x.id = y.id WHERE ';
        const ctx = createCtx({ statement, statementCursorOffset: statement.length, clause: 'column' });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableColumnSuggestions = loadTableColumnSuggestions;

        const result = await columnContributor.contribute(ctx);
        // 作用域内 a、b 两张表的字段均被加载
        expect(loadTableColumnSuggestions).toHaveBeenCalledWith('test', 'a', ctx.range);
        expect(loadTableColumnSuggestions).toHaveBeenCalledWith('test', 'b', ctx.range);
        // 同名字段去重，插入文本按方言包裹
        expect(result?.suggestions.map((s) => s.insertText)).toEqual(['`id`', '`name`']);
    });

    it('表名期望位置（FROM 后）不产出字段建议', async () => {
        const ctx = createCtx({ statement: 'SELECT * FROM ', statementCursorOffset: 14, clause: 'table' });
        expect(await columnContributor.contribute(ctx)).toBeUndefined();
    });

    it('空格触发：INSERT INTO 语句提示目标表字段', async () => {
        const loadTableColumnSuggestions = vi.fn().mockResolvedValue({ suggestions: [] });
        const ctx = createCtx({ statement: 'INSERT INTO users (name, ', statementCursorOffset: 26, clause: 'column' });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableColumnSuggestions = loadTableColumnSuggestions;

        await columnContributor.contribute(ctx);
        expect(loadTableColumnSuggestions).toHaveBeenCalledWith('test', 'users', ctx.range);
    });

    it('`.` 库.表.字段两级限定触发（db1.orders.）', async () => {
        const loadTableColumnSuggestions = vi.fn().mockResolvedValue({ suggestions: [{ insertText: 'order_no' }] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'db1.orders',
            statement: 'SELECT * FROM db1.orders.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableColumnSuggestions = loadTableColumnSuggestions;

        const result = await columnContributor.contribute(ctx);
        expect(result?.exclusive).toBe(true);
        expect(loadTableColumnSuggestions).toHaveBeenCalledWith('db1', 'orders', ctx.range);
        expect(result?.suggestions[0].insertText).toBe('`order_no`');
    });

    it('空格触发：无表子句时返回 undefined', async () => {
        const ctx = createCtx({ statement: 'SELECT 1', statementCursorOffset: 8 });
        expect(await columnContributor.contribute(ctx)).toBeUndefined();
    });
});

describe('schemaContributor', () => {
    it('仅在表名期望位置产出库名建议并按方言包裹', async () => {
        const ctx = createCtx({ clause: 'table' });
        const result = await schemaContributor.contribute(ctx);
        expect(result?.suggestions.map((s) => s.insertText)).toEqual(['`test`', '`db1`']);
    });

    it('非表名期望位置（free/column）不产出库名建议', async () => {
        expect(await schemaContributor.contribute(createCtx({ clause: 'free' }))).toBeUndefined();
        expect(await schemaContributor.contribute(createCtx({ clause: 'column' }))).toBeUndefined();
    });

    it('dbs 为空时返回 undefined', async () => {
        expect(await schemaContributor.contribute(createCtx({ clause: 'table', dbs: [] }))).toBeUndefined();
    });
});

describe('tableContributor', () => {
    it('产出表名建议（insertText 包裹 + 排序在字段之后）', async () => {
        const loadTables = vi.fn().mockResolvedValue([{ tableName: 'users', tableComment: '用户表' }]);
        const ctx = createCtx({ clause: 'table' });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTables = loadTables;

        const result = await tableContributor.contribute(ctx);
        expect(loadTables).toHaveBeenCalledWith('test');
        // label 为裸表名，类型词移除，注释移至右侧灰显 description
        expect(labelText(result?.suggestions[0]!)).toBe('users');
        expect((result?.suggestions[0]!.label as { description: string }).description).toBe('用户表');
        expect(result?.suggestions[0]!.insertText).toBe('`users`');
        expect(result?.suggestions[0]!.sortText).toBe('300');
    });

    it('字段/表达式位置（column 子句）不产出表名建议', async () => {
        const ctx = createCtx({ clause: 'column' });
        expect(await tableContributor.contribute(ctx)).toBeUndefined();
    });
});

describe('keywordContributor', () => {
    it('产出关键字/操作符/函数/变量建议（label 裸名，无类型词）', async () => {
        const result = await keywordContributor.contribute(createCtx());
        expect(result?.suggestions.map((s) => labelText(s))).toEqual(['SELECT', '=', 'NOW()', '@@version']);
        expect(result?.suggestions.every((s) => typeof s.label === 'string')).toBe(true);
    });

    it('方言漏配某类建议时降级为空列表，不阻断补全', async () => {
        const dialect = {
            getInfo: () => ({
                editorCompletions: {
                    keywords: [{ label: 'SELECT', description: 'keyword' }],
                    operators: undefined,
                    functions: undefined,
                    variables: undefined,
                },
            }),
        };
        const result = await keywordContributor.contribute(createCtx({ dialect: dialect as unknown as SqlCompletionContext['dialect'] }));
        expect(result?.suggestions).toHaveLength(1);
    });
});

describe('snippetContributor', () => {
    it('按方言产出片段模板（含 snippet 占位符与 InsertAsSnippet 规则）', async () => {
        const result = await snippetContributor.contribute(createCtx({ dbType: 'mysql' }));
        const labels = result?.suggestions.map((s) => labelText(s));
        expect(labels).toContain('JOIN');
        expect(labels).toContain('INSERT');
        expect(labels).toContain('SELECT PAGE (LIMIT)'); // mysql 专属 LIMIT 分页模板

        const join = result?.suggestions.find((s) => labelText(s) === 'JOIN');
        expect(join?.insertText).toContain('${1:');
        expect(join?.insertTextRules).toBe(4);
        expect(join?.sortText?.startsWith('z')).toBe(true);
    });

    it('postgres 使用 OFFSET 分页模板', async () => {
        const result = await snippetContributor.contribute(createCtx({ dbType: 'postgres' }));
        const labels = result?.suggestions.map((s) => labelText(s));
        expect(labels).toContain('SELECT PAGE (OFFSET)');
        expect(labels).not.toContain('SELECT PAGE (LIMIT)');
    });

    it('`.` 触发时不产出片段', async () => {
        expect(await snippetContributor.contribute(createCtx({ isDotTrigger: true }))).toBeUndefined();
    });

    it('字段/表达式位置（column 子句）不产出片段', async () => {
        expect(await snippetContributor.contribute(createCtx({ clause: 'column' }))).toBeUndefined();
    });

    it('未注册方言返回通用模板', () => {
        expect(getSnippets('unknownDb')).toEqual(getSnippets(''));
    });
});

describe('createDefaultContributors 贡献者链', () => {
    it('按序包含五个贡献者', () => {
        const chain = createDefaultContributors();
        expect(chain.map((c) => c.name)).toEqual(['schema', 'column', 'table', 'snippet', 'keyword']);
    });

    it('`.` 触发命中字段场景时提前收敛，仅返回 column 结果', async () => {
        const loadTableColumnSuggestions = vi.fn().mockResolvedValue({ suggestions: [{ insertText: 'id' }] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'u',
            statement: 'SELECT * FROM users u WHERE u.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTableColumnSuggestions = loadTableColumnSuggestions;

        // 模拟补全入口的贡献者链执行逻辑
        const suggestions = [];
        for (const contributor of createDefaultContributors()) {
            const result = await contributor.contribute(ctx);
            if (!result) {
                continue;
            }
            if (result.exclusive) {
                suggestions.push(...result.suggestions);
                break;
            }
            suggestions.push(...result.suggestions);
        }

        expect(suggestions).toHaveLength(1);
        expect(suggestions[0].insertText).toBe('`id`');
    });

    it('空格触发（表名期望位置）时非专属贡献者结果合并', async () => {
        const loadTables = vi.fn().mockResolvedValue([]);
        const ctx = createCtx({ statement: 'FROM ', statementCursorOffset: 5, clause: 'table' });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTables = loadTables;

        const suggestions = [];
        for (const contributor of createDefaultContributors()) {
            const result = await contributor.contribute(ctx);
            if (!result) {
                continue;
            }
            if (result.exclusive) {
                suggestions.push(...result.suggestions);
                break;
            }
            suggestions.push(...result.suggestions);
        }

        // column 未命中表返回 undefined；schema 2 条 + table 空 + snippet 3 条（mysql 含 LIMIT 分页）+ keyword 4 条
        expect(suggestions).toHaveLength(9);
        const snippetFirst = suggestions.find((s) => s.kind === languages.CompletionItemKind.Snippet);
        expect(snippetFirst?.sortText?.startsWith('z')).toBe(true);
    });

    it('字段/表达式位置（column 子句）仅关键字贡献者产出，表/库/片段被上下文过滤', async () => {
        const ctx = createCtx({ statement: 'SELECT ', statementCursorOffset: 7, clause: 'column' });

        const suggestions = [];
        for (const contributor of createDefaultContributors()) {
            const result = await contributor.contribute(ctx);
            if (!result) {
                continue;
            }
            if (result.exclusive) {
                suggestions.push(...result.suggestions);
                break;
            }
            suggestions.push(...result.suggestions);
        }

        // schema/table/snippet 均被 column 子句抑制，仅剩关键字 4 条
        expect(suggestions).toHaveLength(4);
        expect(suggestions.every((s) => s.kind !== languages.CompletionItemKind.Snippet)).toBe(true);
    });
});
