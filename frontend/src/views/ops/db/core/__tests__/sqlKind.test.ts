import { describe, expect, it } from 'vitest';

import { isDdlSql, isQueryStatement, stripLeadingComments } from '../sqlKind';

/**
 * isDdlSql 是「执行成功后要不要失效本地元数据缓存」的唯一判据，
 * 漏判会让 SQL 补全继续给旧表清单/旧列，误判会让每次 DML 都白清一次缓存。
 */
describe('isDdlSql 结构变更语句判定', () => {
    it.each([
        'CREATE TABLE t (id INT)',
        'create  index idx on t (a)',
        '  ALTER TABLE t ADD COLUMN c VARCHAR(10)',
        '\nDROP TABLE t',
        'TRUNCATE TABLE t',
        'RENAME TABLE a TO b',
        // 注释与授权同样改变补全可见内容（表/列注释来源）
        "COMMENT ON COLUMN t.c IS '备注'",
        'GRANT SELECT ON t TO u',
        'REVOKE SELECT ON t FROM u',
        // 仅有关键字本身（无后续内容）也算结构语句
        'create',
        // 切割器保留注释原文，前导注释必须先跳过才能拿到真正首词
        '-- 加字段\nALTER TABLE t ADD COLUMN c INT',
        '/* 建表 */ CREATE TABLE t (id INT)',
        '# mysql 行注释\nDROP TABLE t',
        // 注释区域按迭代跳过：粘贴的 dump 脚本可能带成万行注释头，递归实现会在此爆栈
        `${'-- 注释行\n'.repeat(20000)}ALTER TABLE t ADD COLUMN c INT`,
    ])('识别 DDL: %s', (sql) => {
        expect(isDdlSql(sql)).toBe(true);
    });

    it.each([
        'SELECT * FROM t',
        'WITH cte AS (SELECT 1) SELECT * FROM cte',
        'INSERT INTO t VALUES (1)',
        'UPDATE t SET a = 1',
        'DELETE FROM t WHERE id = 1',
        'SHOW TABLES',
        // 关键字后必须断开：以关键字开头的标识符/列名不是 DDL
        'CREATE_TIME > 1',
        'dropped_rows = 0',
        // 注释里出现 DDL 关键字不得误判（语句本身仍是 DML）
        '-- create a row\nINSERT INTO t VALUES (1)',
        '/* alter me */ SELECT * FROM t',
        // 未闭合的块注释不再往下取词，交给执行层报错而非猜类型
        '/* 未闭合注释',
        '',
        '   ',
    ])('不识别为非 DDL: %s', (sql) => {
        expect(isDdlSql(sql)).toBe(false);
    });
});

/**
 * stripLeadingComments 是 isDdlSql 与 DbInst.isQuerySql 共用的「跳过前导注释取真正首词」前置。
 *
 * 关键契约：它只删除语句最前面的注释/空白，返回的永远是原文的一个后缀（verbatim），
 * 绝不改动注释之后的任何字节——否则判型用的文本与实际执行文本会分叉，
 * 更严重的是把 SQL 判型「顺手截断」。isQuerySql 只拿它做布尔判断，执行仍下发含注释原文。
 */
