import Api from '@/common/Api';
import { AesEncrypt } from '@/common/crypto';
import { getClientId } from '@/common/utils/storage';
import type { PageParam, PageResult } from '@/types/common';
import { createSqlExecNotification, finishSqlExecNotification, registerSqlExecAborter } from '@/components/system-message/db/db-sql-exec-progress';
import { createDataImportNotification, finishDataImportNotification } from '@/components/system-message/db/db-data-import-progress';
import type {
    Db,
    DbInstance,
    DbSql,
    DbSqlExec,
    DbTableInfo,
    DbTableKey,
    DbInstanceServerInfo,
    ColumnMetadata,
    DbInstanceListParam,
    DbListParam,
    SqlExecRes,
    DbMaskRule,
    DbMaskColumn,
    DbMaskRuleQuery,
    DbTableAcl,
    DbTableAclSaveForm,
    DbColumnAcl,
    DbColumnAclSaveForm,
    SensitiveScanResult,
    DbMaskColumnQuery,
    DbMaskRuleSaveForm,
    DbMaskColumnSaveForm,
    DbSessionInfo,
    DbSlowQuery,
    DbLockWait,
    DbMetricSeries,
    DbDdlPrecheckResult,
    DbRecoveryReadiness,
    DbAccountOverview,
    SqlExplainPlan,
    SqlRollback,
    SchemaDiffResult,
    SchemaDiffParam,
    SchemaBaselineParam,
    SchemaBaselineResult,
    SchemaBaselineRow,
    SchemaDriftResult,
    DbCapabilities,
    DbMetadataObject,
    ImportColumn,
    ImportPreviewResult,
    DataImportResult,
    ConnTestResult,
} from './types';

