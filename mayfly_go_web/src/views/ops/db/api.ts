import Api from '@/common/Api';
import { Base64 } from 'js-base64';

export const dbApi = {
    // 获取权限列表
    dbs: Api.newGet('/dbs'),
    dbTags: Api.newGet('/dbs/tags'),
    saveDb: Api.newPost('/dbs'),
    deleteDb: Api.newDelete('/dbs/{id}'),
    dumpDb: Api.newPost('/dbs/{id}/dump'),
    tableInfos: Api.newGet('/dbs/{id}/t-infos'),
    tableIndex: Api.newGet('/dbs/{id}/t-index'),
    tableDdl: Api.newGet('/dbs/{id}/t-create-ddl'),
    columnMetadata: Api.newGet('/dbs/{id}/c-metadata'),
    pgSchemas: Api.newGet('/dbs/{id}/pg/schemas'),
    // 获取表即列提示
    hintTables: Api.newGet('/dbs/{id}/hint-tables'),
    sqlExec: Api.newPost('/dbs/{id}/exec-sql').withBeforeHandler((param: any) => {
        // sql编码处理
        if (param.sql) {
            param.sql = Base64.encode(param.sql);
        }
        return param;
    }),
    // 保存sql
    saveSql: Api.newPost('/dbs/{id}/sql'),
    // 获取保存的sql
    getSql: Api.newGet('/dbs/{id}/sql'),
    // 获取保存的sql names
    getSqlNames: Api.newGet('/dbs/{id}/sql-names'),
    deleteDbSql: Api.newDelete('/dbs/{id}/sql'),
    // 获取数据库sql执行记录
    getSqlExecs: Api.newGet('/dbs/{dbId}/sql-execs'),

    // 获取权限列表
    instances: Api.newGet('/instances'),
    getInstance: Api.newGet('/instances/{instanceId}'),
    getAllDatabase: Api.newGet('/instances/{instanceId}/databases'),
    testConn: Api.newPost('/instances/test-conn'),
    saveInstance: Api.newPost('/instances'),
    getInstancePwd: Api.newGet('/instances/{id}/pwd'),
    deleteInstance: Api.newDelete('/instances/{id}'),
};
