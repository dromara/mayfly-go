import { describe, expect, it, vi } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

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

import { defaultSplitOptions, getSqlSplitOptions, getCurrentStatement, maskSqlComments, resolveSqlSplitOptions, splitSqlStatements, SqlSplitOptions } from '../sqlParser';
import { DbType } from '../../../../dialect/dbType';

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
        // 注释保留原文：语句偏移与原文一一对应，剔除注释会粘连 token 并丢失 hint 语义
        expect(result[1].text).toBe('-- comment; here\nSELECT 2');
        // textStart 跳过前导空白，指向文本首字符（此处为注释起始符）
        expect(sql.slice(result[1].textStart, result[1].end)).toBe(result[1].text);
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

describe('splitSqlStatements 方言选项（数据安全回归）', () => {
    it('反引号标识符内的分号不切割（mysql）', () => {
        const sql = 'SELECT * FROM `a;b`; SELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT * FROM `a;b`');
        expect(result[1].text).toBe('SELECT 2');
    });

    it('默认选项不识别反引号（保持历史行为）', () => {
        const sql = 'SELECT * FROM `a;b`;';
        const result = splitSqlStatements(sql);
        expect(result.length).toBeGreaterThan(1); // a;b 被误切（历史行为）
    });

    it('反引号内的 -- 与 /* 不视为注释（mysql）', () => {
        const sql = 'SELECT `a--b` FROM t; SELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT `a--b` FROM t');
    });

    it('反引号内反斜杠为普通字符（mysql，不吞掉闭合反引号）', () => {
        const sql = 'SELECT * FROM `a\\b`; SELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        // 标识符内反斜杠不做转义，不吞掉第二个反引号，标识符正常闭合后分号切割
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT * FROM `a\\b`');
        expect(result[1].text).toBe('SELECT 2');
    });

    it('反引号双写转义不终止标识符（mysql）', () => {
        const sql = 'SELECT * FROM `a``b;c` WHERE id = 1;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        expect(result).toHaveLength(1);
        expect(result[0].text).toBe('SELECT * FROM `a``b;c` WHERE id = 1');
    });

    it('mysql # 行注释内的分号不切割', () => {
        const sql = 'SELECT 1; # comment; here\nSELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT 1');
        expect(result[1].text).toBe('# comment; here\nSELECT 2');
    });

    it('默认选项 # 为普通字符（保持历史行为）', () => {
        const sql = 'SELECT 1; # comment; here\nSELECT 2;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(3); // # 注释被切分（历史行为）
    });

    it('postgres dollar-quoted 字符串内的分号不切割', () => {
        const sql = "SELECT $$a;b$$; SELECT 2;";
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('postgres'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT $$a;b$$');
    });

    it('带标签 dollar-quote 内的分号不切割且内容可跨行（postgres）', () => {
        const sql = 'SELECT $fn$ body ; more\nlines $fn$; SELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('postgres'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT $fn$ body ; more\nlines $fn$');
    });

    it('位置参数 $1 不误判为 dollar-quote 开标签（postgres）', () => {
        const sql = 'SELECT $1; SELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('postgres'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe('SELECT $1');
    });

    it('dollar-quote 内反斜杠为普通字符（postgres）', () => {
        const sql = "SELECT $$a\\'b;c$$; SELECT 2;";
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('postgres'));
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe("SELECT $$a\\'b;c$$");
    });

    it('非 pg 方言不启用 dollar-quote', () => {
        const sql = 'SELECT $$a;b$$; SELECT 2;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        // mysql 语义下 $$a;b$$ 内分号仍会切割（历史行为）
        expect(result.length).toBeGreaterThan(2);
    });

    it('backslashEscape=false 时反斜杠为普通字符（标准 SQL/PG）', () => {
        const sql = "SELECT 'a\\'; SELECT 2;";
        const result = splitSqlStatements(sql, ';', { backslashEscape: false });
        // \' 不再转义：字符串在第二个引号处闭合，随后分号切割且 SELECT 2 独立
        expect(result).toHaveLength(2);
        expect(result[0].text).toBe("SELECT 'a\\'");
        expect(result[1].text).toBe('SELECT 2');
    });

    it('backslashEscape=true（默认）保持转义语义', () => {
        const sql = "SELECT 'a\\'; SELECT 2;";
        const result = splitSqlStatements(sql);
        // \' 为转义引号，字符串未闭合，整段（含末尾分号）作为一条末尾语句
        expect(result).toHaveLength(1);
        expect(result[0].text).toBe("SELECT 'a\\'; SELECT 2;");
    });

    it('字符串以反斜杠结尾时不拼接字面量 undefined（缺陷回归）', () => {
        const result = splitSqlStatements("SELECT 'a\\");
        expect(result).toHaveLength(1);
        expect(result[0].text).not.toContain('undefined');
        expect(result[0].text).toBe("SELECT 'a\\");
    });

    it('未闭合字符串/注释的末尾语句兜底', () => {
        expect(splitSqlStatements("SELECT 'abc")).toHaveLength(1);
        expect(splitSqlStatements('SELECT /* unclosed')).toHaveLength(1);
    });

    it('sqlite 方言：反引号/方括号生效且反斜杠为普通字符', () => {
        expect(getSqlSplitOptions('sqlite')).toMatchObject({ backslashEscape: false, backtickQuote: true, bracketQuote: true });
    });

    it('未注册方言返回空选项（标准 SQL 语义）', () => {
        expect(getSqlSplitOptions('unknownDb')).toEqual({});
        expect(getSqlSplitOptions()).toEqual({});
    });

    it('postgres 方言：dollar-quote/E 串/嵌套注释开启且反斜杠不转义', () => {
        expect(getSqlSplitOptions('postgres')).toMatchObject({ backslashEscape: false, dollarQuote: true, escapeStringPrefix: true, nestedBlockComment: true });
    });

    it('嵌套块注释在第一个 */ 闭合', () => {
        const sql = '/* /* inner */ SELECT 1;';
        const result = splitSqlStatements(sql);
        expect(result).toHaveLength(1);
        expect(result[0].text).toBe('/* /* inner */ SELECT 1');
    });
});

