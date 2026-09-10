import { describe, expect, it, vi } from 'vitest';

// mock 方言注册表，避免 quoter 引入真实 dialect 模块连带加载 db/useI18n（localStorage 依赖）
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

// mock monaco-editor，避免加载完整编辑器依赖（sqlParser 依赖其类型导出）
vi.mock('monaco-editor', () => ({
    editor: {},
    Position: class Position {
        constructor(
            public lineNumber: number,
            public column: number
        ) {}
    },
}));

import { getSqlSplitOptions, splitSqlStatements } from '../../../component/sqleditor/utils/sqlParser';
import { extractStatementAt, parseCursorTokens, resolveCursorClause, resolveCursorZone, resolveNearestTableContext, resolveScopeTableContexts, resolveTableContext } from '../sqlContext';

describe('extractStatementAt', () => {
    it('提取光标所在语句及光标相对偏移', () => {
        const sql = 'SELECT 1; SELECT 2';
        // 语句文本不含前导空白（起点 10），故 offset 11 对应文本内偏移 1
        const r = extractStatementAt(sql, 11);
        expect(r?.text).toBe('SELECT 2');
        expect(r?.cursorOffset).toBe(1);
    });

    it('注释保留在语句文本中但被掩码为空白，偏移仍与原文对齐', () => {
        const sql = 'SELECT 1; -- 说明\nSELECT 2 FROM t';
        const offset = sql.length; // 光标在文末
        const r = extractStatementAt(sql, offset);
        // 文本长度与原文区间（-- 说明\nSELECT 2 FROM t）一致，仅注释内容替为空白
        expect(r?.text.length).toBe('-- 说明\nSELECT 2 FROM t'.length);
        expect(r?.cursorOffset).toBe(r?.text.length);
        expect(r?.text).not.toContain('说明');
        expect(r?.text).toContain('SELECT 2 FROM t');
    });

    it('字符串中的分号不会被误认为语句分隔符', () => {
        const sql = "SELECT * FROM t WHERE remark = 'a;b'";
        expect(extractStatementAt(sql, 10)?.text).toBe(sql);
    });

    it('光标位于语句间隙（分号后未输入）返回 undefined', () => {
        const sql = 'SELECT 1; ';
        expect(extractStatementAt(sql, 10)).toBeUndefined();
    });

    it('光标在结束分号上返回当前语句', () => {
        const r = extractStatementAt('SELECT 1;', 8);
        expect(r?.text).toBe('SELECT 1');
        expect(r?.cursorOffset).toBe(8);
    });

    it('方言选项透传：反引号标识符内光标不跨语句（mysql）', () => {
        const sql = 'SELECT * FROM `a;b` WHERE id = 10';
        const r = extractStatementAt(sql, 5, getSqlSplitOptions('mysql'));
        expect(r?.text).toBe(sql);
        // 不传方言选项时历史行为：a;b 被误切
        expect(extractStatementAt(sql, 5)?.text).toBe('SELECT * FROM `a');
    });

    it('方言选项透传：dollar-quote 内光标不跨语句（postgres）', () => {
        const sql = 'SELECT $$a;b$$; SELECT 2;';
        const r = extractStatementAt(sql, 10, getSqlSplitOptions('postgres'));
        expect(r?.text).toBe('SELECT $$a;b$$');
    });
});

