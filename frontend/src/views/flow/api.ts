import Api from '@/common/Api';
import type { PageParam, PageResult } from '@/types/common';
import type {
    HisProcinstOp,
    PolicySchema,
    Procdef,
    ProcdefPolicyHis,
    ProcdefResourceParam,
    ProcdefVO,
    Procinst,
    ProcinstTask,
    RuleSegment,
    RuleNode,
    SimulateResult,
    TriggerPolicy,
} from './types';

/** 策略试算入参：只针对编辑器里的草稿策略求值，不落库也不回读已保存策略 */
export interface PolicySimulateParam {
    bizType: string;
    policy?: TriggerPolicy;
    raw: Record<string, string>;
}

export const procdefApi = {
    list: Api.newGet<PageResult<ProcdefVO>, PageParam>('/flow/procdefs'),
    detail: Api.newGet<ProcdefVO>('/flow/procdefs/detail/{id}'),
    flowDef: Api.newGet<Record<string, unknown>>('/flow/procdefs/flowdef/{id}'),
    getByResource: Api.newGet<Procdef, ProcdefResourceParam>('/flow/procdefs/{resourceType}/{resourceCode}'),
    save: Api.newPost<number>('/flow/procdefs'),
    saveFlowDef: Api.newPost<void>('/flow/procdefs/flowdef'),
    del: Api.newDelete<void>('/flow/procdefs/{id}'),
    /** 下发场景的字段字典、检查项与可复用条件组，驱动策略构建器与流程条件编辑器 */
    policySchema: Api.newGet<PolicySchema>('/flow/procdefs/policy-schema'),
    /** 策略变更时间线（最新的在前） */
    policyHistory: Api.newGet<ProcdefPolicyHis[]>('/flow/procdefs/policy-history/{id}'),
    /** 试算：粘贴一条 SQL / 命令看结论与命中链路 */
    policySimulate: Api.newPost<SimulateResult, PolicySimulateParam>('/flow/procdefs/policy-simulate'),
};

export const procinstApi = {
    list: Api.newGet<PageResult<Procinst>, PageParam>('/flow/procinsts'),
    start: Api.newPost<void>('/flow/procinsts/start'),
    detail: Api.newGet<Procinst>('/flow/procinsts/{id}'),
    cancel: Api.newPost<void>('/flow/procinsts/{id}/cancel'),
    // 定时执行：到点前改期与取消（仅「已批准、待执行」工单可操作）
    rescheduleExec: Api.newPost<void, { id: number; plannedExecTime?: string | null }>('/flow/procinsts/{id}/reschedule-exec'),
    cancelExec: Api.newPost<void, { id: number }>('/flow/procinsts/{id}/cancel-exec'),
    hisOp: Api.newGet<HisProcinstOp[]>('/flow/his-procinsts-op/{id}'),
};

/** 我的任务（待办/已办）查询参数：与后端 entity.ProcinstTaskQuery 对齐 */
export interface ProcinstTaskQuery extends PageParam {
    /** 流程任务状态：待办传 1（ProcinstTaskStatus.Process） */
    status?: number;
    bizType?: string;
    bizKey?: string;
}

export const procinstTaskApi = {
    // 待办/已办查询：status 为流程任务状态（首页指标卡只取待办），其余筛选条件与列表页同一模型
    tasks: Api.newGet<PageResult<ProcinstTask>, ProcinstTaskQuery>('/flow/procinsts/tasks'),
    passTask: Api.newPost<void>('/flow/procinsts/tasks/pass'),
    backTask: Api.newPost<void>('/flow/procinsts/tasks/back'),
    rejectTask: Api.newPost<void>('/flow/procinsts/tasks/reject'),
};

/** 可复用条件组：把常用判断沉淀成一条可被规则引用的数据 */
export interface RuleSegmentForm {
    id?: number;
    ref: string;
    name: string;
    bizType: string;
    remark?: string;
    ruleNode?: RuleNode;
}

export const ruleSegmentApi = {
    list: Api.newGet<PageResult<RuleSegment>, PageParam>('/flow/rule-segments'),
    save: Api.newPost<number, RuleSegmentForm>('/flow/rule-segments'),
    del: Api.newDelete<void>('/flow/rule-segments/{id}'),
};
