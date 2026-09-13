import { describe, expect, it, vi } from 'vitest';
import { languages } from '@/components/monaco/setup';

// mock monaco 装配入口：贡献者与断言都只用到 languages 上的枚举常量，无需真实编辑器
vi.mock('@/components/monaco/setup', () => ({
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
// 分页模板预设属方言层事实，补全层只做消费；此处引用真实预设以校验装配链路
import { limitCommaPageSnippet, limitOffsetPageSnippet } from '../../dialect/shared/snippets';
import type { SqlCompletionContext } from '../types';

/** 读取建议项 label 文本（兼容 string 与 { label, description } 两种形态） */
const labelText = (s: languages.CompletionItem): string => (typeof s.label === 'string' ? s.label : s.label.label);

/** 构建最小可用的补全上下文，dbInst 各加载方法均可按用例覆写 */
const createCtx = (overrides: Partial<SqlCompletionContext> = {}): SqlCompletionContext => {
    // dbInst 只提供数据（表信息 / 字段提示原始串），建议项形状由 suggestions.ts 负责
    const dbInst = {
        loadTables: vi.fn().mockResolvedValue([]),
        loadDbHints: vi.fn().mockResolvedValue({}),
    };
    const dialect = {
        getEditorCompletions: async () => ({
            keywords: [{ label: 'SELECT', description: 'keyword' }],
            operators: [{ label: '=', description: 'operator' }],
            functions: [{ label: 'NOW()', description: 'func' }],
            variables: [{ label: '@@version', description: 'var' }],
        }),
        quoteIdentifier: (name: string) => '`' + name + '`',
        // 分页模板由方言自描述，缺省给 mysql 系的 LIMIT 逗号形预设
        getPageSnippet: () => limitCommaPageSnippet,
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
        const loadDbHints = vi.fn().mockResolvedValue({ users: ['id  [int][]'] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'u',
            statement: 'SELECT * FROM users u WHERE u.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadDbHints = loadDbHints;

        const result = await columnContributor.contribute(ctx);
        expect(result?.exclusive).toBe(true);
        expect(loadDbHints).toHaveBeenCalledWith('test');
        expect(result?.suggestions[0].insertText).toBe('`id`');
    });

    it('`.` 库名触发：返回该库的表建议', async () => {
        const loadTables = vi.fn().mockResolvedValue([{ tableName: 'orders', tableComment: '' }]);
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'db1',
            statement: 'SELECT * FROM db1.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTables = loadTables;

        const result = await columnContributor.contribute(ctx);
        expect(result?.exclusive).toBe(true);
        expect(loadTables).toHaveBeenCalledWith('db1');
        expect(result?.suggestions[0].insertText).toBe('`orders`');
    });

    it('`.` 库名触发（mssql 库名前缀形态 db 带路径）', async () => {
        const loadTables = vi.fn().mockResolvedValue([]);
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'db1',
            db: 'id_1/root',
            dbs: ['db1'],
            statement: 'SELECT * FROM db1.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadTables = loadTables;

        await columnContributor.contribute(ctx);
        expect(loadTables).toHaveBeenCalledWith('id_1/db1');
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
        // 字段提示按库整体返回：b 表的 id 与 a 表同名
        const loadDbHints = vi.fn().mockResolvedValue({ a: ['id  [int][]'], b: ['name  [varchar][]', 'id  [int][]'] });
        const statement = 'SELECT * FROM a x JOIN b y ON x.id = y.id WHERE ';
        const ctx = createCtx({ statement, statementCursorOffset: statement.length, clause: 'column' });
        (ctx.dbInst as unknown as Record<string, unknown>).loadDbHints = loadDbHints;

        const result = await columnContributor.contribute(ctx);
        // 作用域内 a、b 两张表各自被读取一次
        expect(loadDbHints).toHaveBeenCalledTimes(2);
        // 同名字段去重，插入文本按方言包裹
        expect(result?.suggestions.map((s) => s.insertText)).toEqual(['`id`', '`name`']);
    });

    it('表名期望位置（FROM 后）不产出字段建议', async () => {
        const ctx = createCtx({ statement: 'SELECT * FROM ', statementCursorOffset: 14, clause: 'table' });
        expect(await columnContributor.contribute(ctx)).toBeUndefined();
    });

    it('空格触发：INSERT INTO 语句提示目标表字段', async () => {
        const loadDbHints = vi.fn().mockResolvedValue({ users: ['name  [varchar][]'] });
        const ctx = createCtx({ statement: 'INSERT INTO users (name, ', statementCursorOffset: 26, clause: 'column' });
        (ctx.dbInst as unknown as Record<string, unknown>).loadDbHints = loadDbHints;

        const result = await columnContributor.contribute(ctx);
        expect(loadDbHints).toHaveBeenCalledWith('test');
        expect(result?.suggestions.map((s) => s.insertText)).toEqual(['`name`']);
    });

    it('`.` 库.表.字段两级限定触发（db1.orders.）', async () => {
        const loadDbHints = vi.fn().mockResolvedValue({ orders: ['order_no  [varchar][]'] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'db1.orders',
            statement: 'SELECT * FROM db1.orders.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadDbHints = loadDbHints;

        const result = await columnContributor.contribute(ctx);
        expect(result?.exclusive).toBe(true);
        expect(loadDbHints).toHaveBeenCalledWith('db1');
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
            getEditorCompletions: async () => ({
                keywords: [{ label: 'SELECT', description: 'keyword' }],
                operators: undefined,
                functions: undefined,
                variables: undefined,
            }),
        };
        const result = await keywordContributor.contribute(createCtx({ dialect: dialect as unknown as SqlCompletionContext['dialect'] }));
        expect(result?.suggestions).toHaveLength(1);
    });
});

describe('snippetContributor', () => {
    it('产出通用模板 + 方言分页模板（含 snippet 占位符与 InsertAsSnippet 规则）', async () => {
        const result = await snippetContributor.contribute(createCtx());
        const labels = result?.suggestions.map((s) => labelText(s));
        expect(labels).toContain('JOIN');
        expect(labels).toContain('INSERT');
        expect(labels).toContain('SELECT PAGE (LIMIT)'); // 缺省方言（mysql 系）的 LIMIT 分页模板

        const join = result?.suggestions.find((s) => labelText(s) === 'JOIN');
        expect(join?.insertText).toContain('${1:');
        expect(join?.insertTextRules).toBe(4);
        expect(join?.sortText?.startsWith('z')).toBe(true);
    });

    it('分页模板随方言自描述而变（pg 系换成 OFFSET 形）', async () => {
        const ctx = createCtx({ dbType: 'postgres' });
        // 片段取自 dialect.getPageSnippet() 而非 ctx.dbType：换预设即换模板，补全层不参与决策
        (ctx.dialect as unknown as Record<string, unknown>).getPageSnippet = () => limitOffsetPageSnippet;

        const result = await snippetContributor.contribute(ctx);
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

    it('getSnippets 组装通用模板与方言分页模板，不含 dbType 分支', () => {
        const snippets = getSnippets(createCtx().dialect);
        // 通用模板恒定在前，方言只贡献分页那一项：新增方言无需回到补全层登记
        expect(snippets.map((s) => s.label)).toEqual(['JOIN', 'INSERT', 'SELECT PAGE (LIMIT)']);
    });
});

describe('createDefaultContributors 贡献者链', () => {
    it('按序包含五个贡献者', () => {
        const chain = createDefaultContributors();
        expect(chain.map((c) => c.name)).toEqual(['schema', 'column', 'table', 'snippet', 'keyword']);
    });

    it('`.` 触发命中字段场景时提前收敛，仅返回 column 结果', async () => {
        const loadDbHints = vi.fn().mockResolvedValue({ users: ['id  [int][]'] });
        const ctx = createCtx({
            isDotTrigger: true,
            dotAlias: 'u',
            statement: 'SELECT * FROM users u WHERE u.',
        });
        (ctx.dbInst as unknown as Record<string, unknown>).loadDbHints = loadDbHints;

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

        // column 未命中表返回 undefined；schema 2 条 + table 空 + snippet 3 条（含方言分页模板）+ keyword 4 条
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