describe('splitSqlStatements 注释保留（与服务端切割语义一致）', () => {
    it('mysql 可执行注释整体保留且单独成句', () => {
        const sql = '/*!40101 SET NAMES utf8 */;\nSELECT 1;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('mysql'));
        expect(result.map((r) => r.text)).toEqual(['/*!40101 SET NAMES utf8 */', 'SELECT 1']);
    });

    it('非可执行注释单独成分段时被丢弃（不产生空语句）', () => {
        expect(splitSqlStatements('/* just a note */;', ';', getSqlSplitOptions('mysql'))).toHaveLength(0);
    });

    it('优化器 hint 保留原文（oracle）', () => {
        const sql = 'SELECT /*+ FULL(t) */ * FROM t;';
        const result = splitSqlStatements(sql, ';', getSqlSplitOptions('oracle'));
        expect(result[0].text).toBe('SELECT /*+ FULL(t) */ * FROM t');
    });
});

describe('maskSqlComments', () => {
    it('掩码后长度与换行数不变，注释内容被替换为空白', () => {
        const sql = "SELECT 1 -- 说明 'x'\nFROM t /* 块\n注释 */ WHERE a='y'";
        const masked = maskSqlComments(sql);
        expect(masked.length).toBe(sql.length);
        expect(masked.split('\n')).toHaveLength(sql.split('\n').length);
        // 非空白字符保持原位置不变，故补全解析的下标仍可与原文对齐
        for (let i = 0; i < masked.length; i++) {
            if (masked[i] !== ' ') expect(masked[i]).toBe(sql[i]);
        }
        expect(masked.startsWith('SELECT 1 ')).toBe(true);
        expect(masked).toContain("WHERE a='y'");
        expect(masked).not.toContain('说明');
        expect(masked).not.toContain("'x'");
        expect(masked).not.toContain('块');
    });

    it('字符串内的 -- 与 # 不是注释，不参与掩码', () => {
        const sql = "SELECT 'a--b' FROM t;";
        expect(maskSqlComments(sql, getSqlSplitOptions('mysql'))).toBe(sql);
    });

    it('mysql 可执行注释是待执行 SQL，不被掩码', () => {
        const sql = '/*!40101 SET NAMES utf8 */';
        expect(maskSqlComments(sql, getSqlSplitOptions('mysql'))).toBe(sql);
    });

    it('无注释时原样返回', () => {
        expect(maskSqlComments('SELECT 1;')).toBe('SELECT 1;');
        expect(maskSqlComments('')).toBe('');
    });

    it('未闭合块注释掩码至文末', () => {
        const sql = 'SELECT 1 /* unclosed\nfoo';
        const masked = maskSqlComments(sql);
        expect(masked.length).toBe(sql.length);
        expect(masked.startsWith('SELECT 1 ')).toBe(true);
        expect(masked).not.toContain('unclosed');
        expect(masked).not.toContain('foo');
    });
});

describe('getCurrentStatement 方言选项', () => {
    const createMockModel = (offsetAtReturn: number) => ({
        getOffsetAt: vi.fn().mockReturnValue(offsetAtReturn),
    });

    it('光标在反引号标识符（含分号）所在语句内（mysql）', () => {
        const sql = 'SELECT * FROM `a;b`; SELECT 2;';
        const model = createMockModel(16) as any;
        const position = { lineNumber: 1, column: 17 } as any;
        expect(getCurrentStatement(sql, position, model, getSqlSplitOptions('mysql'))).toBe('SELECT * FROM `a;b`');
    });

    it('光标在语句间注释区域时归属后续语句（注释随语句一并返回）', () => {
        const sql = 'SELECT 1; # comment; here\nSELECT 2;';
        // 光标在 # 注释区域（offset 12），位置上位于第二条语句的原始范围 [9, 末尾] 内
        const model = createMockModel(12) as any;
        const position = { lineNumber: 1, column: 13 } as any;
        expect(getCurrentStatement(sql, position, model, getSqlSplitOptions('mysql'))).toBe('# comment; here\nSELECT 2');
    });
});

/**
 * 方言×复杂 SQL 回归矩阵。
 * 切割结果与语句偏移（start/end/textStart 均为原文下标）一并校验：注释与字面量保留原文，
 * 故任意用例都要求原文区间去空白后与语句文本完全一致，防止“文本对但位置错”导致执行/补全错位。
 */
type SplitCase = { name: string; sql: string; dbType: string; expected: string[] };

const splitTexts = (sql: string, dbType: string): string[] => {
    const result = splitSqlStatements(sql, ';', getSqlSplitOptions(dbType));
    result.forEach((r) => {
        expect(sql.slice(r.start, r.end).trim()).toBe(r.text);
        expect(sql.slice(r.textStart, r.end).trimEnd()).toBe(r.text);
    });
    return result.map((r) => r.text);
};

const runCases = (cases: SplitCase[]) => {
    it.each(cases)('$name', ({ sql, dbType, expected }) => {
        expect(splitTexts(sql, dbType)).toEqual(expected);
    });
};

