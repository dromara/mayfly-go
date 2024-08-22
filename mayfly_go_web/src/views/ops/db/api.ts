import Api from '@/common/Api';
import { DesEncrypt } from '@/common/des';

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
    copyTable: Api.newPost('/dbs/{id}/copy-table'),
    columnMetadata: Api.newGet('/dbs/{id}/c-metadata'),
    pgSchemas: Api.newGet('/dbs/{id}/pg/schemas'),
    // 获取表即列提示
    hintTables: Api.newGet('/dbs/{id}/hint-tables'),
    sqlExec: Api.newPost('/dbs/{id}/exec-sql').withBeforeHandler(async (param: any) => await encryptField(param, 'sql')),
    // 保存sql
    saveSql: Api.newPost('/dbs/{id}/sql'),
    // 获取保存的sql
    getSql: Api.newGet('/dbs/{id}/sql'),
    // 获取保存的sql names
    getSqlNames: Api.newGet('/dbs/{id}/sql-names'),
    deleteDbSql: Api.newDelete('/dbs/{id}/sql'),
    // 获取数据库sql执行记录
    getSqlExecs: Api.newGet('/dbs/sql-execs'),

    instances: Api.newGet('/instances'),
    getInstance: Api.newGet('/instances/{instanceId}'),
    getAllDatabase: Api.newPost('/instances/databases'),
    getDbNamesByAc: Api.newGet('/instances/databases/{authCertName}'),
    getInstanceServerInfo: Api.newGet('/instances/{instanceId}/server-info'),
    testConn: Api.newPost('/instances/test-conn'),
    saveInstance: Api.newPost('/instances'),
    deleteInstance: Api.newDelete('/instances/{id}'),

    // 获取数据库备份列表
    getDbBackups: Api.newGet('/dbs/{dbId}/backups'),
    createDbBackup: Api.newPost('/dbs/{dbId}/backups'),
    deleteDbBackup: Api.newDelete('/dbs/{dbId}/backups/{backupId}'),
    getDbNamesWithoutBackup: Api.newGet('/dbs/{dbId}/db-names-without-backup'),
    enableDbBackup: Api.newPut('/dbs/{dbId}/backups/{backupId}/enable'),
    disableDbBackup: Api.newPut('/dbs/{dbId}/backups/{backupId}/disable'),
    startDbBackup: Api.newPut('/dbs/{dbId}/backups/{backupId}/start'),
    saveDbBackup: Api.newPut('/dbs/{dbId}/backups/{id}'),
    getDbBackupHistories: Api.newGet('/dbs/{dbId}/backup-histories'),
    restoreDbBackupHistory: Api.newPost('/dbs/{dbId}/backup-histories/{backupHistoryId}/restore'),
    deleteDbBackupHistory: Api.newDelete('/dbs/{dbId}/backup-histories/{backupHistoryId}'),

    // 获取数据库恢复列表
    getDbRestores: Api.newGet('/dbs/{dbId}/restores'),
    createDbRestore: Api.newPost('/dbs/{dbId}/restores'),
    deleteDbRestore: Api.newDelete('/dbs/{dbId}/restores/{restoreId}'),
    getDbNamesWithoutRestore: Api.newGet('/dbs/{dbId}/db-names-without-restore'),
    enableDbRestore: Api.newPut('/dbs/{dbId}/restores/{restoreId}/enable'),
    disableDbRestore: Api.newPut('/dbs/{dbId}/restores/{restoreId}/disable'),
    saveDbRestore: Api.newPut('/dbs/{dbId}/restores/{id}'),

    // 数据同步相关
    datasyncTasks: Api.newGet('/datasync/tasks'),
    saveDatasyncTask: Api.newPost('/datasync/tasks/save').withBeforeHandler(async (param: any) => await encryptField(param, 'dataSql')),
    getDatasyncTask: Api.newGet('/datasync/tasks/{taskId}'),
    deleteDatasyncTask: Api.newDelete('/datasync/tasks/{taskId}/del'),
    updateDatasyncTaskStatus: Api.newPost('/datasync/tasks/{taskId}/status'),
    runDatasyncTask: Api.newPost('/datasync/tasks/{taskId}/run'),
    stopDatasyncTask: Api.newPost('/datasync/tasks/{taskId}/stop'),
    datasyncLogs: Api.newGet('/datasync/tasks/{taskId}/logs'),

    // 数据库迁移相关
    dbTransferTasks: Api.newGet('/dbTransfer'),
    saveDbTransferTask: Api.newPost('/dbTransfer/save'),
    deleteDbTransferTask: Api.newDelete('/dbTransfer/{taskId}/del'),
    runDbTransferTask: Api.newPost('/dbTransfer/{taskId}/run'),
    stopDbTransferTask: Api.newPost('/dbTransfer/{taskId}/stop'),
    dbTransferTaskLogs: Api.newGet('/dbTransfer/{taskId}/logs'),
};

export const dbSqlExecApi = {
    // 根据业务key获取sql执行信息
    getSqlExecByBizKey: Api.newGet('/dbs/sql-execs'),
};
const encryptField = async (param: any, field: string) => {
    // sql编码处理
    if (!param['_encrypted'] && param[field]) {
        // 判断是开发环境就打印sql
        if (process.env.NODE_ENV === 'development') {
            console.log(param[field]);
        }
        // 使用rsa公钥加密sql
        param['_encrypted'] = 1;
        param[field] = DesEncrypt(param[field]);
        // console.log('解密结果', DesDecrypt(param[field]));
    }
    return param;
};
