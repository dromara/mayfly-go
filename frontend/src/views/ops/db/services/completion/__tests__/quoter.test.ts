import { describe, expect, it, vi } from 'vitest';

// mock 方言注册表，避免引入真实 dialect 模块连带加载 monaco 全量依赖
vi.mock('../../../dialect', () => ({
    DbType: {
        mysql: 'mysql',
        mariadb: 'mariadb',
        postgresql: 'postgres',
        sqlite: 'sqlite',
        mssql: 'mssql',
        clickhouse: 'clickhouse',
    },
}));

import { createCompletionQuoter, findWrappingQuote, getDialectQuotePairs, quoteSuggestions, stripIdentifierQuotes } from '../quoter';
import type { DbDialect } from '../../../dialect';
import type { languages } from 'monaco-editor';

describe('getDialectQuotePairs', () => {
    it('mysql 返回反引号', () => {
        expect(getDialectQuotePairs('mysql')).toEqual([{ open: '`', close: '`' }]);
    });

    it('未注册方言回退标准双引号', () => {
        expect(getDialectQuotePairs('postgres')).toEqual([{ open: '"', close: '"' }]);
    });

    it('mssql 返回方括号与双引号', () => {
        expect(getDialectQuotePairs('mssql')).toEqual([
            { open: '[', close: ']' },
            { open: '"', close: '"' },
        ]);
    });
});

describe('findWrappingQuote', () => {
    const backtick = getDialectQuotePairs('mysql');

    it('单词被反引号包裹时命中', () => {
        const line = 'SELECT `id` FROM t';
        // `id` 的单词范围：startColumn=9, endColumn=11（1-based，不含）
        expect(findWrappingQuote(line, 9, 11, backtick)).toEqual({ open: '`', close: '`' });
    });

    it('裸单词不命中', () => {
        const line = 'SELECT id FROM t';
        expect(findWrappingQuote(line, 8, 10, backtick)).toBeUndefined();
    });

    it('仅一侧有引用符不命中', () => {
        const line = "SELECT 'id FROM t";
        expect(findWrappingQuote(line, 9, 11, backtick)).toBeUndefined();
    });
});

describe('stripIdentifierQuotes', () => {
    const backtick = getDialectQuotePairs('mysql');

    it('去除首尾反引号', () => {
        expect(stripIdentifierQuotes('`id`', backtick)).toBe('id');
    });

    it('裸名原样返回', () => {
        expect(stripIdentifierQuotes('id', backtick)).toBe('id');
    });

    it('仅首尾同时匹配才去除', () => {
        expect(stripIdentifierQuotes('a`b', backtick)).toBe('a`b');
    });
});

describe('createCompletionQuoter', () => {
    const dialect = { quoteIdentifier: (name: string) => '`' + name + '`' } as unknown as DbDialect;
    const pairs = getDialectQuotePairs('mysql');

    it('未包裹时按方言包裹', () => {
        const quote = createCompletionQuoter(dialect, pairs, false);
        expect(quote('id')).toBe('`id`');
    });

    it('光标处已包裹时插入裸名，避免二次包裹', () => {
        const quote = createCompletionQuoter(dialect, pairs, true);
        expect(quote('id')).toBe('id');
    });

    it('对自带引用符的名称幂等去包裹后再包裹', () => {
        const quote = createCompletionQuoter(dialect, pairs, false);
        expect(quote('`id`')).toBe('`id`');
    });
});

describe('quoteSuggestions', () => {
    const dialect = { quoteIdentifier: (name: string) => '`' + name + '`' } as unknown as DbDialect;
    const pairs = getDialectQuotePairs('mysql');
    const suggestions = [
        { label: 'id', insertText: 'id' },
        { label: 'users', insertText: 'users', sortText: '3001' },
    ] as unknown as languages.CompletionItem[];

    it('未包裹时对整批建议应用方言包裹', () => {
        const quote = createCompletionQuoter(dialect, pairs, false);
        const res = quoteSuggestions(quote, suggestions);
        expect(res.map((s) => s.insertText)).toEqual(['`id`', '`users`']);
        expect(res[1].sortText).toBe('3001');
    });

    it('已包裹时整批插入裸名，避免二次包裹', () => {
        const quote = createCompletionQuoter(dialect, pairs, true);
        const res = quoteSuggestions(quote, suggestions);
        expect(res.map((s) => s.insertText)).toEqual(['id', 'users']);
    });

    it('无 insertText 的建议项原样保留', () => {
        const res = quoteSuggestions(createCompletionQuoter(dialect, pairs, false), [{ label: 'x' } as languages.CompletionItem]);
        expect(res[0].insertText).toBeUndefined();
    });
});
