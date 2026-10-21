/**
 * DbInst.isQuerySql 的委托接线与集成测试。
 *
 * isQuerySql 委托 core/sqlKind.isQueryStatement——完整关键字矩阵在 sqlKind.test.ts（纯函数层）覆盖，
 * 本文件不重复列举，只验「接线」：前导注释已剥离、fail-closed 写入、体内藏写入（依赖 splitOptions
 * 透传才能区分代码区）、以及方言只读词追加。并以「切割 + 归类」集成用例守住数据安全底线：
 * 归类只读文本、不回写执行体，送后端执行的永远是含注释的完整原文，绝不因判型而截断 SQL。
 */
import { describe, expect, it } from 'vitest';

import { DbInst } from '../db';
import { getSqlSplitOptions, splitSqlStatements } from '../sql-editor/utils/sqlParser';

// isQuerySql 不依赖实例状态，用裸实例即可（构造仅初始化 dbs: Map，无副作用；type 未设回退 mysql 方言）
const inst = new DbInst();

/**
 * 接线：证明 DbInst.isQuerySql 正确把「方言只读词 + 切割语义」透传给 isQueryStatement。
 * 只取代表用例，穷举留给 sqlKind.test.ts。
 */
describe('isQuerySql 委托 isQueryStatement（接线）', () => {
    it.each([
        // 前导注释已剥离后判型（现场问题）；仅前导空白同样
        '-- 保函主表：优惠承担方\nALTER TABLE `t_gua` ADD COLUMN `discount_bearer` VARCHAR(16)',
        '   UPDATE t SET a = 1',
        // fail-closed：非只读白名单的动词一律按写入
        'REPLACE INTO t VALUES (1)',
        'CALL my_proc()',
        // 体内藏写入：需 splitOptions 区分代码区，能判写入即证明 opts 已透传
        'WITH d AS (DELETE FROM t RETURNING id) SELECT * FROM d',
        'SELECT * INTO bak FROM t',
        'EXPLAIN ANALYZE DELETE FROM t',
        // MERGE（upsert）写语句：裸 MERGE 头不在只读白名单→写入；WITH 前置 MERGE 靠体内关键词检出→写入（均会弹备注确认）
        'MERGE INTO t USING s ON (t.id = s.id) WHEN MATCHED THEN UPDATE SET t.a = s.a',
        'WITH c AS (SELECT id FROM t) MERGE INTO u USING c ON (u.id = c.id) WHEN MATCHED THEN UPDATE SET u.a = 1',
    ])('写入/DDL → 非查询(false): %s', (sql) => {
        expect(inst.isQuerySql(sql)).toBe(false);
    });

    it.each([
        'SELECT * FROM t',
        '-- 查一下\nSELECT * FROM t',
        'SHOW TABLES',
        'TABLE orders',
        '(SELECT 1) UNION (SELECT 2)',
        'WITH cte AS (SELECT id FROM t) SELECT * FROM cte',
        // REPLACE() 函数不误伤：opts 透传后代码区判定正确
        "WITH x AS (SELECT replace(name, 'a', 'b') FROM t) SELECT * FROM x",
    ])('只读 → 查询(true): %s', (sql) => {
        expect(inst.isQuerySql(sql)).toBe(true);
    });
});

/**
 * isQuerySql 委托方言：通用基线之外的只读语句由方言能力追加（开闭原则）。
 */
describe('isQuerySql 委托方言追加只读关键字', () => {
    const sqliteInst = new DbInst();
    sqliteInst.type = 'sqlite';
    const mysqlInst = new DbInst();
    mysqlInst.type = 'mysql';

    it('sqlite 把 PRAGMA 视为只读查询（方言能力追加）', () => {
        expect(sqliteInst.isQuerySql('PRAGMA table_info(users)')).toBe(true);
    });

    it('非 sqlite 方言不认 PRAGMA（未列入通用基线 → 按写入处理）', () => {
        expect(mysqlInst.isQuerySql('PRAGMA table_info(users)')).toBe(false);
    });
});