describe('stripLeadingComments 前导注释剥离', () => {
    it.each([
        // 行注释：-- 与 # 均以首个换行结束，返回其后原文
        ['-- 说明\nALTER TABLE t ADD COLUMN c INT', "ALTER TABLE t ADD COLUMN c INT"],
        ['# mysql 注释\nDROP TABLE t', 'DROP TABLE t'],
        // 块注释：以首个 */ 结束
        ['/* 建表 */ CREATE TABLE t (id INT)', 'CREATE TABLE t (id INT)'],
        // 多行/混合注释：迭代跳过，直到遇到真实代码
        ['-- a\n-- b\nDELETE FROM t', 'DELETE FROM t'],
        ['/* x */\n-- y\n\nDROP TABLE t', 'DROP TABLE t'],
        // 仅前导空白：无注释也去空白
        ['   \n\t SELECT 1', 'SELECT 1'],
        // 行注释内含分号/引号也只认换行终止，其后的语句完整保留
        ["-- 注意; 这里有分号 'x\nUPDATE t SET a='y;z'", "UPDATE t SET a='y;z'"],
        // 只剥前导：注释之后的行内注释/尾随注释原样保留（不越界改动代码）
        ['SELECT 1 -- 尾随注释', 'SELECT 1 -- 尾随注释'],
        ['-- lead\nSELECT a /* 内联 */, b', 'SELECT a /* 内联 */, b'],
        // 非注释起始（如 SELECT 里的 # 运算符、字符串里的 --）不触发剥离，原样返回
        ["SELECT '--不是注释'", "SELECT '--不是注释'"],
        ['SELECT a # b FROM t', 'SELECT a # b FROM t'],
    ])('剥离 %s => %s', (sql, expected) => {
        expect(stripLeadingComments(sql)).toBe(expected);
    });

    it.each([
        // 纯注释/仅注释：无真实代码，返回空串（交给执行层，不猜类型）
        ['-- 只是注释', ''],
        ['# 只是注释', ''],
        ['/* 只是注释 */', ''],
        // 未闭合块注释：返回空串而非把未闭合内容当语句
        ['/* 未闭合注释', ''],
        // 空/纯空白
        ['', ''],
        ['   \n ', ''],
    ])('无有效语句 %s => 空串', (sql, expected) => {
        expect(stripLeadingComments(sql)).toBe(expected);
    });

    it('返回值恒为原文后缀，绝不改动注释之后的字节（防判型顺带截断 SQL）', () => {
        const samples = [
            '-- 说明\nALTER TABLE `t` ADD COLUMN c INT DEFAULT 0;',
            '/* a */ /* b */\n-- c\nUPDATE t SET x = \'a;b\' WHERE id = 1;',
            '# lead\nDELETE FROM t WHERE name = \'--x\';',
        ];
        for (const sql of samples) {
            const stripped = stripLeadingComments(sql);
            // 剥离结果必须是原文的结尾子串：证明它只删前导，未在中途改写/丢弃任何字符
            expect(sql.endsWith(stripped), sql).toBe(true);
        }
    });

    it('dump 脚本的海量注释头按迭代跳过，不爆栈', () => {
        const sql = `${'-- 注释行\n'.repeat(20000)}ALTER TABLE t ADD COLUMN c INT`;
        expect(stripLeadingComments(sql)).toBe('ALTER TABLE t ADD COLUMN c INT');
    });
});

/**
 * isQueryStatement 只读/写入归类：与后端 sqlparser.stmtTypeFromKeyword 的只读集合严格对齐。
 *
 * 后端把 select/with/table 归 Select、show/explain/describe/desc 归 Other（均走查询通道回结果集），
 * 其余一律 doExec 按写入执行。前端只有与此同集合，才不会把写入误当查询而漏弹执行确认（危险），
 * 也不会把只读误当写入而错乱结果展示。未命中一律按写入（fail-closed）。
 */
