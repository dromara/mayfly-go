import type { SqlExecRes } from '../../types/sqlExec';

/**
 * 一次执行请求的结果归类：把「执行失败」与「被策略要求审批」分开。
 *
 * 后端对被拦语句返回的是 errorMsg + needApproval 两个都有值的条目——errorMsg 只是
 * 「该操作需要提交工单审批执行」这类提示，语句根本没执行。直接按 errorMsg 判失败
 * 会把它渲染成红色报错，操作者以为出了错，真实情况只是等审批（且提单入口就此丢失）。
 *
 * 「仅提醒」（warnAck）不会出现在这里：执行确认框调用链不带确认位，后端不产生该结论。
 */
export interface SqlExecOutcome {
    /** 第一条被触发策略要求审批的语句原文（未执行，提单预填用）；无则空串 */
    needApprovalSql: string;
    /** 真正执行失败的语句；被拦的语句不会出现在这里 */
    errors: SqlExecRes[];
}

export const classifySqlExecRes = (res: SqlExecRes[]): SqlExecOutcome => {
    let needApprovalSql = '';
    const errors: SqlExecRes[] = [];
    for (const re of res) {
        // 结构化标记优先：策略拦截的 errorMsg 是提示文案，不是失败原因
        if (re.needApproval && re.sql) {
            needApprovalSql = needApprovalSql || re.sql;
            continue;
        }
        if (re.errorMsg) {
            errors.push(re);
        }
    }
    return { needApprovalSql, errors };
};
