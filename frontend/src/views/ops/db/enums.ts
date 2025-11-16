import { EnumValue } from '@/common/Enum';

export const DbGetDbNamesMode = {
    Auto: EnumValue.of(-1, 'db.getDbNamesModeAuto').setTagType('warning'),
    Assign: EnumValue.of(1, 'db.getDbNamesModeAssign').setTagType('primary'),
};

// 数据库sql执行类型
export const DbSqlExecTypeEnum = {
    Update: EnumValue.of(1, 'UPDATE').setTagColor('#E4F5EB'),
    Delete: EnumValue.of(2, 'DELETE').setTagColor('#F9E2AE'),
    Insert: EnumValue.of(3, 'INSERT').setTagColor('#A8DEE0'),
    Query: EnumValue.of(4, 'QUERY').setTagColor('#A8DEE0'),
    Ddl: EnumValue.of(5, 'DDL').setTagColor('#F9E2AE'),
    Other: EnumValue.of(-1, 'OTHER').setTagColor('#F9E2AE'),
};

export const DbSqlExecStatusEnum = {
    Success: EnumValue.of(2, 'common.success').setTagType('success'),
    Fail: EnumValue.of(-2, 'common.fail').setTagType('danger'),
};
