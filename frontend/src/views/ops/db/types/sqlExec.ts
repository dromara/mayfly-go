/**
 * DB 模块 SQL 执行结果类型定义
 */

/** SQL执行结果列信息 (exec-sql 返回的 columns 元素) */
export interface SqlExecResColumn {
    name: string;
    key?: string;
    type?: string;
    /** 是否为脱敏列 (服务端 doQuery 脱敏后下发) */
    masked?: boolean;
    [key: string]: unknown;
}

/** 策略提醒：不阻断执行，但操作者需要知道（后端只下发 i18n key，文案在前端） */
export interface PolicyNotice {
    title: string;
    detail?: Record<string, unknown>;
}

/** SQL执行结果 (后端 exec-sql 接口返回数组的元素) */
export interface SqlExecRes {
    /** 执行的sql */
    sql?: string;
    /** 错误信息，存在则表示该sql执行失败 */
    errorMsg?: string;
    /** 结果集列信息 */
    columns?: SqlExecResColumn[];
    /** 结果集数据行 */
    res?: Record<string, unknown>[];
    affectedRows?: number;
    execTime?: string;
    /** 命中的「仅提醒」级策略结论 */
    notices?: PolicyNotice[];
    /** 该语句因触发策略需提交工单审批（可提单），与「已被禁止执行」互斥 */
    needApproval?: boolean;
    /**
     * 该语句命中「仅提醒」，等待操作者确认（语句未执行）。
     *
     * 与 needApproval 一样必须是结构化标记：确认后是重发同一请求，提单是打开抽屉，
     * 按提示文案区分会在改措辞或切换语言时把用户引向错误方向
     */
    warnAck?: boolean;
    [key: string]: unknown;
}
