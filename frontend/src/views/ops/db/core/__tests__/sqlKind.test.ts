import { describe, expect, it } from 'vitest';

import { isDdlSql } from '../sqlKind';

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