describe('parseCursorTokens', () => {
    it('普通空格触发', () => {
        const r = parseCursorTokens('select * from ');
        expect(r.isDotTrigger).toBe(false);
        expect(r.lastToken).toBe('from');
    });

    it('表别名点触发', () => {
        const r = parseCursorTokens('select * from users u where u.');
        expect(r.isDotTrigger).toBe(true);
        expect(r.dotAlias).toBe('u');
    });

    it('库名点触发', () => {
        const r = parseCursorTokens('select * from mydb.');
        expect(r.isDotTrigger).toBe(true);
        expect(r.dotAlias).toBe('mydb');
    });

    it('逗号粘连形态 a.creator,a.', () => {
        const r = parseCursorTokens('select a.creator,a.');
        expect(r.isDotTrigger).toBe(true);
        expect(r.dotAlias).toBe('a');
    });

    it('带反引号的表名点触发 `users`.', () => {
        const r = parseCursorTokens('select * from `users`.');
        expect(r.isDotTrigger).toBe(true);
        expect(r.dotAlias).toBe('users');
    });

    it('带双引号的库名点触发 "mydb".', () => {
        const r = parseCursorTokens('select * from "mydb".');
        expect(r.isDotTrigger).toBe(true);
        expect(r.dotAlias).toBe('mydb');
    });

    it('别名统一小写化', () => {
        const r = parseCursorTokens('select * from Users U where U.');
        expect(r.dotAlias).toBe('u');
    });
});

describe('resolveTableContext', () => {
    it('按别名解析表名', () => {
        const ctx = resolveTableContext('SELECT * FROM `users` u WHERE u.id = 1', 'u', 'test');
        expect(ctx?.tableName).toBe('users');
        expect(ctx?.db).toBe('test');
    });

    it('未指定别名时返回第一个表', () => {
        const ctx = resolveTableContext('SELECT * FROM a JOIN b ON a.id = b.id', '', 'test');
        expect(ctx?.tableName).toBe('a');
    });

    it('库名.表名形式且默认库名带前缀', () => {
        const ctx = resolveTableContext('SELECT * FROM db1.orders o', 'o', 'id_1/root');
        expect(ctx?.tableName).toBe('orders');
        expect(ctx?.db).toBe('id_1/db1');
    });

    it('别名不存在时返回 undefined', () => {
        expect(resolveTableContext('SELECT * FROM users u', 'x', 'test')).toBeUndefined();
    });

    it('别名匹配忽略大小写（FROM Users U + u.）', () => {
        const ctx = resolveTableContext('SELECT * FROM Users U WHERE U.id = 1', 'u', 'test');
        expect(ctx?.tableName).toBe('Users');
    });

    it('直接用表名点触发（无别名）也可匹配', () => {
        const ctx = resolveTableContext('SELECT * FROM users WHERE users.id = 1', 'users', 'test');
        expect(ctx?.tableName).toBe('users');
    });

    it('INSERT INTO 语句可解析表名', () => {
        const ctx = resolveTableContext("INSERT INTO users (name, age) VALUES ('a', 1)", '', 'test');
        expect(ctx?.tableName).toBe('users');
    });

    it('INSERT INTO 表名后跟括号（无空格）', () => {
        const ctx = resolveTableContext('INSERT INTO users(name, age) VALUES (1, 2)', '', 'test');
        expect(ctx?.tableName).toBe('users');
    });

    it('DELETE FROM 语句可解析表名', () => {
        const ctx = resolveTableContext('DELETE FROM users WHERE id = 1', '', 'test');
        expect(ctx?.tableName).toBe('users');
    });

    it('INSERT INTO 库名.表名形式且默认库名带前缀', () => {
        const ctx = resolveTableContext('INSERT INTO db1.orders (id) VALUES (1)', '', 'id_1/root');
        expect(ctx?.tableName).toBe('orders');
        expect(ctx?.db).toBe('id_1/db1');
    });

    it('表名后的占位关键字不视为别名（UPDATE ... SET）', () => {
        const ctx = resolveTableContext('UPDATE users SET name = 1 WHERE id = 1', 'set', 'test');
        expect(ctx).toBeUndefined();
        const first = resolveTableContext('UPDATE users SET name = 1 WHERE id = 1', '', 'test');
        expect(first?.tableName).toBe('users');
    });
});

