import { Enum } from '@/common/Enum'

/**
 * 枚举类
 */
export default {
    // 数据库sql执行类型
    DbSqlExecTypeEnum: new Enum().add('UPDATE', 'UPDATE', 1)
        .add('DELETE', 'DELETE', 2)
        .add('INSERT', 'INSERT', 3),
}