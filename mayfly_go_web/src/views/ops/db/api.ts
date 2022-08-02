import Api from '@/common/Api';

export const dbApi = {
    // 获取权限列表
    dbs: Api.create("/dbs", 'get'),
    saveDb: Api.create("/dbs", 'post'),
    getAllDatabase: Api.create("/dbs/databases", 'post'),
    getDbPwd: Api.create("/dbs/{id}/pwd", 'get'),
    deleteDb: Api.create("/dbs/{id}", 'delete'),
    dumpDb: Api.create("/dbs/{id}/dump", 'post'),
    tableInfos: Api.create("/dbs/{id}/t-infos", 'get'),
    tableIndex: Api.create("/dbs/{id}/t-index", 'get'),
    tableDdl: Api.create("/dbs/{id}/t-create-ddl", 'get'),
    tableMetadata: Api.create("/dbs/{id}/t-metadata", 'get'),
    columnMetadata: Api.create("/dbs/{id}/c-metadata", 'get'),
    // 获取表即列提示
    hintTables: Api.create("/dbs/{id}/hint-tables", 'get'),
    sqlExec: Api.create("/dbs/{id}/exec-sql", 'post'),
    // 保存sql
    saveSql: Api.create("/dbs/{id}/sql", 'post'),
    // 获取保存的sql
    getSql: Api.create("/dbs/{id}/sql", 'get'),
    // 获取保存的sql names
    getSqlNames: Api.create("/dbs/{id}/sql-names", 'get'),
    deleteDbSql: Api.create("/dbs/{id}/sql", 'delete'),
    // 获取数据库sql执行记录
    getSqlExecs: Api.create("/dbs/{dbId}/sql-execs", 'get'),
}