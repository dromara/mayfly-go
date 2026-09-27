import Api from '@/common/Api';
import { AesEncrypt } from '@/common/crypto';
import { getClientId } from '@/common/utils/storage';
import type { PageParam, PageResult } from '@/types/common';
import { createSqlExecNotification, registerSqlExecAborter } from '@/components/system-message/db/db-sql-exec-progress';
import { createDataImportNotification, finishDataImportNotification } from '@/components/system-message/db/db-data-import-progress';
import type {
    Db,
    DbInstance,
    DbSql,
    DbSqlExec,
    DbTableInfo,
    DbBackup,
    DbBackupHistory,
    DbRestore,
    DbInstanceServerInfo,
    ColumnMetadata,
    DbInstanceListParam,
    DbListParam,
    SqlExecRes,
    DbMaskRule,
    DbMaskColumn,
    DbMaskRuleQuery,
    DbMaskColumnQuery,
    DbMaskRuleSaveForm,
    DbMaskColumnSaveForm,
    DbCapabilities,
    DbMetadataObject,
    ImportColumn,
    ImportPreviewResult,
    DataImportResult,
} from './types';

export const dbApi = {
    // 获取权限列表
    dbs: Api.newGet<PageResult<Db>, DbListParam>('/dbs'),
    dbTags: Api.newGet<Db[]>('/dbs/tags'),
    saveDb: Api.newPost<number>('/dbs'),
    deleteDb: Api.newDelete<void>('/dbs/{id}'),
    dumpDb: Api.newPost<void>('/dbs/{id}/dump'),
    // 表清单：like/limit 为可选的服务端名称过滤下推（超大 schema 资源树按需加载）
    tableInfos: Api.newGet<DbTableInfo[], { id: number; db: string; like?: string; limit?: number }>('/dbs/{id}/t-infos'),
    // 扩展元数据对象节点（视图/序列/存储过程…），由后端 MetaNavigator 提供，按 features 能力位决定是否请求
    metaObjects: Api.newGet<DbMetadataObject[], { id: number; db: string; kind: string; schema?: string }>('/dbs/{id}/meta-objects'),
    // 单个扩展对象的 DDL 原文（点开节点查看）
    metaObjectDdl: Api.newGet<string, { id: number; db: string; kind: string; schema?: string; name: string }>('/dbs/{id}/meta-object-ddl'),
    // 方言能力协商：能力位（features）+ 命名空间层次的单一事实源，前端据此数据驱动渲染、不再各自硬编码
    capabilities: Api.newGet<DbCapabilities, { id: number; db: string }>('/dbs/{id}/capabilities'),
    tableIndex: Api.newGet<Record<string, unknown>[]>('/dbs/{id}/t-index'),
    tableDdl: Api.newGet<string>('/dbs/{id}/t-create-ddl'),
    copyTable: Api.newPost<void>('/dbs/{id}/copy-table'),
    // 获取指定表的列元数据（列名/类型/注释/主键等）：SQL 补全按表 fragment 取列与表结构/数据编辑共用
    columnMetadata: Api.newGet<ColumnMetadata[]>('/dbs/{id}/c-metadata'),
    // 获取库的 schema 列表：后端 GetSchemas 是各方言通用实现（pg/oracle/mssql/dm/clickhouse... 均有），
    // 是否调用由方言能力 supportsSchema 决定，并非 postgres 专属；方法名保持方言中立，路径中的 pg 为历史遗留。
    dbSchemas: Api.newGet<string[]>('/dbs/{id}/pg/schemas'),
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
        param[field] = await AesEncrypt(param[field] as string);
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

    // 创建 Api 实例（走 uploadRaw 传输文件流，后端不返回响应体，进度经通知通道回传）
    const api = Api.newPost<void>(`/dbs/${params.dbId}/exec-sql-file`);

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

/** 数据文件导入参数：目标库表、列映射与解析/写入选项 */
export interface DataImportParams {
    dbId: number;
    dbName: string;
    table: string;
    /** 文件列→表列映射（有序，target 空=跳过） */
    columns: ImportColumn[];
    /** 文件首行是否为表头 */
    hasHeader: boolean;
    /** CSV 分隔符，默认逗号 */
    separator?: string;
    /** Excel 工作表名，默认首个 */
    sheet?: string;
    /** 空单元格是否写入 NULL */
    emptyAsNull?: boolean;
    /** 主键/唯一冲突策略：-1/空=直接插入 1=忽略 2=更新 */
    duplicateStrategy?: number;
    /** 单批插入行数 */
    batchSize?: number;
}

/**
 * 预览表格文件：解析表头与样本行，供构建列映射界面（不落库）
 */
export function previewImportFile(file: File, params: Pick<DataImportParams, 'dbId' | 'hasHeader' | 'separator' | 'sheet'>): Promise<ImportPreviewResult> {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('hasHeader', String(params.hasHeader));
    if (params.separator) formData.append('separator', params.separator);
    if (params.sheet) formData.append('sheet', params.sheet);
    return Api.newPost<ImportPreviewResult>(`/dbs/${params.dbId}/import-data-preview`).request(formData);
}

/**
 * 执行表格文件数据导入：按列映射把文件数据批量写入目标表
 *
 * 导入为同步长请求，期间由后端 Ws 回传进度：这里生成 uploadId 并创建进度通知，
 * 请求结束（成功/失败）后兼底关闭。clientId 随 URL 传递，供后端定位回传目标连接。
 */
export async function importTableData(file: File, params: DataImportParams): Promise<DataImportResult> {
    const uploadId = `data_import_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    const formData = new FormData();
    formData.append('file', file);
    formData.append('db', params.dbName);
    formData.append('table', params.table);
    formData.append('columns', JSON.stringify(params.columns));
    formData.append('hasHeader', String(params.hasHeader));
    formData.append('uploadId', uploadId);
    if (params.separator) formData.append('separator', params.separator);
    if (params.sheet) formData.append('sheet', params.sheet);
    formData.append('emptyAsNull', String(params.emptyAsNull ?? true));
    if (params.duplicateStrategy) formData.append('duplicateStrategy', String(params.duplicateStrategy));
    if (params.batchSize) formData.append('batchSize', String(params.batchSize));

    createDataImportNotification({
        uploadId,
        title: file.name,
        dbCode: '',
        dbName: params.dbName,
        table: params.table,
        imported: 0,
        total: 0,
        terminated: false,
        status: 'importing',
    });

    try {
        return await Api.newPost<DataImportResult>(`/dbs/${params.dbId}/import-data?clientId=${getClientId()}`).request(formData);
    } finally {
        finishDataImportNotification(uploadId);
    }
}
