import { readFileSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

/**
 * SQL 执行记录筛选的契约守卫
 *
 * 存在理由：这个页面的默认查询条件一旦写死状态集合，「待执行 / 不执行」的记录就会在审计列表里
 * 永远看不见（不是报错，只是查不到），而界面看起来一切正常。这类「默认值把一类记录藏起来」的
 * 回归无法靠渲染断言稳定捕获（要 mock 整条 DataTable 取数链），故直接锁源码里的默认条件与筛选项。
 */

const FILE = path.resolve(__dirname, '../DbSqlExecLog.vue');
const src = readFileSync(FILE, 'utf8');

describe('DbSqlExecLog 筛选条件', () => {
    it('默认查询条件不得写死状态集合（否则待执行/不执行的记录永远查不到）', () => {
        expect(src, 'status 默认值必须为空，由用户显式选择状态').toMatch(/status:\s*''\s+as number \| ''/);
        expect(src, '不允许再把默认状态拼成「成功+失败」').not.toMatch(/status:\s*\[DbSqlExecStatusEnum/);
    });

    it('必须提供状态筛选项（排查失败时只想要失败记录）', () => {
        expect(src).toMatch(/SearchItem\.select\('status',\s*'common\.status'\)\.withEnum\(DbSqlExecStatusEnum\)/);
    });

    it('SQL 全文模糊与时间范围筛选都要在位', () => {
        expect(src).toMatch(/SearchItem\.input\('keyword',\s*'db\.sqlKeyword'\)/);
        expect(src).toMatch(/SearchItem\.datePicker\('execTimeRange',\s*'db\.execTime'\)/);
    });
});