describe('方言能力位矩阵（与服务端 dbm/sqlparser 选型对齐 + 前端补齐项）', () => {
    const cases: { dbType: string; expected: Partial<SqlSplitOptions> }[] = [
        { dbType: 'mysql', expected: { backslashEscape: true, backtickQuote: true, hashComment: true, lineCommentNeedsWhitespace: true, executableComment: true, blockMode: 'procedural' } },
        { dbType: 'mariadb', expected: { backslashEscape: true, backtickQuote: true, hashComment: true, lineCommentNeedsWhitespace: true, executableComment: true, blockMode: 'procedural' } },
        { dbType: 'postgres', expected: { backslashEscape: false, dollarQuote: true, escapeStringPrefix: true, nestedBlockComment: true, doubleQuoteAsIdentifier: true, blockMode: 'sql' } },
        { dbType: 'gauss', expected: { backslashEscape: false, dollarQuote: true, escapeStringPrefix: true, nestedBlockComment: true, doubleQuoteAsIdentifier: true, blockMode: 'sql' } },
        { dbType: 'kingbaseEs', expected: { backslashEscape: false, dollarQuote: true, escapeStringPrefix: true, nestedBlockComment: true, doubleQuoteAsIdentifier: true, blockMode: 'sql' } },
        { dbType: 'vastbase', expected: { backslashEscape: false, dollarQuote: true, escapeStringPrefix: true, nestedBlockComment: true, doubleQuoteAsIdentifier: true, blockMode: 'sql' } },
        { dbType: 'clickhouse', expected: { backslashEscape: true, backtickQuote: true, hashComment: true } },
        { dbType: 'sqlite', expected: { backslashEscape: false, backtickQuote: true, bracketQuote: true, doubleQuoteAsIdentifier: true } },
        { dbType: 'mssql', expected: { backslashEscape: false, bracketQuote: true, doubleQuoteAsIdentifier: true, blockMode: 'sql' } },
        { dbType: 'oracle', expected: { backslashEscape: false, altQuoteLiteral: true, doubleQuoteAsIdentifier: true, blockMode: 'procedural' } },
        { dbType: 'dm', expected: { backslashEscape: false, altQuoteLiteral: true, doubleQuoteAsIdentifier: true, blockMode: 'procedural' } },
    ];

    it.each(cases)('$dbType 解析为完整语义', ({ dbType, expected }) => {
        expect(resolveSqlSplitOptions(getSqlSplitOptions(dbType))).toEqual({ ...defaultSplitOptions, ...expected });
    });

    it('能力位集合完备（无未声明字段）', () => {
        expect(Object.keys(defaultSplitOptions).sort()).toEqual([
            'altQuoteLiteral',
            'backslashEscape',
            'backtickQuote',
            'blockMode',
            'bracketQuote',
            'dollarQuote',
            'doubleQuoteAsIdentifier',
            'escapeStringPrefix',
            'executableComment',
            'hashComment',
            'lineCommentNeedsWhitespace',
            'nestedBlockComment',
        ]);
    });

    it('已注册方言覆盖全部 DbType', () => {
        Object.values(DbType).forEach((dbType) => {
            expect(getSqlSplitOptions(dbType), dbType).not.toEqual({});
        });
    });
});

describe('切割回归：行注释与块注释', () => {
    runCases([
        { name: '块注释保留原文，闭合符即 token 分隔（mysql）', sql: 'SELECT count(*)/*x*/FROM t;', dbType: 'mysql', expected: ['SELECT count(*)/*x*/FROM t'] },
        { name: '行注释保留原文，换行仍作分隔（mysql）', sql: 'SELECT * FROM t WHERE a=1-- c\nAND b=2;', dbType: 'mysql', expected: ['SELECT * FROM t WHERE a=1-- c\nAND b=2'] },
        { name: '`1--2` 为减法而非注释（mysql）', sql: 'SELECT 1--2; SELECT 2;', dbType: 'mysql', expected: ['SELECT 1--2', 'SELECT 2'] },
        { name: '`--` 后跟制表符仍为注释，无换行则吞掉后续语句（mysql）', sql: 'SELECT 1--\t2; SELECT 2;', dbType: 'mysql', expected: ['SELECT 1--\t2; SELECT 2;'] },
        { name: '行注释内分号不切割（mssql）', sql: 'SELECT 1; -- end\nSELECT 2;', dbType: 'mssql', expected: ['SELECT 1', '-- end\nSELECT 2'] },
        { name: '块注释内含引号不影响闭合（postgres）', sql: "SELECT 1 /* 注释里的 ' 号 */; SELECT 2;", dbType: 'postgres', expected: ["SELECT 1 /* 注释里的 ' 号 */", 'SELECT 2'] },
        { name: '注释内分号不切割（postgres）', sql: 'SELECT 1; /* 说明; 含分号 */ SELECT 2;', dbType: 'postgres', expected: ['SELECT 1', '/* 说明; 含分号 */ SELECT 2'] },
        { name: '临时表名 # 不是注释符（mssql）', sql: 'SELECT * FROM #tmp; SELECT 2;', dbType: 'mssql', expected: ['SELECT * FROM #tmp', 'SELECT 2'] },
        { name: 'PG 的 # 为运算符不是注释符（postgres）', sql: 'SELECT a # b FROM t; SELECT 2;', dbType: 'postgres', expected: ['SELECT a # b FROM t', 'SELECT 2'] },
        { name: '默认选项 `--` 不要求空白（历史行为）', sql: 'SELECT 1--2; SELECT 2;', dbType: 'unknown', expected: ['SELECT 1--2; SELECT 2;'] },
        { name: '默认选项 # 为普通字符（历史行为）', sql: 'SELECT 1 # c; SELECT 2;', dbType: 'unknown', expected: ['SELECT 1 # c', 'SELECT 2'] },
    ]);
});

