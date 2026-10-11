/**
 * DB 模块 VO 类型定义
 * 表信息、列元数据、表格列定义等视图对象
 */
import type { ColumnDefinition, IndexDefinition } from '../dialect/types';
import type { DbInstInfo } from './entity';

// ==================== 表信息 ====================

/** 数据库表信息 */
export interface DbTableInfo {
    tableName: string;
    tableComment: string;
    dataLength: number;
    tableRows: number;
    indexLength: number;
}

/**
 * 执行计划的一个节点（对应后端 dbi.PlanNode）
 *
 * fullScan/children 都是方言给出的事实，前端不按 nodeKind 文本重算判定；
 * children 全空不等于「都是叶子」，而是该方言没给出父子关系（只能平铺）
 */
export interface SqlPlanNode {
    nodeKind: string;
    table?: string;
    accessMethod?: string;
    indexName?: string;
    estRows: number;
    fullScan?: boolean;
    children?: number[];
}

/** 一次计划分析的产物（对应后端 dbi.ExplainResult） */
export interface SqlExplainResult {
    /** 优化器估算行数，不是真实影响行数 */
    rows: number;
    /** 估算口径：explain = 未执行的优化器估算，展示时必须标出 */
    estimateKind: string;
    fullScan: boolean;
    usedIndexes?: string[];
    plan?: SqlPlanNode[];
    warnings?: string[];
}

/**
 * 单条语句的执行计划取数结果（对应后端 application.ExplainPlan）
 *
 * result 与 note 恰好一个非空：拿不到计划时必须有 note，不能把「没有计划」渲染成「这条语句没有扫描」
 */
export interface SqlExplainPlan {
    sql: string;
    stmtType: string;
    result?: SqlExplainResult;
    note?: string;
}

/**
 * 表结构基线入参（建立/刷新与比对共用，对应后端 form.DbSchemaBaselineForm）
 *
 * tables 为空表示该库全部表（服务端限额内），与结构对比同一口径
 */
export interface SchemaBaselineParam {
    id: number;
    db: string;
    tables?: string[];
}

/** 建立基线的结果 */
export interface SchemaBaselineResult {
    /** 纳入基线的表数 */
    tables: number;
    /** 覆盖范围受限之类的说明，没有它「无漂移」会被读成「整个库都没变」 */
    note?: string;
}

/** 一张漂移的表 */
export interface SchemaDriftTable {
    table: string;
    detail: string;
}

/** 一次漂移比对的结果 */
export interface SchemaDriftResult {
    dbId: number;
    db: string;
    checkedAt?: string;
    driftedTables?: SchemaDriftTable[];
    note?: string;
}

/** 基线行（一张表一行，对应后端 entity.DbSchemaBaseline） */
export interface SchemaBaselineRow {
    table: string;
    fingerprint: string;
    columnCount: number;
    primaryKeys: string;
    indexCount: number;
    drifted: boolean;
    driftDetail: string;
    checkedAt?: string;
}

/**
 * 回滚 SQL 生成结果（对应后端 dto.SqlRollback）
 *
 * 服务端只生成文本不执行：回滚本身是一次新变更，要走它自己的预检与审批。
 * rows 是本次生成的语句条数，不等于原执行的影响行数（行数据采集有上限，截断时 note 非空）
 */
export interface SqlRollback {
    execId: number;
    type: number;
    sqls: string[];
    rows: number;
    note?: string;
    /** 生成前是否与执行后的行快照做过并发比对（仅 UPDATE 有意义）：false 表示降级生成，风险已写进 note */
    verified: boolean;
    /** 未生成回滚的行：变更之后又被修改、删除或无法定位，需人工对账 */
    conflicts?: SqlRollbackConflict[];
}

/**
 * 一行未生成回滚的原因（对应后端 dto.SqlRollbackConflict）
 *
 * reason 是枚举（concurrent-modified / row-missing / unknown-position / ambiguous-match）：
 * 前端按枚举映射文案；后端新增枚举而前端未跟上时原样展示枚举值，不吞掉信息
 */
export interface SqlRollbackConflict {
    /** 行标识：UPDATE 是主键值（如 "id=2"），INSERT 是插入行序号（如 "row #2"） */
    key: string;
    reason: string;
}

/**
 * 结构对比（库间表结构漂移检测）结果，对应后端 schema.Result
 *
 * 只报差异不生成 DDL：真要改结构要走它自己的预检与审批。
 * tables 只包含有差异或「比不了」的表；unavailable 是比不了的方面（内省失败、类型认不出）——
 * 它既不是差异也不是一致，不当成一致看是这块功能不误导人的关键
 */
