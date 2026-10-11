import { beforeEach, describe, expect, it, vi } from 'vitest';

// 只 mock 掉取数层：断言的是「弹窗状态如何随服务端结论变化」，不是请求怎么发
const request = vi.fn();
vi.mock('../../api', () => ({
    dbApi: {
        rollbackSql: {
            request: (...args: unknown[]) => request(...args),
        },
    },
}));

import { useRollbackSqlDialog } from '../useRollbackSql';

/**
 * 回滚 SQL 弹窗的状态测试。
 *
 * 这里钉两件事：
 * 1. 回滚 SQL 来自服务端而不是前端拼装——曾经前端自己拼过，联合主键只取第一个列、
 *    找不到主键就拿第一列当主键、字符串值不加转义、BIGINT 经 JSON 丢精度，
 *    生成出来的语句每一条都能「执行成功」，错的却是数据；
 * 2. 服务端拒绝生成时不开空弹窗：空框会被读成「这条记录没什么要还原的」，
 *    而真实情况是一条都没生成。
 */
describe('useRollbackSqlDialog', () => {
    beforeEach(() => request.mockReset());

    it('按执行记录 id 向服务端要回滚 SQL，并展示条数与如实说明', async () => {
        request.mockResolvedValue({
            execId: 9,
            type: 1,
            rows: 2,
            sqls: ["UPDATE t SET a = 'x' WHERE id = 1;", "UPDATE t SET a = 'y' WHERE id = 2;"],
            note: '旧值采集上限为 200 行',
        });
        const { dialog, show } = useRollbackSqlDialog();

        expect(await show(9)).toBe(true);
        expect(request).toHaveBeenCalledWith({ id: 9 });
        expect(dialog.visible).toBe(true);
        expect(dialog.rows).toBe(2);
        expect(dialog.sql.split('\n')).toHaveLength(2);
        // 截断说明必须由界面呈现，否则「生成了 2 条」会被当成「全都能改回去」
        expect(dialog.note).toContain('200');
    });

    it('服务端拒绝生成时不开弹窗、也不留旧内容', async () => {
        const first = useRollbackSqlDialog();
        request.mockResolvedValue({ execId: 1, type: 1, rows: 1, sqls: ['UPDATE t SET a = 1 WHERE id = 1;'] });
        await first.show(1);
        expect(first.dialog.visible).toBe(true);

        const second = useRollbackSqlDialog();
        request.mockImplementationOnce(async () => {
            throw new Error('表没有主键，无法定位到行');
        });
        expect(await second.show(2)).toBe(false);
        expect(second.dialog.visible).toBe(false);
        expect(second.dialog.sql).toBe('');
        expect(second.dialog.rows).toBe(0);
    });

    it('响应为空时等同于没生成：不开弹窗', async () => {
        request.mockResolvedValue(undefined);
        const { dialog, show } = useRollbackSqlDialog();
        expect(await show(3)).toBe(false);
        expect(dialog.visible).toBe(false);
    });

    it('sqls 缺省时呈现空文本而不是 undefined', async () => {
        request.mockResolvedValue({ execId: 4, type: 2, rows: 0 });
        const { dialog, show } = useRollbackSqlDialog();
        await show(4);
        expect(dialog.sql).toBe('');
        expect(dialog.sql).not.toContain('undefined');
    });

    it('未生成回滚的行随原因一起带进弹窗，换一条记录不留上一条的残留', async () => {
        const { dialog, show } = useRollbackSqlDialog();
        request.mockResolvedValueOnce({
            execId: 7,
            type: 1,
            rows: 1,
            sqls: ["UPDATE t SET a = 'x' WHERE id = 1;"],
            note: '其中 2 行在变更之后又被修改、删除或无法定位，未生成它们的回滚，需人工对账',
            verified: true,
            conflicts: [
                { key: 'id=2', reason: 'concurrent-modified' },
                { key: 'id=3', reason: 'row-missing' },
            ],
        });
        await show(7);
        // 逐行带原因展示：只给条数用户没法对账
        expect(dialog.conflicts).toHaveLength(2);
        expect(dialog.conflicts[0]).toEqual({ key: 'id=2', reason: 'concurrent-modified' });

        // 换一条记录再看：上一条的冲突清单不能留下来冒充这一条的对账结果
        request.mockResolvedValueOnce({ execId: 8, type: 3, rows: 1, sqls: ['DELETE FROM t WHERE id = 5;'] });
        await show(8);
        expect(dialog.conflicts).toHaveLength(0);
    });
});
