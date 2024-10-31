import { EnumValue } from '@/common/Enum';

export const ProcdefStatus = {
    Enable: EnumValue.of(1, '启用').setTagType('success'),
    Disable: EnumValue.of(-1, '禁用').setTagType('warning'),
};

export const ProcinstStatus = {
    Active: EnumValue.of(1, '执行中').setTagType('primary'),
    Completed: EnumValue.of(2, '完成').setTagType('success'),
    Suspended: EnumValue.of(-1, '挂起').setTagType('warning'),
    Terminated: EnumValue.of(-2, '终止').setTagType('danger'),
    Cancelled: EnumValue.of(-3, '取消').setTagType('warning'),
};

export const ProcinstBizStatus = {
    Wait: EnumValue.of(1, '待处理').setTagType('primary'),
    Success: EnumValue.of(2, '处理成功').setTagType('success'),
    Fail: EnumValue.of(-2, '处理失败').setTagType('danger'),
    No: EnumValue.of(-1, '不处理').setTagType('warning'),
};

export const ProcinstTaskStatus = {
    Process: EnumValue.of(1, '待处理').setTagType('primary'),
    Pass: EnumValue.of(2, '通过').setTagType('success'),
    Reject: EnumValue.of(-1, '拒绝').setTagType('danger'),
    Back: EnumValue.of(-2, '驳回').setTagType('warning'),
    Canceled: EnumValue.of(-3, '取消').setTagType('warning'),
};

export const FlowBizType = {
    DbSqlExec: EnumValue.of('db_sql_exec_flow', 'DBMS-执行SQL').setTagType('warning'),
    RedisRunWriteCmd: EnumValue.of('redis_run_cmd_flow', 'Redis-执行命令').setTagType('danger'),
};