describe('resolveNearestTableContext（JOIN 多表就近取表）', () => {
    const multiJoinSql = 'SELECT * FROM a x JOIN b y ON x.id = y.id JOIN c z ON y.id = z.id WHERE ';

    it('光标在语句末尾取最后一个表子句', () => {
        const ctx = resolveNearestTableContext(multiJoinSql, multiJoinSql.length, 'test');
        expect(ctx?.tableName).toBe('c');
    });

    it('光标在第一个 JOIN 之后取第二个表', () => {
        // offset 定位到 "JOIN b" 之后
        const ctx = resolveNearestTableContext(multiJoinSql, multiJoinSql.indexOf('JOIN c'), 'test');
        expect(ctx?.tableName).toBe('b');
    });

    it('光标在首个表子句之前时退回第一个表', () => {
        const ctx = resolveNearestTableContext(multiJoinSql, 7, 'test');
        expect(ctx?.tableName).toBe('a');
    });

    it('UPDATE 语句光标在 SET 子句时取 UPDATE 表', () => {
        const sql = 'UPDATE users SET name = ';
        const ctx = resolveNearestTableContext(sql, sql.length, 'test');
        expect(ctx?.tableName).toBe('users');
    });

    it('INSERT INTO 语句取 INTO 后的表', () => {
        const sql = 'INSERT INTO users (name, age) VALUES (1, ';
        const ctx = resolveNearestTableContext(sql, sql.length, 'test');
        expect(ctx?.tableName).toBe('users');
    });

    it('无表子句时返回 undefined', () => {
        expect(resolveNearestTableContext('SELECT 1', 9, 'test')).toBeUndefined();
    });
});

describe('resolveScopeTableContexts（作用域全表）', () => {
    it('多表 JOIN 按出现顺序返回全部表（含别名）', () => {
        const ctxs = resolveScopeTableContexts('SELECT * FROM a x JOIN b y ON x.id = y.id JOIN c z ON y.id = z.id', 'test');
        expect(ctxs.map((c) => `${c.tableName}:${c.tableAlias}`)).toEqual(['a:x', 'b:y', 'c:z']);
    });

    it('子查询内的表也纳入作用域', () => {
        const ctxs = resolveScopeTableContexts('SELECT * FROM t1 WHERE id IN (SELECT id FROM t2)', 'test');
        expect(ctxs.map((c) => c.tableName)).toEqual(['t1', 't2']);
    });

    it('无表时返回空数组', () => {
        expect(resolveScopeTableContexts('SELECT 1', 'test')).toEqual([]);
    });
});

describe('resolveCursorClause（子句感知）', () => {
    it('空语句为 free', () => {
        expect(resolveCursorClause('', 0, false)).toBe('free');
    });

    it('无关键字语句为 free', () => {
        expect(resolveCursorClause('SELEC', 5, false)).toBe('free');
    });

    it('FROM/JOIN/INTO/UPDATE 后为 table（表名期望位置）', () => {
        expect(resolveCursorClause('SELECT * FROM ', 14, false)).toBe('table');
        expect(resolveCursorClause('SELECT * FROM t JOIN ', 21, false)).toBe('table');
        expect(resolveCursorClause('INSERT INTO ', 12, false)).toBe('table');
        expect(resolveCursorClause('UPDATE ', 7, false)).toBe('table');
        expect(resolveCursorClause('DELETE FROM ', 12, false)).toBe('table');
    });

    it('SELECT/WHERE/ON/SET/ORDER BY/HAVING 后为 column（字段/表达式位置）', () => {
        expect(resolveCursorClause('SELECT ', 7, false)).toBe('column');
        expect(resolveCursorClause('SELECT * FROM t WHERE ', 22, false)).toBe('column');
        expect(resolveCursorClause('SELECT * FROM a x JOIN b y ON x.id = y.id AND ', 46, false)).toBe('column');
        expect(resolveCursorClause('UPDATE users SET name = ', 24, false)).toBe('column');
        expect(resolveCursorClause('SELECT * FROM t ORDER BY ', 25, false)).toBe('column');
        expect(resolveCursorClause('SELECT * FROM t GROUP BY ', 25, false)).toBe('column');
        expect(resolveCursorClause('SELECT * FROM t HAVING ', 23, false)).toBe('column');
    });

    it('INSERT INTO 表名后的未闭合括号内为 column（列清单）', () => {
        expect(resolveCursorClause('INSERT INTO users (name, ', 25, false)).toBe('column');
        // 括号已闭合且 VALUES 后为表达式位置，兜底为 column（不再回到 table）
        expect(resolveCursorClause('INSERT INTO users (a) VALUES ', 29, false)).toBe('column');
    });

    it('左括号紧邻光标时为 column（IN 子查询/列清单参数位）', () => {
        expect(resolveCursorClause('SELECT * FROM t WHERE id IN (', 29, true)).toBe('column');
        expect(resolveCursorClause('INSERT INTO t (', 15, true)).toBe('column');
    });

    it('子查询内的 SELECT 为 column', () => {
        expect(resolveCursorClause('SELECT * FROM t WHERE id IN (SELECT ', 36, false)).toBe('column');
    });
});