export const dbApi = {
    // 数据库列表（导出走 GET /dbs/{id}/dump 直链下载，由 DbList 拼带凭据的地址，不经本层）
    dbs: Api.newGet<PageResult<Db>, DbListParam>('/dbs'),
    saveDb: Api.newPost<number>('/dbs'),
    deleteDb: Api.newDelete<void>('/dbs/{id}'),
    // 表清单：like/limit/offset 为可选的服务端名称过滤与分页下推（超大 schema 资源树首屏/续载按需取页）
    tableInfos: Api.newGet<DbTableInfo[], { id: number; db: string; like?: string; limit?: number; offset?: number }>('/dbs/{id}/t-infos'),
    // 扩展元数据对象节点（视图/序列/存储过程…），由后端 MetaNavigator 提供，按 features 能力位决定是否请求
    metaObjects: Api.newGet<DbMetadataObject[], { id: number; db: string; kind: string; schema?: string }>('/dbs/{id}/meta-objects'),
    // 单个扩展对象的 DDL 原文（点开节点查看）
    metaObjectDdl: Api.newGet<string, { id: number; db: string; kind: string; schema?: string; name: string }>('/dbs/{id}/meta-object-ddl'),
    // 方言能力协商：能力位（features）+ 命名空间层次的单一事实源，前端据此数据驱动渲染、不再各自硬编码
    capabilities: Api.newGet<DbCapabilities, { id: number; db: string }>('/dbs/{id}/capabilities'),
    tableIndex: Api.newGet<Record<string, unknown>[]>('/dbs/{id}/t-index'),
    // 表的主键/唯一键约束（含约束名）：编辑表撤销主键要引用约束名（pg/mssql 无 DROP PRIMARY KEY），
    // 索引列表刻意排除了主键索引，故必须单列一个接口
    tableKeys: Api.newGet<DbTableKey[]>('/dbs/{id}/t-keys'),
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
    // 活动会话列表：按库取连接后代理方言的会话内省能力（不支持的方言返回「不支持」）
    dbSessions: Api.newGet<DbSessionInfo[], { dbId: number; db: string }>('/dbs/sessions'),
    // 取消指定会话当前语句
    cancelDbSession: Api.newPost<void, { id: number; db: string; sessionId: string }>('/dbs/sessions/cancel'),
    // 慢查询聚合与锁等待链：实例级事实，按所选库做可达性校验
    dbSlowQueries: Api.newGet<DbSlowQuery[], { id: number; db: string; limit?: number }>('/dbs/{id}/slow-queries'),
    dbLockWaits: Api.newGet<DbLockWait[], { id: number; db: string }>('/dbs/{id}/lock-waits'),
    // 指标趋势：来自平台采样表（不连目标库），库不可达时历史曲线仍可看
    dbMetrics: Api.newGet<DbMetricSeries, { id: number; metric: string; minutes: number }>('/dbs/{id}/metrics'),
    // DDL 预检：只读表规模与主键，给出「会不会丢数据/要不要重建表」的提示（不做放行判定）
    // 恢复能力：PITR 前提（binlog / WAL 归档）是否具备；不具备时后端给原因
    // 账号盘点：只读，返回账号事实与高危标记（平台不代改账号）
    dbAccounts: Api.newGet<DbAccountOverview, { id: number; db: string }>('/dbs/{id}/accounts'),
    dbRecoveryReadiness: Api.newGet<DbRecoveryReadiness, { id: number; db: string }>('/dbs/{id}/recovery-readiness'),
    dbDdlPrecheck: Api.newPost<DbDdlPrecheckResult, { id: number; db: string; sql: string }>('/dbs/{id}/ddl-precheck'),
    // 取消自己发起的一次执行（服务端中断，不只是断开 HTTP）
    cancelSqlExec: Api.newPost<void, { execId: string }>('/dbs/cancel-exec'),
    // 取语句的执行计划：与预检同一条切割口径，同样只读（后端不用 ANALYZE，看计划不执行语句）
    sqlExplain: Api.newPost<SqlExplainPlan[], { id: number; db: string; sql: string }>('/dbs/sql-explain'),
    // 依据执行记录当时采到的旧值生成回滚 SQL（只生成文本，不执行）
    rollbackSql: Api.newGet<SqlRollback, { id: number }>(`/dbs/sql-execs/{id}/rollback-sql`),
    // 结构对比（库间表结构漂移检测）：只报差异，不生成也不执行 DDL
    schemaDiff: Api.newPost<SchemaDiffResult, SchemaDiffParam>('/dbs/schema-diff'),
    // 建立/刷新表结构基线（以当前结构为准覆盖）
    saveSchemaBaseline: Api.newPost<SchemaBaselineResult, SchemaBaselineParam>('/dbs/schema-baseline'),
    // 查看某个库已有的基线行
    listSchemaBaseline: Api.newGet<SchemaBaselineRow[], { dbId: number; db: string }>('/dbs/schema-baseline'),
    // 立即比对一次当前结构与基线
    checkSchemaBaseline: Api.newPost<SchemaDriftResult, SchemaBaselineParam>('/dbs/schema-baseline/check'),
    // 清除某个库的基线
    deleteSchemaBaseline: Api.newDelete<void, { dbId: number; db: string }>('/dbs/schema-baseline'),
    // 获取数据库兼容版本
    getCompatibleDbVersion: Api.newGet<string>('/dbs/{id}/version'),
    // 采样某列出现过的去重值（编辑器值补全）；方言无分页改写能力时后端报错，调用侧静默降级为无候选
    columnValues: Api.newGet<string[], { id: number; db: string; tableName: string; column: string }>('/dbs/{id}/column-values'),
    // 敏感列扫描：按列名识别候选，并标出是否已被列保护覆盖
    scanSensitiveColumns: Api.newGet<SensitiveScanResult, { id: number; db: string }>('/dbs/{id}/sensitive-columns'),

    instances: Api.newGet<PageResult<DbInstance>, DbInstanceListParam>('/instances'),
    getInstance: Api.newGet<DbInstance>('/instances/{instanceId}'),
    getAllDatabase: Api.newPost<string[]>('/instances/databases'),
    getDbNamesByAc: Api.newGet<string[]>('/instances/databases/{authCertName}'),
    getInstanceServerInfo: Api.newGet<DbInstanceServerInfo>('/instances/{instanceId}/server-info'),
    // 测试连接：返回本连接实际协商出的加密状态（available=false 表示读不到，不是未加密）
    testConn: Api.newPost<ConnTestResult>('/instances/test-conn'),
    saveInstance: Api.newPost<number>('/instances'),
    deleteInstance: Api.newDelete<void>('/instances/{id}'),
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
export const dbTableAclApi = {
    // 某库的表访问控制规则
    list: Api.newGet<DbTableAcl[], { dbId: number }>('/dbs/table-acls'),
    // 新增规则（同一库同一张表由服务端按表名去重为更新）
    save: Api.newPost<void, DbTableAclSaveForm>('/dbs/table-acls'),
    // 更新规则
    update: Api.newPut<void, DbTableAclSaveForm>('/dbs/table-acls'),
    // 删除规则
    remove: Api.newDelete<void, { id: number }>('/dbs/table-acls/{id}'),
};

export const dbColumnAclApi = {
    // 某库的列级访问控制规则
    list: Api.newGet<DbColumnAcl[], { dbId: number }>('/dbs/column-acls'),
    save: Api.newPost<void, DbColumnAclSaveForm>('/dbs/column-acls'),
    update: Api.newPut<void, DbColumnAclSaveForm>('/dbs/column-acls'),
    remove: Api.newDelete<void, { id: number }>('/dbs/column-acls/{id}'),
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
    //
    // 请求收尾（成功/失败）后统一兜底关闭进度通知：执行终态本应由 Ws 消息回传，
    // 一旦那条消息丢失，通知与状态表就会永久悬挂（服务端执行完才响应，故此处关闭不会掩盖仍在跑的进度）
    const { abort } = api.uploadRaw(file, queryParams, {
        onSuccess: () => {
            finishSqlExecNotification(uploadId);
            options.onSuccess?.();
        },
        onError: (error) => {
            finishSqlExecNotification(uploadId);
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

/**
 * 生成交互式执行 id
 *
 * 随每次 exec-sql 请求上送：服务端据此登记进行中的执行，
 * 「取消」才能既断 HTTP 又真正中断数据库侧的语句（见 dbApi.cancelSqlExec）
 */
export function newSqlExecId(): string {
    return `sql_exec_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
}
