import Api from '@/common/Api';
import { AesEncrypt } from '@/common/crypto';
import type { PageParam, PageResult } from '@/types/common';
import { createSqlExecNotification, registerSqlExecAborter } from '@/components/system-message/db/db-sql-exec-progress';
import type { Db, DbInstance, DbSql, DbSqlExec, DbTableInfo, DbBackup, DbBackupHistory, DbRestore, DbInstanceServerInfo, ColumnMetadata, DbInstanceListParam, DbListParam, SqlExecRes, DbMaskRule, DbMaskColumn, DbMaskRuleQuery, DbMaskColumnQuery, DbMaskRuleSaveForm, DbMaskColumnSaveForm } from './types';

export const dbApi = {
    // 获取权限列表
    dbs: Api.newGet<PageResult<Db>, DbListParam>('/dbs'),
    dbTags: Api.newGet<Db[]>('/dbs/tags'),
    saveDb: Api.newPost<void>('/dbs'),
    deleteDb: Api.newDelete<void>('/dbs/{id}'),
    dumpDb: Api.newPost<void>('/dbs/{id}/dump'),
    tableInfos: Api.newGet<DbTableInfo[]>('/dbs/{id}/t-infos'),
    tableIndex: Api.newGet<Record<string, unknown>[]>('/dbs/{id}/t-index'),
    tableDdl: Api.newGet<string>('/dbs/{id}/t-create-ddl'),
    copyTable: Api.newPost<void>('/dbs/{id}/copy-table'),
    columnMetadata: Api.newGet<ColumnMetadata[]>('/dbs/{id}/c-metadata'),
    pgSchemas: Api.newGet<string[]>('/dbs/{id}/pg/schemas'),
    // 获取表即列提示
    hintTables: Api.newGet<string[]>('/dbs/{id}/hint-tables'),
    sqlExec: Api.newPost<SqlExecRes[], Record<string, unknown>>('/dbs/{id}/exec-sql').withBeforeHandler(async (param) => await encryptField(param, 'sql')),
    // 保存sql
    saveSql: Api.newPost<void>('/dbs/{id}/sql'),
    // 获取保存的sql
    getSql: Api.newGet<DbSql>('/dbs/{id}/sql'),
    // 获取保存的sql names
    getSqlNames: Api.newGet<DbSql[]>('/dbs/{id}/sql-names'),
    deleteDbSql: Api.newDelete<void>('/dbs/{id}/sql'),
    // 获取数据库sql执行记录
    getSqlExecs: Api.newGet<PageResult<DbSqlExec>, PageParam>('/dbs/sql-execs'),
    // 获取数据库兼容版本
    getCompatibleDbVersion: Api.newGet<string>('/dbs/{id}/version'),

    instances: Api.newGet<PageResult<DbInstance>, DbInstanceListParam>('/instances'),
    getInstance: Api.newGet<DbInstance>('/instances/{instanceId}'),
    getAllDatabase: Api.newPost<string[]>('/instances/databases'),
    getDbNamesByAc: Api.newGet<string[]>('/instances/databases/{authCertName}'),
    getInstanceServerInfo: Api.newGet<DbInstanceServerInfo>('/instances/{instanceId}/server-info'),
    testConn: Api.newPost<void>('/instances/test-conn'),
    saveInstance: Api.newPost<number>('/instances'),
    deleteInstance: Api.newDelete<void>('/instances/{id}'),

    // 获取数据库备份列表
    getDbBackups: Api.newGet<DbBackup[]>('/dbs/{dbId}/backups'),
    createDbBackup: Api.newPost<void>('/dbs/{dbId}/backups'),
    deleteDbBackup: Api.newDelete<void>('/dbs/{dbId}/backups/{backupId}'),
    getDbNamesWithoutBackup: Api.newGet<string[]>('/dbs/{dbId}/db-names-without-backup'),
    enableDbBackup: Api.newPut<void>('/dbs/{dbId}/backups/{backupId}/enable'),
    disableDbBackup: Api.newPut<void>('/dbs/{dbId}/backups/{backupId}/disable'),
    startDbBackup: Api.newPut<void>('/dbs/{dbId}/backups/{backupId}/start'),
    saveDbBackup: Api.newPut<void>('/dbs/{dbId}/backups/{id}'),
    getDbBackupHistories: Api.newGet<DbBackupHistory[]>('/dbs/{dbId}/backup-histories'),
    restoreDbBackupHistory: Api.newPost<void>('/dbs/{dbId}/backup-histories/{backupHistoryId}/restore'),
    deleteDbBackupHistory: Api.newDelete<void>('/dbs/{dbId}/backup-histories/{backupHistoryId}'),

    // 获取数据库恢复列表
    getDbRestores: Api.newGet<DbRestore[]>('/dbs/{dbId}/restores'),
    createDbRestore: Api.newPost<void>('/dbs/{dbId}/restores'),
    deleteDbRestore: Api.newDelete<void>('/dbs/{dbId}/restores/{restoreId}'),
    getDbNamesWithoutRestore: Api.newGet<string[]>('/dbs/{dbId}/db-names-without-restore'),
    enableDbRestore: Api.newPut<void>('/dbs/{dbId}/restores/{restoreId}/enable'),
    disableDbRestore: Api.newPut<void>('/dbs/{dbId}/restores/{restoreId}/disable'),
    saveDbRestore: Api.newPut<void>('/dbs/{dbId}/restores/{id}'),
};