describe('resolveCursorZone', () => {
    it('普通代码区域', () => {
        expect(resolveCursorZone('SELECT * FROM t WHERE id = 1', 10)).toBe('code');
    });

    it('单引号字符串内为 string', () => {
        const sql = "SELECT * FROM t WHERE remark = 'a;b'";
        // 字符串内部偏移
        expect(resolveCursorZone(sql, 33)).toBe('string');
        // 结束引号之后为 code
        expect(resolveCursorZone(sql, 36)).toBe('code');
    });

    it('支持反斜杠转义的引号', () => {
        const sql = "SELECT 'it\\'s ok' FROM t";
        // 转义引号后的内容仍在字符串内
        expect(resolveCursorZone(sql, 15)).toBe('string');
        expect(resolveCursorZone(sql, 22)).toBe('code');
    });

    it('行注释内为 comment，换行后恢复 code', () => {
        const sql = '-- comment\nSELECT 1';
        expect(resolveCursorZone(sql, 5)).toBe('comment');
        expect(resolveCursorZone(sql, 12)).toBe('code');
    });

    it('块注释内为 comment，结束后恢复 code', () => {
        const sql = '/* comment */ SELECT 1';
        expect(resolveCursorZone(sql, 5)).toBe('comment');
        expect(resolveCursorZone(sql, 14)).toBe('code');
    });

    it('双引号/反引号不视为字符串（标识符引用符）', () => {
        const sql = 'SELECT `id` FROM t';
        expect(resolveCursorZone(sql, 9)).toBe('code');
    });

    it('backtickQuote 时反引号内 -- 不视为注释（mysql）', () => {
        const sql = 'SELECT `a--b` FROM t';
        // 默认：-- 进入注释区
        expect(resolveCursorZone(sql, 11)).toBe('comment');
        // mysql：-- 在反引号标识符内，仍为 code
        expect(resolveCursorZone(sql, 11, getSqlSplitOptions('mysql'))).toBe('code');
    });

    it('backslashEscape=false 时反斜杠不转义（标准 SQL/PG）', () => {
        const sql = "SELECT 'a\\'; SELECT 2";
        // 默认：\' 转义，字符串一直未闭合到末尾
        expect(resolveCursorZone(sql, sql.length - 1)).toBe('string');
        // 标准 SQL：字符串在第二个引号闭合，后续为 code
        expect(resolveCursorZone(sql, sql.length - 1, { backslashEscape: false })).toBe('code');
    });

    it('hashComment 时 # 注释区内为 comment（mysql）', () => {
        const sql = 'SELECT 1; # note\nSELECT 2';
        expect(resolveCursorZone(sql, 12, getSqlSplitOptions('mysql'))).toBe('comment');
        expect(resolveCursorZone(sql, 12)).toBe('code');
    });

    it('dollar-quote 内容保持 code（函数体内仍提供提示，postgres）', () => {
        const sql = 'CREATE FUNCTION f() AS $$ SELECT * FROM t $$;';
        expect(resolveCursorZone(sql, 32, getSqlSplitOptions('postgres'))).toBe('code');
    });
});

