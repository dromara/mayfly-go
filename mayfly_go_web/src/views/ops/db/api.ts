import Api from '@/common/Api';

export const dbApi = {
    // 获取权限列表
    dbs: Api.newGet('/dbs'),
    saveDb: Api.newPost('/dbs'),
    getAllDatabase: Api.newPost('/dbs/databases'),
    getDbPwd: Api.newGet('/dbs/{id}/pwd'),
    deleteDb: Api.newDelete('/dbs/{id}'),
    dumpDb: Api.newPost('/dbs/{id}/dump'),
    tableInfos: Api.newGet('/dbs/{id}/t-infos'),
    tableIndex: Api.newGet('/dbs/{id}/t-index'),
    tableDdl: Api.newGet('/dbs/{id}/t-create-ddl'),
    tableMetadata: Api.newGet('/dbs/{id}/t-metadata'),
    columnMetadata: Api.newGet('/dbs/{id}/c-metadata'),
    // 获取表即列提示
    hintTables: Api.newGet('/dbs/{id}/hint-tables'),
    sqlExec: Api.newPost('/dbs/{id}/exec-sql'),
    // 保存sql
    saveSql: Api.newPost('/dbs/{id}/sql'),
    // 获取保存的sql
    getSql: Api.newGet('/dbs/{id}/sql'),
    // 获取保存的sql names
    getSqlNames: Api.newGet('/dbs/{id}/sql-names'),
    deleteDbSql: Api.newDelete('/dbs/{id}/sql'),
    // 获取数据库sql执行记录
    getSqlExecs: Api.newGet('/dbs/{dbId}/sql-execs'),
};
