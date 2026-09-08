/**
 * DB 模块类型定义
 * 对应后端: db/domain/entity/* + db/api/form/* + db/api/vo/*
 */
import type { MachineAuthCert } from '@/views/ops/machine/types';
import type { BaseModel, PageParam } from '@/types/common';

// ==================== Entity ====================

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
    srcDbId: number;
    srcDbName: string;
    srcTagPath: string;
    dataSql: string;
    pageSize: number;
    updField: string;
    updFieldVal: string;
    updFieldSrc: string;
    targetDbId: number;
    targetDbName: string;
    targetTagPath: string;
    targetTableName: string;
    fieldMap: string;
    duplicateStrategy: number;
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
}

/** 数据库迁移任务实体 (对应 entity.DbTransferTask) */
export interface DbTransferTask extends BaseModel {
    id: number;
    taskName: string;
    taskKey: string;
    cronAble: number;
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

// ==================== VO ====================

/** 数据库表信息 */
export interface DbTableInfo {
    tableName: string;
    tableComment: string;
    dataLength: number;
    tableRows: number;
    indexLength: number;
}

/** 数据库表列信息 (对应 columnMetadata API) */
export interface ColumnMetadata {
    columnName: string;
    dataType: string;
    columnComment?: string;
    isPrimaryKey?: boolean;
    nullable?: boolean;
    charMaxLength?: number;
    numPrecision?: number;
    numScale?: number;
    /** 列默认值 */
    columnDefault?: string;
    /** 是否自增 */
    autoIncrement?: boolean;
    /** 动态设置: 显示列类型(含长度) */
    columnType?: string;
    /** 动态设置: 是否显示长度 */
    showLength?: number | null;
    /** 动态设置: 是否显示小数位数 */
    showScale?: number | null;
}

/** 表格列定义 (DbTableData/DbTableDataForm columns prop 输入；dataType 由组件内部依据 columnType 计算) */
export interface TableColumnDef {
    columnName: string;
    dataType?: string;
    columnComment?: string;
    columnType?: string;
    key?: string;
    show?: boolean;
    /** 是否为脱敏列 (后端 doQuery 脱敏后下发) */
    masked?: boolean;
    nullable?: boolean;
    isPrimaryKey?: boolean;
    autoIncrement?: boolean;
    /** 数据类型角标 (组件内部计算) */
    dataTypeSubscript?: string;
    /** 列备注 (组件内部计算) */
    remark?: string;
}

/** 数据库表列信息 (对应 DbTableColumn) */
export interface DbTableColumn {
    name: string;
    type: string;
    comment: string;
    nullable: boolean;
    isPrimaryKey: boolean;
    defaultValue: string;
    extra: string;
}

/** 数据库表索引信息 */
export interface DbTableIndex {
    indexName: string;
    columnName: string;
    indexType: string;
    isUnique: boolean;
}

/** 表创建/编辑弹窗数据 (DbTableOp 的 data prop) */
export interface TableOpData {
    /** 是否编辑现有表 */
    edit: boolean;
    /** 表基本信息 (tableName/tableComment) */
    row: Record<string, unknown>;
    /** 表索引信息 */
    indexs?: Record<string, unknown>[];
    /** 表列元数据 */
    columns?: ColumnMetadata[];
}

/** SQL执行结果列信息 (exec-sql 返回的 columns 元素) */
export interface SqlExecResColumn {
    name: string;
    key?: string;
    type?: string;
    /** 是否为脱敏列 (服务端 doQuery 脱敏后下发) */
    masked?: boolean;
    [key: string]: unknown;
}

/** SQL执行结果 (后端 exec-sql 接口返回数组的元素) */
export interface SqlExecRes {
    /** 执行的sql */
    sql?: string;
    /** 错误信息，存在则表示该sql执行失败 */
    errorMsg?: string;
    /** 结果集列信息 */
    columns?: SqlExecResColumn[];
    /** 结果集数据行 */
    res?: Record<string, unknown>[];
    affectedRows?: number;
    execTime?: string;
    [key: string]: unknown;
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
    uptime: string;
    connections: number;
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
