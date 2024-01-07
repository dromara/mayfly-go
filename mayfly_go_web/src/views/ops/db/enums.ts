import { EnumValue } from '@/common/Enum';

// 数据库sql执行类型
export const DbSqlExecTypeEnum = {
    Update: EnumValue.of(1, 'UPDATE').setTagColor('#E4F5EB'),
    Delete: EnumValue.of(2, 'DELETE').setTagColor('#F9E2AE'),
    Insert: EnumValue.of(3, 'INSERT').setTagColor('#A8DEE0'),
    Query: EnumValue.of(4, 'QUERY').setTagColor('#A8DEE0'),
    Other: EnumValue.of(-1, 'OTHER').setTagColor('#F9E2AE'),
};

export const DbDataSyncRecentStateEnum = {
    Success: EnumValue.of(1, '成功').setTagType('success'),
    Fail: EnumValue.of(-1, '失败').setTagType('danger'),
};

export const DbDataSyncLogStatusEnum = {
    Success: EnumValue.of(1, '成功').setTagType('success'),
    Wait: EnumValue.of(2, '同步中').setTagType('primary'),
    Fail: EnumValue.of(-1, '失败').setTagType('danger'),
};

export const DbDataSyncRunningStateEnum = {
    Success: EnumValue.of(1, '运行中').setTagType('success'),
    Wait: EnumValue.of(2, '待运行').setTagType('primary'),
    Fail: EnumValue.of(3, '已停止').setTagType('danger'),
};
