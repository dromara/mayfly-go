import { EnumValue } from '@/common/Enum';

export const ProcdefStatus = {
    Enable: EnumValue.of(1, 'flow.enable').setTagType('success'),
    Disable: EnumValue.of(-1, 'flow.disable').setTagType('warning'),
};

export const UserTaskCandidateType = {
    Account: EnumValue.of('account', 'common.account'),
    Role: EnumValue.of('role', 'common.role'),
    Other: EnumValue.of('other', 'common.other'),
};

export const ProcinstStatus = {
    Active: EnumValue.of(1, 'flow.active').setTagType('primary'),
    Completed: EnumValue.of(2, 'flow.completed').setTagType('success'),
    Suspended: EnumValue.of(-1, 'flow.suspended').setTagType('warning'),
    Back: EnumValue.of(-11, 'flow.back').setTagType('warning'),
    Terminated: EnumValue.of(-2, 'flow.terminated').setTagType('danger'),
    Cancelled: EnumValue.of(-3, 'flow.cancelled').setTagType('warning'),
};

export const ProcinstBizStatus = {
    Wait: EnumValue.of(1, 'flow.waitHandle').setTagType('primary'),
    Success: EnumValue.of(2, 'flow.handleSuccess').setTagType('success'),
    Fail: EnumValue.of(-2, 'flow.handleFail').setTagType('danger'),
    No: EnumValue.of(-1, 'flow.noHandle').setTagType('warning'),
};

export const ProcinstTaskStatus = {
    Process: EnumValue.of(1, 'flow.waitProcess').setTagType('primary'),
    Pass: EnumValue.of(2, 'flow.pass').setTagType('success'),
    Reject: EnumValue.of(-1, 'flow.reject').setTagType('danger'),
    Back: EnumValue.of(-2, 'flow.back').setTagType('warning'),
    Canceled: EnumValue.of(-3, 'flow.canceled').setTagType('warning'),
};

export const HisProcinstOpState = {
    Pending: EnumValue.of(1, 'flow.waitProcess').setTagType('primary'),
    Completed: EnumValue.of(2, 'flow.pass').setTagType('success'),
    Failed: EnumValue.of(-1, 'flow.reject').setTagType('danger'),
};

export const FlowBizType = {
    DbSqlExec: EnumValue.of('db_sql_exec_flow', 'flow.dbSqlExec').setTagType('warning'),
    RedisRunWriteCmd: EnumValue.of('redis_run_cmd_flow', 'flow.redisRunCmd').setTagType('danger'),
};

/**
 * 流程内部条件（连线跳转、节点完成）使用的字段字典标识，与后端 flow.application.FlowInstanceBizType 一致。
 *
 * 前端只需要这一个已知标识来选中字典，字典里有哪些字段、能用哪些操作符全部由 policy-schema 下发
 */
export const FLOW_INSTANCE_BIZ_TYPE = 'flow_instance';

/** 策略变更动作，与后端 entity.ProcdefPolicyAction 一致 */
export const ProcdefPolicyAction = {
    Create: EnumValue.of(1, 'flow.policyHistory.create').setTagType('success'),
    Update: EnumValue.of(2, 'flow.policyHistory.update').setTagType('primary'),
    Delete: EnumValue.of(3, 'flow.policyHistory.delete').setTagType('danger'),
};
