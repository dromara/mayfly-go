/**
 * Milvus Collection 创建/编辑相关类型定义
 */

/** 索引参数 (各索引类型动态参数，已知字段 + 索引签名) */
export interface FieldIndexParams {
    M?: number;
    efConstruction?: number;
    m?: number;
    nbits?: number;
    nlist?: number;
    drop_ratio_build?: number;
    [key: string]: unknown;
}

/** Collection 字段表单结构 (已知字段 + 索引签名容纳动态字段) */
export interface CollectionField {
    name: string;
    dataType: number;
    isPrimaryKey?: boolean;
    autoID?: boolean;
    description?: string;
    isDynamic?: boolean;
    isPartitionKey?: boolean;
    isClusteringKey?: boolean;
    nullable?: boolean;
    mmap?: boolean;
    defaultValue?: string;
    dim?: number;
    maxLength?: number;
    elementType?: number;
    maxCapacity?: number;
    readonly?: boolean;
    indexType?: string;
    metricType?: string;
    indexParams: FieldIndexParams;
    [key: string]: unknown;
}

/** 索引模板项 */
export interface IndexTemplateItem {
    label: string;
    group: string;
    params: FieldIndexParams;
}

/** 字段类型索引配置 */
export interface IndexConfig {
    category: string;
    indexes: Record<string, IndexTemplateItem>;
    metrics: string[];
}

/** API 返回的字段结构 (大写字段名，动态结构) */
export interface ApiField {
    Name?: string;
    DataType?: number;
    PrimaryKey?: boolean;
    AutoID?: boolean;
    Description?: string;
    IsDynamic?: boolean;
    IsPartitionKey?: boolean;
    IsClusteringKey?: boolean;
    Nullable?: boolean;
    DefaultValue?: string;
    ElementType?: number;
    TypeParams?: { dim?: string; max_length?: string; [key: string]: unknown };
    IndexParams?: { index_type?: string; indexType?: string; metric_type?: string; metricType?: string; params?: string | Record<string, unknown>; [key: string]: unknown };
    Indexes?: { IndexType?: string; index_type?: string; MetricType?: string; metric_type?: string; Params?: Record<string, unknown>; indexParams?: Record<string, unknown>; [key: string]: unknown }[];
    [key: string]: unknown;
}

/** API 返回的 Collection 结构 (编辑/复制时的 editData) */
export interface ApiCollection {
    Name?: string;
    name?: string;
    ShardNum?: number;
    ConsistencyLevel?: number;
    description?: string;
    Schema?: { Description?: string; Fields?: ApiField[]; EnableDynamicField?: boolean; [key: string]: unknown };
    schema?: { Description?: string; Fields?: ApiField[]; EnableDynamicField?: boolean; [key: string]: unknown };
    [key: string]: unknown;
}

/** 动态字段索引项 */
export interface DynamicFieldIndex {
    indexName: string;
    indexType: string;
    json_path?: string;
    json_cast_type?: string;
    json_cast_function?: string;
    field_name?: string;
}
