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
    [key: string]: unknown;
}
