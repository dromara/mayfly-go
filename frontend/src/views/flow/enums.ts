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