/**
 * 切割 + 归类集成：守住「判型不截断执行」这条底线。
 *
 * 归类读的是切割器产出的完整语句文本；送后端执行的也是同一段原文。此处断言：
 * 1) 每条语句文本逐字来自原文（含前导注释与 COMMENT 字面量），首尾都不缺；
 * 2) 注释/字面量里的分号不会把语句切错；
 * 3) 归类结果符合预期（非查询合并、查询独立）。
 */
describe('isQuerySql × splitSqlStatements：归类不截断执行文本', () => {
    const mysql = getSqlSplitOptions('mysql');
    const splitTexts = (sql: string) => splitSqlStatements(sql, ';', mysql).map((s) => s.text);

    it('现场脚本：两条带注释的 ALTER 均判非查询，且原文完整保留', () => {
        const script = `-- 保函主表：优惠承担方
ALTER TABLE \`t_gua\`
    ADD COLUMN \`discount_bearer\` VARCHAR(16) DEFAULT NULL COMMENT '优惠承担方（用券时从券模板带出）：SP-服务商 DS-平台方 BOTH-双方（空=双方）' AFTER \`discount_amount\`;
-- 对账单明细：快照自 t_gua
ALTER TABLE \`t_gua_bill_item\`
    ADD COLUMN \`discount_bearer\` VARCHAR(16) DEFAULT NULL COMMENT '优惠承担方（快照自 t_gua）：SP-服务商 DS-平台方 BOTH-双方（空=双方）' AFTER \`discount_amount\`;`;

        const stmts = splitTexts(script);
        expect(stmts).toHaveLength(2);

        // 两条都判非查询 → 合并进同一个 tab（第一列显示 SQL），不再各占一个 tab
        expect(stmts.map((s) => inst.isQuerySql(s))).toEqual([false, false]);

        // 每条语句文本都是原文的连续子串（逐字保留，未截断）
        for (const s of stmts) {
            expect(script.includes(s)).toBe(true);
        }
        // 首（前导注释）与尾（AFTER 子句）都在，证明两端未被裁掉
        expect(stmts[0]).toContain('保函主表');
        expect(stmts[0]).toContain('AFTER `discount_amount`');
        expect(stmts[1]).toContain('对账单明细');
        expect(stmts[1]).toContain('AFTER `discount_amount`');
    });

    it('混合脚本：UPDATE 归非查询、SELECT 归查询，两条文本互不吞并', () => {
        const script = `-- 改数据
UPDATE t SET a = 1 WHERE id = 1;
-- 查结果
SELECT * FROM t;`;

        const stmts = splitTexts(script);
        expect(stmts).toHaveLength(2);
        expect(stmts.map((s) => inst.isQuerySql(s))).toEqual([false, true]);
        expect(stmts[0]).toBe('-- 改数据\nUPDATE t SET a = 1 WHERE id = 1');
        expect(stmts[1]).toBe('-- 查结果\nSELECT * FROM t');
    });

    it('注释里的分号不切错语句，归类仍按真实首词', () => {
        const script = `-- 注意; 这里有分号
DELETE FROM t WHERE id = 1;
SELECT 2;`;

        const stmts = splitTexts(script);
        // 行注释内的分号不切割：仍是两条语句，第一条完整含注释与分号文本
        expect(stmts).toHaveLength(2);
        expect(stmts[0]).toBe("-- 注意; 这里有分号\nDELETE FROM t WHERE id = 1");
        expect(stmts.map((s) => inst.isQuerySql(s))).toEqual([false, true]);
    });

    it('字面量里的分号与注释起始符不干扰切割与归类', () => {
        const script = `-- 说明
INSERT INTO t(name, remark) VALUES('张三', '含分号;与--破折号');
SELECT 2;`;

        const stmts = splitTexts(script);
        expect(stmts).toHaveLength(2);
        expect(stmts[0]).toBe("-- 说明\nINSERT INTO t(name, remark) VALUES('张三', '含分号;与--破折号')");
        expect(inst.isQuerySql(stmts[0])).toBe(false);
        expect(inst.isQuerySql(stmts[1])).toBe(true);
    });
});
