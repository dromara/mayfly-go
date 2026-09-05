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

import { extractStatementAt, parseCursorTokens, resolveCursorZone, resolveTableContext } from '../sqlContext';

describe('extractStatementAt', () => {
    it('提取光标所在语句', () => {
        const sql = 'SELECT 1; SELECT 2';
        // 第二条语句中间偏移量
        expect(extractStatementAt(sql, 11)).toBe('SELECT 2');
    });

    it('字符串中的分号不会被误认为语句分隔符', () => {
        const sql = "SELECT * FROM t WHERE remark = 'a;b'";
        expect(extractStatementAt(sql, 10)).toBe(sql);
    });

    it('光标位于语句间隙（分号后未输入）返回空串', () => {
        const sql = 'SELECT 1; ';
        expect(extractStatementAt(sql, 10)).toBe('');
    });

    it('光标在结束分号上返回当前语句', () => {
        const sql = 'SELECT 1;';
        expect(extractStatementAt(sql, 8)).toBe('SELECT 1');
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
});
