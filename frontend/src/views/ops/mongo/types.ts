/**
 * Mongo 模块类型定义。
 *
 * 响应形状与后端 `internal/mongo/api/vo` + `application/dto` 逐字段对齐：
 * 后端不再直接透传 driver 结构体，字段一律小驼峰，因此这里不再出现 `Name`/`SizeOnDisk` 这类大写契约。
 */
import type { BaseModel, PageParam } from '@/types/common';
import type { DocMode } from './docview/extjson';

/** 实例列表查询入参（tagPath 由资源树按标签分组带入，keyword 走关键词搜索） */
export interface MongoListParam extends PageParam {
    tagPath?: string;
    keyword?: string;
}

/** Mongo 实例（对应 vo.Mongo） */
export interface Mongo extends BaseModel {
    id: number;
    code: string;
    name: string;
    /** 脱敏后的连接串（密码固定为 ****）。明文凭证不回传前端，改连接串时需在编辑表单重新输入 */
    uri: string;
    sshTunnelMachineId: number;
}

/** 数据库信息（对应 dto.Database） */
export interface MongoDatabase {
    name: string;
    sizeOnDisk: number;
    empty: boolean;
}

/** 集合信息（对应 dto.Collection）。type 为 collection / view，视图不可写 */
export interface MongoCollection {
    name: string;
    type: string;
    readOnly: boolean;
}

/** 查询结果里的单个文档（对应 mongodoc.QueryDoc） */
export interface MongoDoc {
    /** 服务端签发的 locate 令牌：写回时原样带回，普通 JSON 无法表达的 _id 类型由它承载 */
    idToken: string;
    /**
     * 主键类型名（objectId / string / number / date / binary / document / array ...）。
     *
     * ObjectID 主键与「24 位十六进制的字符串主键」回传后长得一模一样，不给出类型就无法区分正在改哪一条。
     */
    idKind: string;
    /** 存储内容指纹，更新时作为 baseHash 回传即可实现乐观锁 */
    hash: string;
    /** 编码模式：plain 为普通 JSON，extjson 含 $ 类型包装 */
    mode: DocMode;
    doc: Record<string, unknown>;
}

/** 集合头部读数（对应 dto.CollectionStats，服务端已从 $collStats 中收拢并拉平字段位置） */
export interface MongoCollectionStats {
    ns: string;
    count: number;
    avgObjSize: number;
    storageSize: number;
    freeStorageSize: number;
    totalIndexSize: number;
    totalSize: number;
    nindexes: number;
}

/** 文档查询结果（对应 dto.DocPage） */
export interface MongoDocPage {
    docs: MongoDoc[];
    /** 匹配总数；未开启统计时为 -1，不能拿已加载条数或集合总数冒充 */
    total: number;
    /** 结果因上限被截断 */
    truncated: boolean;
    /** 实际生效的条数上限 */
    limit: number;
    /** 集合统计，与查询同一次动作返回 */
    stats?: MongoCollectionStats;
    /** 统计不可用的原因（视图、无 collStats 权限等）：只遮住头部读数，不影响已取到的结果 */
    statsError?: string;
}

/** 写操作结果（对应 dto.WriteResult） */
export interface MongoWriteResult {
    matchedCount: number;
    modifiedCount: number;
    deletedCount: number;
    insertedCount: number;
    /** 批量更新开启 upsert 时新增的文档数 */
    upsertedCount: number;
    /** 提交内容与存储内容一致，未产生写入 */
    noChange: boolean;
    /** 插入成功时各文档的主键令牌 */
    insertedIds?: string[];
}

/** 命令目录条目（对应 mongodoc.CommandSpec） */
export interface MongoCommandSpec {
    name: string;
    /** read / dataSave / dataDel / structSave / structDel / admin */
    level: string;
    /** 该命令要求的权限码，只读命令为空串 */
    permission: string;
    needConfirm: boolean;
    template?: string;
    descKey?: string;
}

/** 索引键：字段名 + 方向（asc/desc 或 2dsphere 这类命名方向） */
export interface MongoIndexKey {
    field: string;
    direction: string;
}

/** 索引信息（对应 dto.IndexInfo） */
export interface MongoIndexInfo {
    name: string;
    keys: MongoIndexKey[];
    /** 服务返回的完整定义，复制定义与确认选项以它为准 */
    spec: Record<string, unknown>;
    sizeBytes: number;
    unique: boolean;
}

/** 集合元信息（对应 dto.CollectionMeta）：统计与索引一次拉齐 */
export interface MongoCollectionMeta {
    stats?: MongoCollectionStats;
    statsError?: string;
    indexes: MongoIndexInfo[];
    /** 集合总文档数；统计不可用时为 -1 */
    totalDocs: number;
}

/** 聚合入参 */
export interface AggParam {
    id: number;
    database: string;
    collection: string;
    pipeline: unknown[];
    allowDiskUse?: boolean;
    explain?: boolean;
}

/** 批量更新入参 */
export interface BatchUpdateParam {
    id: number;
    database: string;
    collection: string;
    filter?: unknown;
    /** 只能是 $ 操作符文档或更新管道 */
    update: unknown;
    upsert?: boolean;
    expectCount: number;
}

/** 批量删除入参 */
export interface BatchDeleteParam {
    id: number;
    database: string;
    collection: string;
    filter?: unknown;
    expectCount: number;
}

/** 导出入参（没有 limit：条数上限由服务端配置决定） */
export interface ExportParam {
    id: number;
    database: string;
    collection: string;
    filter?: unknown;
    /** 取值与后端 mongoexport.Formats 同源：json（NDJSON，类型无损）/ csv */
    format: ExportFormat;
}

/** 支持的导出格式，字面值与 `application/mongoexport` 的 Format 常量一致 */
export type ExportFormat = 'json' | 'csv';

/** 导出范围：按当前查询条件，或只导勾选中的那几条 */
export type ExportScope = 'condition' | 'selected';

/** 导出结果 */
export interface MongoExportResult {
    fileName: string;
    contentType: string;
    count: number;
    content: string;
}

/** 文档查询入参 */
export interface QueryDocsParam {
    id: number;
    database: string;
    collection: string;
    filter?: unknown;
    sort?: unknown;
    projection?: unknown;
    skip?: number;
    limit?: number;
    withCount?: boolean;
}

/** 文档插入入参 */
export interface InsertDocsParam {
    id: number;
    database: string;
    collection: string;
    /** 单个文档或文档数组 */
    docs: unknown;
}

/** 文档更新入参 */
export interface UpdateDocParam {
    id: number;
    database: string;
    collection: string;
    idToken: string;
    baseHash: string;
    doc: unknown;
}

/** 文档批量删除入参 */
export interface DeleteDocsParam {
    id: number;
    database: string;
    collection: string;
    idTokens: string[];
}

/** 管理命令入参 */
export interface RunCommandParam {
    id: number;
    database: string;
    /** 命令 JSON 原文，字段顺序与类型包装按字面提交 */
    command: unknown;
}

/** 集合级操作的路径参数 */
export interface CollectionParam {
    id: number;
    database: string;
    collection: string;
}

/** 结构销毁的确认名称参数（须与目标名完全一致） */
export interface ConfirmNameParam extends CollectionParam {
    confirmName: string;
}
