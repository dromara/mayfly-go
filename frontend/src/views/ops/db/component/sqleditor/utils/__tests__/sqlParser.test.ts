import { describe, expect, it, vi } from 'vitest';

// mock monaco-editor，避免加载完整编辑器依赖
vi.mock('monaco-editor', () => ({
    editor: {},
    Position: class Position {
        constructor(
            public lineNumber: number,
            public column: number
        ) {}
    },
}));

import { getCurrentStatement, splitSqlStatements } from '../sqlParser';

describe('splitSqlStatements', () => {
    it('单条语句（带分号）', () => {
        const result = splitSqlStatements('SELECT * FROM users;');
        expect(result).toHaveLength(1);
        expect(result[0].text).toBe('SELECT * FROM users');
        expect(result[0].start).toBe(0);
        expect(result[0].end).toBe(19); // 分号所在索引
    });

    it('多条语句', () => {
        const sql = 'SELECT 1; SELECT 2; SELECT 3;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(3);
        expect(result[0].text).toBe('SELECT 1');
        expect(result[1].text).toBe('SELECT 2');
        expect(result[2].text).toBe('SELECT 3');
    });

    it('最后一条语句无分号', () => {
        const sql = 'SELECT 1; SELECT 2';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[1].text).toBe('SELECT 2');
        expect(result[1].end).toBe(sql.length);
    });

    it('忽略空语句（连续分号）', () => {
        const sql = 'SELECT 1;; ; SELECT 2;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT 1');
        expect(result[1].text).toBe('SELECT 2');
    });

    it('字符串内的分号不分割', () => {
        const sql = "SELECT * FROM t WHERE name = 'a;b'; SELECT 2;";
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe("SELECT * FROM t WHERE name = 'a;b'");
    });

    it('双引号字符串内的分号不分割', () => {
        const sql = 'SELECT * FROM t WHERE name = "a;b"; SELECT 2;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT * FROM t WHERE name = "a;b"');
    });

    it('转义引号不终止字符串', () => {
        const sql = "SELECT * FROM t WHERE name = 'it\\'s'; SELECT 2;";
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe("SELECT * FROM t WHERE name = 'it\\'s'");
    });

    it('单行注释内的分号不分割', () => {
        const sql = 'SELECT 1; -- comment; here\nSELECT 2;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT 1');
        expect(result[1].text).toBe('SELECT 2');
    });

    it('多行注释内的分号不分割', () => {
        const sql = 'SELECT 1; /* comment; here */ SELECT 2;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT 1');
    });

    it('自定义分隔符', () => {
        const sql = 'SELECT 1/ SELECT 2/';
        const result = splitSqlStatements(sql, '/');
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT 1');
        expect(result[1].text).toBe('SELECT 2');
    });

    it('空字符串返回空数组', () => {
        expect(splitSqlStatements('')).toHaveLength(0);
        expect(splitSqlStatements('   ')).toHaveLength(0);
    });

    it('位置信息正确', () => {
        const sql = 'SELECT 1; SELECT 2';
        const result = splitSqlStatements(sql);
        expect(result[0].start).toBe(0);
        expect(result[0].end).toBe(8); // 分号位置
        expect(result[1].start).toBe(9); // 分号后一位（含空格）
        expect(result[1].end).toBe(18); // 字符串末尾
    });
});

describe('getCurrentStatement', () => {
    const createMockModel = (offsetAtReturn: number) => ({
        getOffsetAt: vi.fn().mockReturnValue(offsetAtReturn),
    });

    it('光标在第一条语句内', () => {
        const sql = 'SELECT 1; SELECT 2;';
        const model = createMockModel(3) as any;
        const position = { lineNumber: 1, column: 4 } as any;
        expect(getCurrentStatement(sql, position, model)).toBe('SELECT 1');
    });

    it('光标在第二条语句内', () => {
        const sql = 'SELECT 1; SELECT 2;';
        const model = createMockModel(12) as any;
        const position = { lineNumber: 1, column: 13 } as any;
        expect(getCurrentStatement(sql, position, model)).toBe('SELECT 2');
    });

    it('光标在分号后一个位置仍返回当前语句', () => {
        const sql = 'SELECT 1; SELECT 2;';
        // offset = 9 即分号后一位
        const model = createMockModel(9) as any;
        const position = { lineNumber: 1, column: 10 } as any;
        expect(getCurrentStatement(sql, position, model)).toBe('SELECT 1');
    });

    it('光标在语句之后的空白区域返回最后一条语句', () => {
        const sql = 'SELECT 1;   ';
        // offset=11 在尾部空白处，超出所有语句范围，fallback 返回最后一条
        const model = createMockModel(11) as any;
        const position = { lineNumber: 1, column: 12 } as any;
        expect(getCurrentStatement(sql, position, model)).toBe('SELECT 1');
    });

    it('无语句时返回 null', () => {
        const sql = '   ';
        const model = createMockModel(0) as any;
        const position = { lineNumber: 1, column: 1 } as any;
        expect(getCurrentStatement(sql, position, model)).toBeNull();
    });
});
