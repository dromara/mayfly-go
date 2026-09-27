/**
 * DB 模块实体类型定义
 * 对应后端: db/domain/entity/*
 */
import type { MachineAuthCert } from '@/views/ops/machine/types';
import type { BaseModel, PageParam } from '@/types/common';

/** 数据库实例VO (用于 DbInst.getOrNewInst 参数) */
export interface DbInstInfo {
    id: number;
    tagPath?: string;
    host?: string;
    name?: string;
    type?: string;
    databases?: string[];
    database?: string;
    getDatabaseMode?: number;
    authCertName?: string;
}

/** 数据库实体 (对应 entity.Db) */
export interface Db extends BaseModel {
    id: number;
    code: string;
    name: string;
    getDatabaseMode?: number;
    database?: string;
    remark?: string;
    instanceId?: number;
    authCertName?: string;
    /** 以下字段仅列表 API 返回（后端 vo.DbListVO 关联实例信息填充） */
    instanceCode?: string;
    type?: string;
    host?: string;
    port?: number;
}

/**
 * Db 编辑表单 (id/name/instanceId 允许 null 表示未选择)
 *
 * 它是 `DbEdit` 组件 confirm 事件的载荷类型，父级（DbList）收到后补 instanceId 直接提交给
 * saveDb 的 upsert 接口，故属于跨组件的边界契约，定义在此而非组件内部。
 */
export type DbForm = Partial<Omit<Db, 'id' | 'name' | 'instanceId'>> & {
    id?: number | null;
    name?: string | null;
    instanceId?: number | null;
};

/** 获取库名所需参数 (Db/DbInstance 或动态记录中的相关字段) */
export type DbNamesParam = {
    getDatabaseMode?: number;
    database?: string;
    authCertName?: string;
};

/** 数据库实例实体 (对应 entity.DbInstance) */
export interface DbInstance extends BaseModel {
    id: number;
    code: string;
    name: string;
    type: string;
    host: string;
    port: number;
    network: string;
    params?: string;
    remark?: string;
    sshTunnelMachineId: number;
    authCerts?: MachineAuthCert[];
    selectAuthCert?: MachineAuthCert;
    extra?: Record<string, unknown>;
}

/** 数据库SQL实体 (对应 entity.DbSql) */
export interface DbSql extends BaseModel {
    id: number;
    dbId: number;
    db: string;
    type: number;
    sql: string;
    name: string;
}

/** 数据库SQL执行记录实体 (对应 entity.DbSqlExec) */
export interface DbSqlExec extends BaseModel {
    id: number;
    dbId: number;
    db: string;
    table: string;
    type: number;
    sql: string;
    oldValue: string;
    remark: string;
    status: number;
    res: string;
    flowBizKey: string;
}

/** 数据库实例列表查询参数 */
export interface DbInstanceListParam extends PageParam {
    id?: number;
    tagPath?: string;
}

/** 数据库列表查询参数 */
export interface DbListParam extends PageParam {
    id?: number;
    tagPath?: string;
    instanceId?: number;
}

/** 数据同步任务实体 (对应 entity.DataSyncTask) */
export interface DataSyncTask extends BaseModel {
    id: number;
    taskName: string;
    taskCron: string;
    status: number;
    taskKey: string;
    recentState: number;
    runningState: number;
    syncMode: number;
    srcDbId: number;
    srcDbName: string;
    srcTagPath: string;
    dataSql: string;
    pageSize: number;
    updField: string;
    updFieldVal: string;
    updFieldSrc: string;
    updFieldSecondary: string;
    softDeleteField: string;
    softDeleteValue: string;
    transformRules: string;
    filterCondition: string;
    nullStrategy: number;
    nullDefault: string;
    targetDbId: number;
    targetDbName: string;
    targetTagPath: string;
    targetTableName: string;
    fieldMap: string;
    duplicateStrategy: number;
    schemaEvolveMode: number;
    biDirEnabled: boolean;
    reverseTaskId: number;
    conflictStrategy: number;
    biDirTimestampField: string;
}