describe('切割回归：字符串与引用标识符', () => {
    runCases([
        { name: '双写引号转义（postgres）', sql: "SELECT 'a'';b' AS x; SELECT 2;", dbType: 'postgres', expected: ["SELECT 'a'';b' AS x", 'SELECT 2'] },
        { name: '双写引号转义（oracle）', sql: "SELECT 'a''b;c' FROM dual; SELECT 2 FROM dual;", dbType: 'oracle', expected: ["SELECT 'a''b;c' FROM dual", 'SELECT 2 FROM dual'] },
        { name: '反斜杠不转义，其后的引号即闭合（postgres）', sql: "SELECT 'a\\'; SELECT 2;", dbType: 'postgres', expected: ["SELECT 'a\\'", 'SELECT 2'] },
        { name: '反斜杠不转义（dm）', sql: "SELECT 'a\\'; SELECT 2 FROM dual;", dbType: 'dm', expected: ["SELECT 'a\\'", 'SELECT 2 FROM dual'] },
        { name: '反斜杠转义生效（clickhouse）', sql: "SELECT * FROM t WHERE a = 'x\\'y;b'; SELECT 2;", dbType: 'clickhouse', expected: ["SELECT * FROM t WHERE a = 'x\\'y;b'", 'SELECT 2'] },
        { name: '双写与反斜杠混合（mysql）', sql: "SELECT 'a''b\\'c;d'; SELECT 2;", dbType: 'mysql', expected: ["SELECT 'a''b\\'c;d'", 'SELECT 2'] },
        { name: 'E 串内反斜杠为转义符（postgres）', sql: "SELECT E'it\\'s'; SELECT 2;", dbType: 'postgres', expected: ["SELECT E'it\\'s'", 'SELECT 2'] },
        { name: '列名尾字母 e 不误判为 E 前缀（postgres）', sql: "SELECT table_name, 'a;b' FROM t; SELECT 2;", dbType: 'postgres', expected: ["SELECT table_name, 'a;b' FROM t", 'SELECT 2'] },
        { name: 'dollar-quote 嵌套内层标签不构成闭合（postgres）', sql: 'SELECT $$ outer $inner$ mid $inner$ end $$; SELECT 2;', dbType: 'postgres', expected: ['SELECT $$ outer $inner$ mid $inner$ end $$', 'SELECT 2'] },
        { name: '位置参数 $1 不当作开标签（postgres）', sql: 'SELECT * FROM t WHERE id=$1 AND name=$2; SELECT 2;', dbType: 'postgres', expected: ['SELECT * FROM t WHERE id=$1 AND name=$2', 'SELECT 2'] },
        { name: '类型转换与数组字面量（postgres）', sql: "SELECT a::int[], '{1;2;3}'::int[] FROM t; SELECT 2;", dbType: 'postgres', expected: ["SELECT a::int[], '{1;2;3}'::int[] FROM t", 'SELECT 2'] },
        { name: '嵌套块注释在闭合的嵌套层之外切割（postgres）', sql: 'SELECT 1 /* a /* b; */ c */; SELECT 2;', dbType: 'postgres', expected: ['SELECT 1 /* a /* b; */ c */', 'SELECT 2'] },
        { name: '非嵌套方言在首个 */ 闭合，残余作为代码（mysql）', sql: 'SELECT 1 /* a /* b; */ c */; SELECT 2;', dbType: 'mysql', expected: ['SELECT 1 /* a /* b; */ c */', 'SELECT 2'] },
        { name: '方括号标识符内分号不切（mssql）', sql: 'SELECT [a;b] FROM t; SELECT 2;', dbType: 'mssql', expected: ['SELECT [a;b] FROM t', 'SELECT 2'] },
        { name: ']] 为转义右括号（mssql）', sql: 'SELECT [a]]b;c] FROM t;', dbType: 'mssql', expected: ['SELECT [a]]b;c] FROM t'] },
        { name: '标识符内左括号为普通字符（mssql）', sql: 'SELECT [a[b]]] FROM t; SELECT 2;', dbType: 'mssql', expected: ['SELECT [a[b]]] FROM t', 'SELECT 2'] },
        { name: 'N 前缀字符串（mssql）', sql: "SELECT N'a;b'; SELECT 2;", dbType: 'mssql', expected: ["SELECT N'a;b'", 'SELECT 2'] },
        { name: 'CTE 名含分号（mssql）', sql: ';WITH [x;y] AS (SELECT 1 a) SELECT * FROM [x;y]; SELECT 2;', dbType: 'mssql', expected: ['WITH [x;y] AS (SELECT 1 a) SELECT * FROM [x;y]', 'SELECT 2'] },
        { name: '方括号标识符（sqlite）', sql: 'SELECT [a;b] FROM t;', dbType: 'sqlite', expected: ['SELECT [a;b] FROM t'] },
        { name: '反引号标识符（sqlite）', sql: 'SELECT `a;b` FROM t; SELECT 2;', dbType: 'sqlite', expected: ['SELECT `a;b` FROM t', 'SELECT 2'] },
        { name: '双引号为标识符，内 -- 不是注释（postgres）', sql: 'SELECT "a--b" FROM t;', dbType: 'postgres', expected: ['SELECT "a--b" FROM t'] },
        { name: 'oracle q-quote 括号型定界符', sql: "INSERT INTO t VALUES (q'[it's; ok]'); SELECT 2 FROM dual;", dbType: 'oracle', expected: ["INSERT INTO t VALUES (q'[it's; ok]')", 'SELECT 2 FROM dual'] },
        { name: 'oracle q-quote 自定义定界符', sql: "SELECT q'!it's; here!' FROM dual; SELECT 2 FROM dual;", dbType: 'oracle', expected: ["SELECT q'!it's; here!' FROM dual", 'SELECT 2 FROM dual'] },
        { name: 'oracle 空串与拼接', sql: "SELECT '' || name || ';' FROM t; SELECT 2 FROM dual;", dbType: 'oracle', expected: ["SELECT '' || name || ';' FROM t", 'SELECT 2 FROM dual'] },
        { name: 'dm q-quote 生效', sql: "SELECT q'[a;b]' FROM dual; SELECT 2 FROM dual;", dbType: 'dm', expected: ["SELECT q'[a;b]' FROM dual", 'SELECT 2 FROM dual'] },
        { name: '未闭合 q-quote 归为一条（降级不报错）', sql: "SELECT q'[abc FROM dual;", dbType: 'oracle', expected: ["SELECT q'[abc FROM dual;"] },
    ]);
});