describe('isQueryStatement 只读/写入归类', () => {
    it.each([
        'SELECT * FROM t',
        '  select 1',
        'WITH cte AS (SELECT 1) SELECT * FROM cte',
        'TABLE orders', // mssql 的 TABLE 查询
        '(SELECT 1) UNION (SELECT 2)', // 括号查询/UNION，后端按 SelectStmt 解析
        'SHOW TABLES',
        'SHOW CREATE TABLE t',
        'DESC users',
        'DESCRIBE users',
        'EXPLAIN SELECT * FROM t',
        // 前导注释必须先剥离再取首词
        '-- 查一下\nSELECT * FROM t',
        '/* 统计 */ SELECT COUNT(*) FROM t',
        '# 说明\nWITH x AS (SELECT 1) SELECT * FROM x',
    ])('只读 → 查询(true): %s', (sql) => {
        expect(isQueryStatement(sql)).toBe(true);
    });

    it.each([
        'INSERT INTO t VALUES (1)',
        'REPLACE INTO t VALUES (1)', // mysql 写入：前缀黑名单旧实现会漏判为查询
        'UPDATE t SET a = 1',
        'DELETE FROM t WHERE id = 1',
        'MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET a = 1',
        'CALL my_proc()',
        'CREATE TABLE t (id INT)',
        'ALTER TABLE t ADD COLUMN c INT',
        'DROP TABLE t',
        'TRUNCATE TABLE t',
        'GRANT SELECT ON t TO u',
        "COMMENT ON COLUMN t.c IS '备注'",
        // 裸 VALUES 后端归未识别→doExec（写入通道），前端一致按非查询
        'VALUES (1), (2)',
        // 整词边界：以关键字开头的标识符不是对应语句
        'table_name = 1',
        'selected_count > 0',
        'dropped_rows = 0',
        // 注释里出现只读关键字但语句本身是写入
        '-- select it\nUPDATE t SET a = 1',
        '/* show me */ DELETE FROM t',
        '',
        '   ',
    ])('写入/DDL → 非查询(false): %s', (sql) => {
        expect(isQueryStatement(sql)).toBe(false);
    });

    it('方言追加只读关键字：sqlite 的 pragma 命中追加集才归查询', () => {
        // 未追加时按写入（fail-closed）
        expect(isQueryStatement('PRAGMA table_info(users)')).toBe(false);
        // 方言声明追加后归查询
        expect(isQueryStatement('PRAGMA table_info(users)', { extraReadKeywords: ['pragma'] })).toBe(true);
    });

    it('方言追加项走整词匹配，不得被子串命中', () => {
        // pragma_x 不是 pragma，即便方言追加了 pragma 也不归查询
        expect(isQueryStatement('pragma_x = 1', { extraReadKeywords: ['pragma'] })).toBe(false);
    });
});

/**
 * 首词是只读、但体内藏写入的形态：必须降级为写入（弹确认、合并 tab）。
 *
 * 同时验证不误伤：纯只读 CTE/普通查询、以及同名词出现在字符串/注释里时，不得被误判为写入。
 */
describe('isQueryStatement 首词只读但体内藏写入', () => {
    it.each([
        'WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d',
        'WITH u AS (UPDATE t SET a = 1 RETURNING *) SELECT * FROM u',
        'WITH i AS (INSERT INTO t VALUES (1) RETURNING *) SELECT * FROM i',
        'SELECT * INTO bak_users FROM users',
        'EXPLAIN ANALYZE DELETE FROM t WHERE id = 1',
        'EXPLAIN (ANALYZE, BUFFERS) UPDATE t SET a = 1',
    ])('体内写入 → 非查询(false): %s', (sql) => {
        expect(isQueryStatement(sql)).toBe(false);
    });

    it.each([
        // 纯只读 CTE / 普通查询不误伤
        'WITH cte AS (SELECT id FROM t) SELECT * FROM cte',
        'SELECT * FROM t',
        'SELECT id FROM t ORDER BY id',
        // EXPLAIN 不带 analyze 只出计划、不执行，仍是读
        'EXPLAIN SELECT * FROM t',
        'EXPLAIN DELETE FROM t',
        // 字符串/注释里的同名词不算体内写入
        "SELECT 'into the wild' AS title FROM songs",
        "SELECT id FROM t WHERE note = 'do not delete from prod'",
        'WITH x AS (SELECT delete_flag, updated_at FROM t) SELECT * FROM x',
        // REPLACE() 是 mysql 常见字符串函数，不能因体内出现 replace 就把只读 CTE 误判为写入
        "WITH x AS (SELECT replace(name, 'a', 'b') FROM t) SELECT * FROM x",
        '-- 这里提到 insert 只是注释\nWITH x AS (SELECT 1) SELECT * FROM x',
    ])('只读不误伤(true): %s', (sql) => {
        expect(isQueryStatement(sql)).toBe(true);
    });
});