export interface SchemaDiffResult {
    source: string;
    target: string;
    compared: number;
    diffTables: number;
    tables?: SchemaTableDiff[];
    note?: string;
}

export interface SchemaTableDiff {
    table: string;
    kind: string;
    columns?: SchemaColumnDiff[];
    indexes?: SchemaIndexDiff[];
    primaryKey?: { kind: string; source?: string; target?: string };
    unavailable?: { aspect: string; reason: string }[];
}

export interface SchemaColumnDiff {
    column: string;
    kind: string;
    source?: string;
    target?: string;
}

export interface SchemaIndexDiff {
    kind: string;
    columns: string;
    source?: string;
    target?: string;
}

/** 结构对比入参（对应后端 form.DbSchemaDiffForm） */
export interface SchemaDiffParam {
    srcDbId: number;
    srcDb: string;
    dstDbId: number;
    dstDb: string;
    tables?: string[];
}

/**
 * 活动会话信息（对应后端 dbi.SessionInfo）
 *
 * 字段全为库侧事实：startedAt 可能为空（部分方言只给出已执行时长，不虚构时间点），
 * lockWait 由方言按自身系统表判据给出，前端不再按 state 文本猜是否等锁
 */
export interface DbSessionInfo {
    /** 会话标识，取消时回传 */
    id: string;
    db?: string;
    user?: string;
    /** 当前语句已执行毫秒数 */
    elapsedMs: number;
    state?: string;
    query?: string;
    lockWait: boolean;
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

// ==================== 数据文件导入 ====================

/** 列映射：一个文件列 → 一个数据库列（target 为空表示跳过该文件列） */
export interface ImportColumn {
    /** 文件列的位置下标字符串（如 "0"）；后端按列位置映射，预览与导入对同一文件解析列序稳定 */
    source: string;
    /** 目标数据库列名；空=跳过 */
    target: string;
}

/** 文件预览结果 (对应 import-data-preview API) */
export interface ImportPreviewResult {
    headers: string[];
    sampleRows: string[][];
    totalRows: number;
    /** Excel 内的全部工作表名（CSV 等为 undefined），供工作表下拉选择 */
    sheets?: string[];
}

/** 数据导入结果 (对应 import-data API) */
export interface DataImportResult {
    totalRows: number;
    imported: number;
    batchCount: number;
    affectedRow: number;
    /** 触发策略提醒：不阻断导入，但必须让操作者看见（内容由服务端按请求语言下发） */
    warnings?: string[];
}

// ==================== 表格列定义 ====================

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

/**
 * el-table-v2 渲染列：setTableColumns 动态生成的列 + 行号列，
 * 在 TableColumnDef 基础上补充渲染所需字段（列元数据字段均为可选，行号列无 columnName 等）
 */
export interface RenderTableColumn extends Partial<TableColumnDef> {
    key: string;
    title: string;
    width?: number;
    fixed?: boolean;
    align?: 'left' | 'center' | 'right';
    headerClass?: string;
    class?: string;
    sortable?: boolean;
    hidden?: boolean;
}

// ==================== 表结构信息 ====================

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

/** 数据库表索引信息 (对应 tableIndex API，多列索引的 columnName 为逗号分隔串) */
export interface DbTableIndex {
    indexName: string;
    columnName: string;
    indexType: string;
    isUnique: boolean;
    /** 索引注释（表编辑回显与 SchemaDiff 需要） */
    indexComment?: string;
    /**
     * 后端 t-index 下发的是动态扁平结构（API 声明即 Record<string, unknown>[]），
     * 不同方言可能附带额外字段。保留索引签名，使其能直接从 API 返回值收窄，
     * 无需在各消费点重复声明局部结构或走 `as unknown as` 双重断言。
     */
    [key: string]: unknown;
}

/** 表创建/编辑弹窗数据 (DbTableOp 的 data prop) */
export interface TableOpData {
    /** 是否编辑现有表 */
    edit: boolean;
    /** 表基本信息 (tableName/tableComment) */
    row: Record<string, unknown>;
    /** 表索引信息 */
    indexes?: Record<string, unknown>[];
    /** 表列元数据 */
    columns?: ColumnMetadata[];
    /** 预处理的表单数据（编辑模式由 onEditTable 提前构建，避免 watch 阻塞抽屉动画） */
    formData?: {
        tableName: string;
        tableComment: string;
        oldTableName: string;
        oldTableComment: string;
        db?: string;
        fields: { res: ColumnDefinition[]; oldFields: ColumnDefinition[] };
        indexes: { res: IndexDefinition[]; oldIndexes: IndexDefinition[]; columns: { name: string; comment: string }[] };
    };
}

// ==================== 资源树节点参数 ====================

/**
 * db 资源树「库」粒度节点的 params，即 `DbSelectTree` 的 `selectDb` 事件载荷。
 *
 * 生产方是资源树（节点数据由后端列表拼装，结构随 kind 变化），消费方是迁移/同步等需要
 * 「选中某个库」的表单。此前该结构在生产方与两个消费方里各写一份、宽严不一，
 * 父子两端类型已对不上，故收敛为唯一定义。
 *
 * `id`/`db` 在库粒度节点上必然存在（changeNode 依赖它们回填），其余字段按节点类型缺省。
 * 带索引签名是因为消费方会以 `(params: Record<string, unknown>)` 形态接收该事件。
 */
export interface DbNodeParams {
    /** 实例 id */
    id: number;
    /** 库名 */
    db: string;
    /** 实例下的库名列表（SQL 补全需要，可能尚未加载） */
    dbs?: string[];
    /** 数据库类型（mysql / oracle / ...） */
    type?: string;
    /** 实例名 */
    name?: string;
    /** 标签路径 */
    tagPath?: string;
    /** 供 DbInst.getOrNewInst 使用，由 dbs 赋入 */
    databases?: string[];
    /** 其他动态字段（不同 kind 节点结构不同） */
    [key: string]: unknown;
}

/**
 * db 资源树「扩展元数据对象」叶子节点（视图/序列/存储过程…）的 params。
 *
 * 生产方是 contributors.ts 的 DbObjectMenuKind（展开后端 MetadataObject 时叠加对象字段），
 * 消费方是 DbDataOp 的对象查看回调（查看 DDL / 属性面板 / 浏览数据）。
 * 继承库粒度 DbNodeParams，仅补充对象级字段。
 */
export interface DbObjectNodeParams extends DbNodeParams {
    /** 数据库类型（对象节点必带，用于取方言） */
    type: string;
    /** 对象类别（view/sequence/...），与后端 dbi.ObjectKind 对齐 */
    objKind: string;
    /** 对象名 */
    objName: string;
    /** 所属 schema（pg 等分层库有；mysql 类可缺省） */
    schema?: string;
    /** 对象定义属性（如序列 dataType/startValue/...），随 meta-objects 列表返回 */
    objAttrs?: Record<string, unknown>;
}

/**
 * db 资源树「表」粒度节点的 params。
 *
 * 生产方是 resource/contributors.ts 的表节点（由库节点 params 展开后叠加表字段），
 * 消费方是 DbDataOp 的表级操作回调（编辑/删除/重命名/复制/生成 DDL）。
 * 继承库粒度的 DbNodeParams，仅补充表级字段。
 */
export interface DbTableNodeParams extends DbNodeParams {
    /** 数据库类型（表节点必带，用于取方言） */
    type: string;
    /** 实例版本，用于命中版本特化方言（如 oracle11） */
    version?: string;
    /** 表名：DbTableKind 叶子节点必有；DbTableMenuKind（表菜单节点）无此字段 */
    tableName?: string;
    /** 表注释 */
    tableComment?: string;
    /** 父节点 key，操作完成后据此局部刷新树 */
    parentKey?: string;
    /** 节点 key */
    key?: string;
    /** schema 名（supportsSchema 的方言才有） */
    schema?: string;
}

/**
 * 数据库树节点数据：DbDataOp 的 dbInfo prop 形态。
 * 在实例信息之上叠加资源树注入的节点属性。
 */
export interface DbTreeNodeData extends DbInstInfo {
    /** 资源树节点 key */
    nodeKey?: string;
    /** 实例 code（资源树节点 key 首段，用于重建 sql-menu 等节点 key） */
    instCode?: string;
    /** 库 code（资源树节点 key 中段，同上） */
    dbCode?: string;
    /** 实例下的库名列表 */
    dbs?: string[];
    /** 当前库名 */
    db?: string;
}

/**
 * 树节点回调数据：右键菜单/双击等动作传给 DbDataOp 表级操作的载荷。
 * params 为节点自身数据，其余字段由树组件补齐。
 */
export interface TreeNodeCallbackData {
    /** 节点 params */
    params: DbTableNodeParams;
    /** 节点显示文本 */
    label?: string;
    /** 父节点 key */
    parentKey?: string;
    /** 节点 key */
    key?: string;
    /** 实例版本 */
    version?: string;
}