describe('切割回归：复合语句块 BEGIN..END', () => {
    runCases([
        { name: '存储过程体不切割（mysql）', sql: 'CREATE PROCEDURE p(IN a INT) BEGIN SELECT 1; SELECT a; END;\nSELECT 2;', dbType: 'mysql', expected: ['CREATE PROCEDURE p(IN a INT) BEGIN SELECT 1; SELECT a; END', 'SELECT 2'] },
        { name: 'IF..END IF 与 CASE 嵌套（mysql）', sql: 'CREATE PROCEDURE p() BEGIN IF a THEN SELECT 1; ELSE SELECT CASE WHEN x THEN 1 END; END IF; LOOP SET @i=1; IF @i>2 THEN LEAVE l; END IF; END LOOP l; END; SELECT 9;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN IF a THEN SELECT 1; ELSE SELECT CASE WHEN x THEN 1 END; END IF; LOOP SET @i=1; IF @i>2 THEN LEAVE l; END IF; END LOOP l; END', 'SELECT 9'] },
        { name: '标签 LOOP（mysql）', sql: 'CREATE PROCEDURE p() BEGIN l: LOOP SELECT 1; END LOOP l; END;SELECT 2;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN l: LOOP SELECT 1; END LOOP l; END', 'SELECT 2'] },
        { name: '触发器体（mysql）', sql: 'CREATE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW BEGIN SET @a = 1; SET @b = 2; END;\nSELECT 2;', dbType: 'mysql', expected: ['CREATE TRIGGER tr BEFORE INSERT ON t FOR EACH ROW BEGIN SET @a = 1; SET @b = 2; END', 'SELECT 2'] },
        { name: '事务开启语句照常切割（mysql）', sql: 'BEGIN; SELECT 1; COMMIT;', dbType: 'mysql', expected: ['BEGIN', 'SELECT 1', 'COMMIT'] },
        { name: '空块（mysql）', sql: 'CREATE PROCEDURE p() BEGIN END;SELECT 1;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN END', 'SELECT 1'] },
        { name: '块内字符串含分号（mysql）', sql: "CREATE PROCEDURE p() BEGIN SELECT 'a;b'; END;SELECT 2;", dbType: 'mysql', expected: ["CREATE PROCEDURE p() BEGIN SELECT 'a;b'; END", 'SELECT 2'] },
        { name: 'END 紧跟分号（mysql）', sql: 'CREATE PROCEDURE p() BEGIN SELECT 1;END;SELECT 2;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN SELECT 1;END', 'SELECT 2'] },
        { name: '关键字小写同样生效（mysql）', sql: 'create procedure p() begin select 1; end;select 2;', dbType: 'mysql', expected: ['create procedure p() begin select 1; end', 'select 2'] },
        { name: '块内 DDL 的 IF NOT EXISTS 不入块（mysql）', sql: 'CREATE PROCEDURE p() BEGIN CREATE TABLE IF NOT EXISTS t (a INT); SELECT CASE WHEN x THEN 1 END; END;SELECT 1;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN CREATE TABLE IF NOT EXISTS t (a INT); SELECT CASE WHEN x THEN 1 END; END', 'SELECT 1'] },
        { name: 'BEGIN 无配对 END 时降级为普通语句切割（mysql）', sql: 'CREATE PROCEDURE p() BEGIN SELECT 1; SELECT 2;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN SELECT 1', 'SELECT 2'] },
        { name: 'BEGIN TRY/CATCH（mssql）', sql: 'BEGIN TRY SELECT 1; END TRY BEGIN CATCH SELECT 2; END CATCH;\nSELECT 3;', dbType: 'mssql', expected: ['BEGIN TRY SELECT 1; END TRY BEGIN CATCH SELECT 2; END CATCH', 'SELECT 3'] },
        { name: '嵌套 BEGIN（mssql）', sql: 'BEGIN BEGIN SELECT 1; END END;SELECT 2;', dbType: 'mssql', expected: ['BEGIN BEGIN SELECT 1; END END', 'SELECT 2'] },
        { name: 'IF/ELSE 无 END 闭合，按普通语句切割（mssql）', sql: 'IF 1=1 SELECT 1; ELSE SELECT 2;', dbType: 'mssql', expected: ['IF 1=1 SELECT 1', 'ELSE SELECT 2'] },
        { name: 'DECLARE 为独立语句（mssql）', sql: 'DECLARE @i INT; WHILE @i < 10 BEGIN SET @i = @i + 1; END;\nSELECT 4;', dbType: 'mssql', expected: ['DECLARE @i INT', 'WHILE @i < 10 BEGIN SET @i = @i + 1; END', 'SELECT 4'] },
        { name: 'Oracle 匿名块 DECLARE..BEGIN..END', sql: 'DECLARE x INT; BEGIN x := 1; DBMS_OUTPUT.PUT_LINE(x); END;\nSELECT 2 FROM dual;', dbType: 'oracle', expected: ['DECLARE x INT; BEGIN x := 1; DBMS_OUTPUT.PUT_LINE(x); END', 'SELECT 2 FROM dual'] },
        { name: 'Oracle FOR..LOOP（大小写混合）', sql: 'CREATE PROCEDURE p AS BEGIN FOR i IN 1..3 LOOP INSERT INTO t VALUES(i); END LOOP; END;\nSELECT 2 FROM dual;', dbType: 'oracle', expected: ['CREATE PROCEDURE p AS BEGIN FOR i IN 1..3 LOOP INSERT INTO t VALUES(i); END LOOP; END', 'SELECT 2 FROM dual'] },
        { name: 'PG BEGIN ATOMIC', sql: 'CREATE FUNCTION f() RETURNS int LANGUAGE sql BEGIN ATOMIC RETURN 1; END;\nSELECT 2;', dbType: 'postgres', expected: ['CREATE FUNCTION f() RETURNS int LANGUAGE sql BEGIN ATOMIC RETURN 1; END', 'SELECT 2'] },
        { name: 'PG 事务 BEGIN 不入口（postgres）', sql: 'BEGIN; SELECT 1; COMMIT;', dbType: 'postgres', expected: ['BEGIN', 'SELECT 1', 'COMMIT'] },
        { name: 'PG DO 块（体在 dollar-quote 内）', sql: 'DO $$ DECLARE x int; BEGIN x := 1; END $$;\nSELECT 2;', dbType: 'postgres', expected: ['DO $$ DECLARE x int; BEGIN x := 1; END $$', 'SELECT 2'] },
        { name: '顶层表达式 CASE 正常闭合（postgres）', sql: 'SELECT CASE WHEN a THEN 1 END; SELECT 2;', dbType: 'postgres', expected: ['SELECT CASE WHEN a THEN 1 END', 'SELECT 2'] },
        { name: '无块感知方言仍按分号切割（sqlite）', sql: 'BEGIN; INSERT INTO t VALUES(1); COMMIT;', dbType: 'sqlite', expected: ['BEGIN', 'INSERT INTO t VALUES(1)', 'COMMIT'] },
        { name: 'CASE 表达式不入口（clickhouse，blockMode=none）', sql: 'SELECT CASE WHEN a THEN 1 END; SELECT 2;', dbType: 'clickhouse', expected: ['SELECT CASE WHEN a THEN 1 END', 'SELECT 2'] },
    ]);
});