describe('resolveCursorZone 方言区域归属（与切割共用扫描器）', () => {
    const cases: { name: string; sql: string; offset: number; dbType: string; expected: string }[] = [
        { name: '方括号标识符内仍给代码提示（mssql）', sql: 'SELECT [a;b] FROM t', offset: 8, dbType: 'mssql', expected: 'code' },
        { name: '双引号标识符内的 -- 不是注释（postgres）', sql: 'SELECT "a--b" FROM t', offset: 8, dbType: 'postgres', expected: 'code' },
        { name: '双引号标识符内的 -- 不是注释（sqlite）', sql: 'SELECT "a;b" FROM t', offset: 8, dbType: 'sqlite', expected: 'code' },
        { name: '双引号为字符串字面量，不给提示（mysql）', sql: 'SELECT "a;b" FROM t', offset: 8, dbType: 'mysql', expected: 'string' },
        { name: 'E 串内部为字符串（postgres）', sql: "SELECT E'it\\'s' FROM t", offset: 10, dbType: 'postgres', expected: 'string' },
        { name: 'q-quote 内部为字符串（oracle）', sql: "SELECT q'[a;b]' FROM dual", offset: 11, dbType: 'oracle', expected: 'string' },
        { name: '双写引号中间仍在字符串内（oracle）', sql: "SELECT 'a''b' FROM dual", offset: 9, dbType: 'oracle', expected: 'string' },
        { name: '# 为运算符不是注释（postgres）', sql: 'SELECT a # b FROM t', offset: 9, dbType: 'postgres', expected: 'code' },
        { name: '# 行注释内不提示（clickhouse）', sql: 'SELECT 1 # c\nSELECT 2', offset: 11, dbType: 'clickhouse', expected: 'comment' },
        { name: '嵌套块注释内层为注释（postgres）', sql: 'SELECT /* a /* b; */ c */ 1', offset: 19, dbType: 'postgres', expected: 'comment' },
        { name: '-- 后无空白时为运算（mysql）', sql: 'SELECT 1--2 FROM t', offset: 10, dbType: 'mysql', expected: 'code' },
        { name: '-- 后无空白仍为注释（postgres）', sql: 'SELECT 1--2 FROM t', offset: 10, dbType: 'postgres', expected: 'comment' },
        { name: '行注释内为 comment（mssql）', sql: 'SELECT 1; -- end\nSELECT 2', offset: 12, dbType: 'mssql', expected: 'comment' },
        { name: '复合块体内为 code（mysql）', sql: 'CREATE PROCEDURE p() BEGIN SELECT 1; END', offset: 32, dbType: 'mysql', expected: 'code' },
    ];

    it.each(cases)('$name', ({ sql, offset, dbType, expected }) => {
        expect(resolveCursorZone(sql, offset, getSqlSplitOptions(dbType))).toBe(expected);
    });

    it('区段边界：闭合引号之后恢复 code', () => {
        const sql = "SELECT E'it\\'s' FROM t";
        expect(resolveCursorZone(sql, sql.indexOf('FROM'), getSqlSplitOptions('postgres'))).toBe('code');
    });

    it('越界偏移自动收敛（不抛异常）', () => {
        const sql = "SELECT 'abc";
        expect(resolveCursorZone(sql, 999, getSqlSplitOptions('postgres'))).toBe('string');
        expect(resolveCursorZone(sql, -5, getSqlSplitOptions('postgres'))).toBe('code');
    });

    it('与切割保持一致：字符串内的分号不产生语句边界', () => {
        const sql = "SELECT 'a;b'; SELECT 2";
        const opts = getSqlSplitOptions('postgres');
        // 偏移 9 为字符串内部的分号：既判定为 string，也不产生语句边界
        expect(resolveCursorZone(sql, 9, opts)).toBe('string');
        expect(splitSqlStatements(sql, ';', opts).map((s) => s.text)).toEqual(["SELECT 'a;b'", 'SELECT 2']);
    });
});
