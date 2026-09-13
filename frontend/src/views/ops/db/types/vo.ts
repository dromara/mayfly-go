/**
 * DB 模块 VO 类型定义
 * 表信息、列元数据、表格列定义等视图对象
 */
import type { RowDefinition, IndexDefinition } from '../dialect/types';
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
    indexs?: Record<string, unknown>[];
    /** 表列元数据 */
    columns?: ColumnMetadata[];
    /** 预处理的表单数据（编辑模式由 onEditTable 提前构建，避免 watch 阻塞抽屉动画） */
    formData?: {
        tableName: string;
        tableComment: string;
        oldTableName: string;
        oldTableComment: string;
        db?: string;
        fields: { res: RowDefinition[]; oldFields: RowDefinition[] };
        indexs: { res: IndexDefinition[]; oldIndexs: IndexDefinition[]; columns: { name: string; remark: string }[] };
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