describe('切割回归：过程关键字同名标识符（不得误入块）', () => {
    runCases([
        { name: '表名 loop（mysql）', sql: 'SELECT * FROM loop; SELECT 2;', dbType: 'mysql', expected: ['SELECT * FROM loop', 'SELECT 2'] },
        { name: '表名 while（mysql）', sql: 'SELECT * FROM `while`; SELECT 2;', dbType: 'mysql', expected: ['SELECT * FROM `while`', 'SELECT 2'] },
        { name: '函数式 IF（mysql）', sql: 'SELECT IF(a,1,2) FROM t; SELECT 2;', dbType: 'mysql', expected: ['SELECT IF(a,1,2) FROM t', 'SELECT 2'] },
        { name: 'DDL IF NOT EXISTS（mysql）', sql: 'CREATE TABLE IF NOT EXISTS t (a INT); SELECT 2;', dbType: 'mysql', expected: ['CREATE TABLE IF NOT EXISTS t (a INT)', 'SELECT 2'] },
        { name: 'REPEAT() 同名函数（mysql）', sql: "SELECT REPEAT('a',2); SELECT 2;", dbType: 'mysql', expected: ["SELECT REPEAT('a',2)", 'SELECT 2'] },
        { name: '子串命中防护（begins/case_id/end_at）', sql: 'UPDATE begins SET case_id = 1 WHERE end_at IS NULL; SELECT 2;', dbType: 'mysql', expected: ['UPDATE begins SET case_id = 1 WHERE end_at IS NULL', 'SELECT 2'] },
        { name: '表名 ends（含 end 前缀）', sql: 'SELECT * FROM ends WHERE ends = 1; SELECT 2;', dbType: 'mysql', expected: ['SELECT * FROM ends WHERE ends = 1', 'SELECT 2'] },
        { name: '表名 loop（oracle）', sql: 'SELECT * FROM loop; SELECT 2 FROM dual;', dbType: 'oracle', expected: ['SELECT * FROM loop', 'SELECT 2 FROM dual'] },
        { name: '表名 while_（mssql）', sql: 'SELECT * FROM while_; SELECT 2;', dbType: 'mssql', expected: ['SELECT * FROM while_', 'SELECT 2'] },
        { name: '表达式 CASE 与普通语句共存（mysql）', sql: 'SELECT CASE WHEN a THEN 1 END, b FROM t; SELECT * FROM loop; SELECT 2;', dbType: 'mysql', expected: ['SELECT CASE WHEN a THEN 1 END, b FROM t', 'SELECT * FROM loop', 'SELECT 2'] },
    ]);
});