/**
 * 数据同步任务列表行 (对应 vo.DataSyncTaskListVO)。
 *
 * 与详情实体的关键差异：后端 VO 将 TaskCron 序列化为 `cron`（键名不同），
 * 且只带列表页所需字段；dataSql/fieldMap 等同步配置不在列表返回，由详情接口回填。
 */
export interface DataSyncTaskListVO {
    id: number;
    taskName: string;
    /** cron 表达式；详情实体里键名为 taskCron */
    cron: string;
    syncMode: number;
    status: number;
    recentState: number;
    runningState: number;
    srcDbId: number;
    srcDbName: string;
    srcTagPath: string;
    /** 库类型由后端列表按库id回查实例带出（同步任务表未冗余存该字段），库被删除时为空 */
    srcDbType: string;
    targetDbId: number;
    targetDbName: string;
    targetTagPath: string;
    targetTableName: string;
    targetDbType: string;
    /** 增量字段与其当前水位（仅增量追加/合并模式参与拼增量条件） */
    updField: string;
    updFieldVal: string;
    pageSize: number;
    duplicateStrategy: number;
    biDirEnabled: boolean;
    reverseTaskId: number;
    createTime: string;
    creator: string;
    updateTime: string;
    modifier: string;
    modifierId?: number;
}

/** 数据同步日志列表行 (对应 vo.DataSyncLogListVO)；VO 不含 runLog/dataSqlFull，运行日志按日志 id 单独获取 */
export interface DataSyncLogListVO {
    id: number;
    taskId: number;
    status: number;
    createTime: string;
    durationMs: number;
    throughput: number;
    batchCount: number;
    insertCount: number;
    updateCount: number;
    deleteCount: number;
    skipCount: number;
    resNum: number;
    schemaChanges: number;
    errText: string;
}

/** 单条数据同步执行日志的运行日志内容 (对应 vo.DataSyncLogRunVO) */
export interface DataSyncLogRunInfo {
    id: number;
    status: number;
    runLog: string;
}

/** 数据同步日志 (对应 entity.DataSyncLog) */
export interface DataSyncLog {
    id: number;
    createTime: string;
    taskId: number;
    dataSqlFull: string;
    resNum: number;
    errText: string;
    status: number;
    durationMs: number;
    throughput: number;
    batchCount: number;
    insertCount: number;
    updateCount: number;
    deleteCount: number;
    skipCount: number;
    schemaChanges: number;
    runLog: string;
}

/** 数据库迁移任务实体 (对应 entity.DbTransferTask) */
export interface DbTransferTask extends BaseModel {
    id: number;
    taskName: string;
    taskKey: string;
    cronEnabled: number;
    cron: string;
    mode: number;
    targetFileDbType: string;
    fileSaveDays: number;
    status: number;
    runningState: number;
    logId: number;
    checkedKeys: string;
    deleteTable: number;
    nameCase: number;
    strategy: number;
    /** 并发度（迁移到数据库时的分片并行度，0表示用后端默认值4） */
    concurrency: number;
    srcDbId: number;
    srcDbName: string;
    srcTagPath: string;
    srcDbType: string;
    srcInstName: string;
    targetDbId: number;
    targetDbName: string;
    targetDbType: string;
    targetInstName: string;
    targetTagPath: string;
    extra?: Record<string, unknown>;
}

/**
 * 迁移任务列表行 (对应 vo.DbTransferTaskListVO)：详情实体去掉后端列表不返回的字段。
 * taskKey 为后端定时任务的内部标识，不对外展示。
 */
export type DbTransferTaskListVO = Omit<DbTransferTask, 'taskKey'>;

/** 迁移导出文件 (对应 entity.DbTransferFile) */
export interface DbTransferFile {
    id: number;
    /** 所属迁移任务 ID */
    taskId: number;
    /** 文件存储键 */
    fileKey: string;
    /** sql 文件的数据库类型 */
    fileDbType: string;
    /** 状态 1、执行中 2、执行成功 -1、执行失败 */
    status: number;
    createTime: string;
}

