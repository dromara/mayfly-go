import { EnumValue } from '@/common/Enum';

// 数据库sql执行类型
export const DbSqlExecTypeEnum = {
    Update: EnumValue.of(1, 'UPDATE').setTagColor('#E4F5EB'),
    Delete: EnumValue.of(2, 'DELETE').setTagColor('#F9E2AE'),
    Insert: EnumValue.of(3, 'INSERT').setTagColor('#A8DEE0'),
    Query: EnumValue.of(4, 'QUERY').setTagColor('#A8DEE0'),
    Other: EnumValue.of(-1, 'OTHER').setTagColor('#F9E2AE'),
};