export const dbSqlExecApi = {
    // 根据业务key获取sql执行信息
    getSqlExecByBizKey: Api.newGet<PageResult<DbSqlExec>, PageParam>('/dbs/sql-execs'),
};

export const dbMaskApi = {
    // 分页获取脱敏规则
    maskRules: Api.newGet<PageResult<DbMaskRule>, DbMaskRuleQuery>('/dbs/mask-rules'),
    // 保存脱敏规则
    saveMaskRule: Api.newPost<void, DbMaskRuleSaveForm>('/dbs/mask-rules'),
    // 修改脱敏规则
    updateMaskRule: Api.newPut<void, DbMaskRuleSaveForm>('/dbs/mask-rules'),
    // 删除脱敏规则
    deleteMaskRule: Api.newDelete<void, { id: number }>('/dbs/mask-rules/{id}'),

    // 分页获取脱敏列标签
    maskColumns: Api.newGet<PageResult<DbMaskColumn>, DbMaskColumnQuery>('/dbs/mask-columns'),
    // 保存脱敏列标签
    saveMaskColumn: Api.newPost<void, DbMaskColumnSaveForm>('/dbs/mask-columns'),
    // 修改脱敏列标签
    updateMaskColumn: Api.newPut<void, DbMaskColumnSaveForm>('/dbs/mask-columns'),
    // 删除脱敏列标签
    deleteMaskColumn: Api.newDelete<void, { id: number }>('/dbs/mask-columns/{id}'),
};
export const encryptField = async (param: Record<string, unknown>, field: string) => {
    // sql编码处理
    if (!param['_encrypted'] && param[field]) {
        // 使用aes加密sql
        param['_encrypted'] = 1;
        param[field] = AesEncrypt(param[field] as string);
        // console.log('解密结果', DesDecrypt(param[field]));
    }
    return param;
};

/**
 * 上传SQL文件并执行
 * @param file 文件对象
 * @param params 上传参数
 * @param options 上传选项
 * @returns { uploadId: string; abort: () => void } 返回包含 uploadId 和中止方法的对象
 */
export function uploadSqlFile(
    file: File,
    params: {
        dbId: number;
        dbName: string;
    },
    options: {
        onSuccess?: () => void;
        onError?: (error: Error) => void;
    } = {}
): { uploadId: string; abort: () => void } {
    // 生成 uploadId
    const uploadId = `sql_exec_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;

    // 构建查询参数对象
    const queryParams: Record<string, string> = {
        db: params.dbName,
        uploadId: uploadId,
        filename: file.name,
    };

    // 创建 Api 实例
    const api = Api.newPost(`/dbs/${params.dbId}/exec-sql-file`);

    // 使用 uploadRaw 直接传递文件流
    const { abort } = api.uploadRaw(file, queryParams, {
        onSuccess: () => {
            options.onSuccess?.();
        },
        onError: (error) => {
            options.onError?.(error);
        },
    });

    // 创建SQL执行进度通知
    createSqlExecNotification(uploadId, {
        id: uploadId,
        title: file.name,
        dbCode: '',
        dbName: params.dbName,
        executedStatements: 0,
        terminated: false,
        status: 'uploading',
        clientId: '',
    });

    // 注册取消器（在获取到abort方法后）
    registerSqlExecAborter(uploadId, abort);

    return { uploadId, abort };
}