/** 数据库迁移执行日志 (对应 entity.DbTransferLog) */
export interface DbTransferLog {
    id: number;
    createTime: string;
    taskId: number;
    mode: number;
    purpose: number;
    targetFile: string;
    errText: string;
    status: number;
    durationMs: number;
    totalRows: number;
    tableCount: number;
    runLog: string;
}

/** 数据库迁移执行日志列表行 (对应 vo.DbTransferLogListVO)；VO 不含 runLog，运行日志按日志 id 单独获取 */
export interface DbTransferLogListVO {
    id: number;
    createTime: string;
    taskId: number;
    mode: number;
    purpose: number;
    targetFile: string;
    errText: string;
    status: number;
    durationMs: number;
    totalRows: number;
    tableCount: number;
}

/** 单条迁移执行日志的运行日志内容 (对应 vo.DbTransferLogRunVO) */
export interface DbTransferLogRunInfo {
    id: number;
    status: number;
    runLog: string;
}

/** 数据库备份信息 */
export interface DbBackup {
    id: number;
    dbId: number;
    dbName: string;
    name: string;
    cron: string;
    status: number;
    remark: string;
    maxSaveDays: number;
    createTime: string;
}

/** 数据库备份历史 */
export interface DbBackupHistory {
    id: number;
    backupId: number;
    dbName: string;
    name: string;
    status: number;
    remark: string;
    createTime: string;
}

/** 数据库恢复信息 */
export interface DbRestore {
    id: number;
    dbId: number;
    dbName: string;
    name: string;
    status: number;
    remark: string;
    createTime: string;
}

/** 数据库实例服务器信息 */
export interface DbInstanceServerInfo {
    version: string;
    majorVersion: number;
    minorVersion: number;
    /** 其余运行时信息（对应后端 dbi.DbServer.Extra，为嵌套对象而非展平） */
    extra?: Record<string, unknown>;
    [key: string]: unknown;
}

/** 脱敏规则 (对应 entity.DbMaskRule) */
export interface DbMaskRule extends BaseModel {
    id: number;
    name: string;
    /** 匹配类型: 1正则 2精确 3前缀 */
    matchType: number;
    pattern: string;
    algorithm: string;
    /** 算法参数json */
    params?: string;
    /** 状态: 1启用 0停用 */
    status: number;
    weight?: number;
    remark?: string;
}

/** 脱敏列标签 (对应 entity.DbMaskColumn) */
export interface DbMaskColumn extends BaseModel {
    id: number;
    instanceId: number;
    dbName?: string;
    tableName?: string;
    columnName?: string;
    /** 动作: 1绑定规则 2豁免 */
    action: number;
    ruleId?: number;
    /** 直接指定的算法，优先于规则 */
    algorithm?: string;
    params?: string;
    remark?: string;
}

/** 脱敏规则查询条件 */
export interface DbMaskRuleQuery extends PageParam {
    name?: string;
    keyword?: string;
    status?: number;
}

/** 脱敏规则保存表单 */
export type DbMaskRuleSaveForm = Partial<Omit<DbMaskRule, keyof BaseModel>>;

/** 脱敏列标签查询条件 */
export interface DbMaskColumnQuery extends PageParam {
    instanceId?: number;
    dbName?: string;
    tableName?: string;
    columnName?: string;
}

/** 脱敏列标签保存表单 */
export type DbMaskColumnSaveForm = Partial<Omit<DbMaskColumn, keyof BaseModel>>;

// 方言能力协商响应（GET /dbs/{id}/capabilities）：能力单一事实源来自后端，前端据此数据驱动渲染资源树
export interface DbNamespace {
    HasDatabase: boolean;
    HasSchema: boolean;
    HasCatalog: boolean;
}

export interface DbCapabilities {
    dbType: string;
    // 支持的能力名清单（如 "schemas"/"view"/"sequence"/"relation"…），由后端 SupportedFeatures() 稳定升序输出
    features: string[];
    namespace: DbNamespace;
}

// 扩展元数据对象节点（GET /dbs/{id}/meta-objects?kind=view|sequence|…）
export interface DbMetadataObject {
    name: string;
    kind: string;
    schema?: string;
    comment?: string;
    attrs?: Record<string, unknown>;
}
