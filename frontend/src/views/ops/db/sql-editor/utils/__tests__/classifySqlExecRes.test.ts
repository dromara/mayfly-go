/**
 * classifySqlExecRes 的归类语义测试。
 *
 * 这条归类是提单入口的唯一判据：被策略拦下的语句（needApproval）一旦混进 errors，
 * 就会渲染成红色执行失败且没有提单出口——操作者被卡死在确认框里。
 */
import { describe, expect, it } from 'vitest';
import { classifySqlExecRes } from '../classifySqlExecRes';

const ok = (sql: string) => ({ sql, res: [{ rowsAffected: 1 }] });
const approval = (sql: string) => ({
    sql,
    // 后端对被拦语句 errorMsg 与 needApproval 同值：errorMsg 只是提示文案，语句没执行
    errorMsg: '该操作需要提交工单审批执行，审批通过后会自动执行本次操作',
    needApproval: true,
});
const fail = (sql: string, msg = 'syntax error') => ({ sql, errorMsg: msg });

describe('classifySqlExecRes', () => {
    it('全部成功时两个桶都为空', () => {
        const outcome = classifySqlExecRes([ok('SELECT 1'), ok('UPDATE t SET a=1')]);
        expect(outcome.needApprovalSql).toBe('');
        expect(outcome.errors).toHaveLength(0);
    });

    it('被策略拦下的语句收进提单预填，不算执行失败', () => {
        const outcome = classifySqlExecRes([approval('UPDATE t SET a=1 WHERE id=1')]);
        expect(outcome.needApprovalSql).toBe('UPDATE t SET a=1 WHERE id=1');
        expect(outcome.errors).toHaveLength(0);
    });

    it('真正的执行失败进 errors，不给提单入口', () => {
        const outcome = classifySqlExecRes([fail('UPDATE t SET a=1')]);
        expect(outcome.needApprovalSql).toBe('');
        expect(outcome.errors).toHaveLength(1);
        expect(outcome.errors[0].sql).toBe('UPDATE t SET a=1');
    });

    it('混合批次：被拦语句与失败语句各归各桶', () => {
        const outcome = classifySqlExecRes([ok('SELECT 1'), approval('DROP TABLE t'), fail('INSERT INTO t VALUES(1)')]);
        expect(outcome.needApprovalSql).toBe('DROP TABLE t');
        expect(outcome.errors).toHaveLength(1);
        expect(outcome.errors[0].sql).toBe('INSERT INTO t VALUES(1)');
    });

    it('多条被拦语句取第一条原文预填', () => {
        const outcome = classifySqlExecRes([approval('UPDATE a SET x=1'), approval('UPDATE b SET y=1')]);
        expect(outcome.needApprovalSql).toBe('UPDATE a SET x=1');
    });

    it('needApproval 但语句为空时不进提单桶（无原文可预填，防御空回包）', () => {
        // SqlExecRes 带索引签名，该字面量本就可赋值，无需断言
        const outcome = classifySqlExecRes([{ needApproval: true }]);
        expect(outcome.needApprovalSql).toBe('');
    });

    it('禁止执行的语句（非 needApproval 的失败）进 errors：提单也不会执行，不能给入口', () => {
        const outcome = classifySqlExecRes([fail('DROP TABLE t', '该操作已被管理员禁止执行')]);
        expect(outcome.errors).toHaveLength(1);
        expect(outcome.needApprovalSql).toBe('');
    });
});