describe('切割回归：真实脚本与复杂字面量', () => {
    runCases([
        { name: '相邻字符串字面量跨行拼接（postgres）', sql: "SELECT 'a'\n'b;c';", dbType: 'postgres', expected: ["SELECT 'a'\n'b;c'"] },
        { name: '字符串含 NUL 字节不影响闭合（mysql）', sql: "SELECT 'a\u0000b;c'; SELECT 2;", dbType: 'mysql', expected: ["SELECT 'a\u0000b;c'", 'SELECT 2'] },
        { name: '单行超长字面量整条保留（mysql）', sql: `SELECT '${'a'.repeat(70000)};x';`, dbType: 'mysql', expected: [`SELECT '${'a'.repeat(70000)};x'`] },
        { name: '嵌套 CASE（mysql）', sql: 'SELECT CASE WHEN CASE WHEN x THEN 1 END THEN 2 END; SELECT 2;', dbType: 'mysql', expected: ['SELECT CASE WHEN CASE WHEN x THEN 1 END THEN 2 END', 'SELECT 2'] },
        { name: '匿名块 EXCEPTION 段（oracle）', sql: 'BEGIN x:=1; EXCEPTION WHEN OTHERS THEN NULL; END;\nSELECT 2 FROM dual;', dbType: 'oracle', expected: ['BEGIN x:=1; EXCEPTION WHEN OTHERS THEN NULL; END', 'SELECT 2 FROM dual'] },
        { name: '块内字符串含 END 文本不出块（mysql）', sql: "CREATE PROCEDURE p() BEGIN SELECT 'END;'; END;SELECT 2;", dbType: 'mysql', expected: ["CREATE PROCEDURE p() BEGIN SELECT 'END;'; END", 'SELECT 2'] },
        { name: '注释中的 END 不出块，无配对 END 则关闭块感知重切（mysql）', sql: 'CREATE PROCEDURE p() BEGIN SELECT 1; -- END\nSELECT 2;', dbType: 'mysql', expected: ['CREATE PROCEDURE p() BEGIN SELECT 1', '-- END\nSELECT 2'] },
        { name: 'CASE 缺配对 END 不吞并后续语句（mysql）', sql: 'SELECT CASE WHEN a THEN 1;SELECT 2;', dbType: 'mysql', expected: ['SELECT CASE WHEN a THEN 1', 'SELECT 2'] },
        { name: 'BEGIN 无分号且块未闭合时降级重切（postgres）', sql: 'BEGIN\nSELECT 1;SELECT 2;SELECT 3;', dbType: 'postgres', expected: ['BEGIN\nSELECT 1', 'SELECT 2', 'SELECT 3'] },
        { name: 'mssql IF..BEGIN..END ELSE..BEGIN..END', sql: 'IF 1=1 BEGIN SELECT 1; END ELSE BEGIN SELECT 2; END;', dbType: 'mssql', expected: ['IF 1=1 BEGIN SELECT 1; END ELSE BEGIN SELECT 2; END'] },
        // 客户端指令型分隔符不在能力表范围（与服务端 DialectSplitter 一致），此用例锁定当前行为不得恶化为吞并整段脚本
        { name: 'DELIMITER 指令不识别（mysql，已知限制）', sql: 'DELIMITER ;;\nCREATE PROCEDURE p() BEGIN SELECT 1;;\nEND;;\nDELIMITER ;', dbType: 'mysql', expected: ['DELIMITER', 'CREATE PROCEDURE p() BEGIN SELECT 1;;\nEND', 'DELIMITER'] },
    ]);

    it('导入文件开头的 UTF-8 BOM 不归入语句文本，但保留在偏移内', () => {
        const sql = '\uFEFFSELECT 1;\nSELECT 2;';
        const result = splitSqlStatements(sql);
        expect(result.map((r) => r.text)).toEqual(['SELECT 1', 'SELECT 2']);
        // start 仍指向原文下标（含 BOM），textStart 跳过 BOM 与空白
        expect(result[0].start).toBe(0);
        expect(result[0].textStart).toBe(1);
        expect(sql.slice(result[0].textStart, result[0].end)).toBe('SELECT 1');
    });
});

describe('切割回归：真实复杂 SQL', () => {
    runCases([
        {
            name: '带游标的完整存储过程（mysql）',
            sql: 'CREATE DEFINER=root@localhost PROCEDURE sp_x(IN p_id INT)\nBEGIN\n  DECLARE v_cnt INT DEFAULT 0;\n  DECLARE done INT DEFAULT FALSE;\n  DECLARE cur CURSOR FOR SELECT id FROM t WHERE id=p_id;\n  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;\n  IF v_cnt > 0 THEN\n    OPEN cur;\n    read_loop: LOOP\n      FETCH cur INTO v_cnt;\n      IF done THEN LEAVE read_loop; END IF;\n    END LOOP read_loop;\n    CLOSE cur;\n  END IF;\n  SELECT CASE WHEN v_cnt=0 THEN \'空\' ELSE CONCAT(\'共\', v_cnt, \'条\') END;\nEND;',
            dbType: 'mysql',
            expected: ['CREATE DEFINER=root@localhost PROCEDURE sp_x(IN p_id INT)\nBEGIN\n  DECLARE v_cnt INT DEFAULT 0;\n  DECLARE done INT DEFAULT FALSE;\n  DECLARE cur CURSOR FOR SELECT id FROM t WHERE id=p_id;\n  DECLARE CONTINUE HANDLER FOR NOT FOUND SET done = TRUE;\n  IF v_cnt > 0 THEN\n    OPEN cur;\n    read_loop: LOOP\n      FETCH cur INTO v_cnt;\n      IF done THEN LEAVE read_loop; END IF;\n    END LOOP read_loop;\n    CLOSE cur;\n  END IF;\n  SELECT CASE WHEN v_cnt=0 THEN \'空\' ELSE CONCAT(\'共\', v_cnt, \'条\') END;\nEND'],
        },
        { name: 'CTE + 窗口函数（mysql）', sql: 'WITH t AS (SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b) rn FROM x) SELECT * FROM t WHERE rn=1; SELECT 2;', dbType: 'mysql', expected: ['WITH t AS (SELECT ROW_NUMBER() OVER (PARTITION BY a ORDER BY b) rn FROM x) SELECT * FROM t WHERE rn=1', 'SELECT 2'] },
        { name: 'JSON 路径含分号（mysql）', sql: "SELECT JSON_EXTRACT(doc, '$.a[0].b;c') FROM t; SELECT 2;", dbType: 'mysql', expected: ["SELECT JSON_EXTRACT(doc, '$.a[0].b;c') FROM t", 'SELECT 2'] },
        { name: '中文与全角标点字面量（mysql）', sql: "INSERT INTO t(name, remark) VALUES('张三', '含分号;与中文——破折号'); SELECT 2;", dbType: 'mysql', expected: ["INSERT INTO t(name, remark) VALUES('张三', '含分号;与中文——破折号')", 'SELECT 2'] },
        { name: '中文标识符（mysql）', sql: 'SELECT `用户;名` FROM `订单表`; SELECT 2;', dbType: 'mysql', expected: ['SELECT `用户;名` FROM `订单表`', 'SELECT 2'] },
        { name: 'CRLF 换行下的行注释（mysql）', sql: 'SELECT 1; -- c\r\nSELECT 2;\r\n', dbType: 'mysql', expected: ['SELECT 1', '-- c\r\nSELECT 2'] },
        // 行注释仅以 \n 终止（与服务端 StatementScanner 一致）：全文只用 CR 换行时，注释后文本不产出语句
        { name: '仅 CR 换行的行注释延伸至文末（mysql）', sql: 'SELECT 1; -- c\rSELECT 2;', dbType: 'mysql', expected: ['SELECT 1'] },
        { name: '函数定义后跟普通语句（postgres）', sql: 'CREATE FUNCTION f() RETURNS void AS $$ BEGIN PERFORM 1; END $$ LANGUAGE plpgsql;\nDROP FUNCTION f();', dbType: 'postgres', expected: ['CREATE FUNCTION f() RETURNS void AS $$ BEGIN PERFORM 1; END $$ LANGUAGE plpgsql', 'DROP FUNCTION f()'] },
        { name: '注释后的复合块定义（mysql）', sql: '/* lead */ CREATE PROCEDURE p() BEGIN SELECT 1; END;\nSELECT 2;', dbType: 'mysql', expected: ['/* lead */ CREATE PROCEDURE p() BEGIN SELECT 1; END', 'SELECT 2'] },
        { name: 'INTERVAL 与字面量（clickhouse）', sql: "SELECT now() - INTERVAL 3 DAY AS d, 'a;b'; SELECT 2;", dbType: 'clickhouse', expected: ["SELECT now() - INTERVAL 3 DAY AS d, 'a;b'", 'SELECT 2'] },
    ]);
});

/**
 * SQL 切割跨语言契约（前端侧）：与服务端 internal/db/dbm/split_contract_test.go 读取**同一份** fixture。
 *
 * 切割语义在 Go 与 TS 各有一份状态机实现，而本文件的切割结果直接决定
 * 「SQL 编辑器把哪些文本作为独立语句发给服务端」；两侧各自维护期望值时，
 * 任何一侧单独改规则都不会让另一侧变红，故共享一份按真实数据库词法书写的用例。
 * 新增方言时必须同步向该 fixture 追加用例，两侧的“全方言覆盖”断言会拦住漏补行为。
 */
const findRepoRoot = (start: string): string => {
    let dir = start;
    for (let i = 0; i < 12; i++) {
        if (existsSync(join(dir, 'server')) && existsSync(join(dir, 'frontend'))) {
            return dir;
        }
        const parent = dirname(dir);
        if (parent === dir) {
            break;
        }
        dir = parent;
    }
    throw new Error('未定位到仓库根，无法读取 SQL 切割契约文件');
};

const SPLIT_CONTRACT_FILE = join(findRepoRoot(import.meta.dirname), 'server', 'internal/db/dbm/sqlparser/testdata/split_cases.json');

type SplitContractCase = { name: string; dbType: string; sql: string; expected: string[] };

describe('SQL 切割跨语言契约（与服务端共用 testdata/split_cases.json）', () => {
    const { cases } = JSON.parse(readFileSync(SPLIT_CONTRACT_FILE, 'utf-8')) as { cases: SplitContractCase[] };

    it('契约存在且非空', () => {
        expect(cases.length).toBeGreaterThanOrEqual(25);
    });

    it('服务端注册的全部方言（含别名）均有契约用例', () => {
        // 与 server/internal/db/dbm/dialect_registry_test.go 的 expectedDbTypes 保持一致
        const covered = new Set(cases.map((c) => c.dbType));
        ['mysql', 'mariadb', 'postgres', 'gauss', 'kingbaseEs', 'vastbase', 'sqlite', 'mssql', 'oracle', 'dm', 'clickhouse'].forEach((dbType) => {
            expect(covered.has(dbType), `方言 ${dbType} 缺少切割契约用例`).toBe(true);
        });
    });

    it.each(cases)('$name', ({ sql, dbType, expected }) => {
        // splitTexts 额外校验语句偏移与原文区间一致（仅文本一致不防“文本对位置错”）
        expect(splitTexts(sql, dbType), `[${dbType}] ${sql}`).toEqual(expected);
    });
});
