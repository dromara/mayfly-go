/**
 * Milvus Collection 字段类型与索引配置常量
 */
import type { IndexConfig, IndexTemplateItem } from './types';

// 字段类型分组定义
export const fieldTypeGroups: Record<string, { label: string; value: number }[]> = {
    向量: [
        { label: 'FloatVector', value: 101 },
        { label: 'BinaryVector', value: 100 },
        { label: 'Float16Vector', value: 102 },
        { label: 'BFloat16Vector', value: 103 },
        { label: 'SparseFloatVector', value: 104 },
        { label: 'Int8Vector', value: 105 },
    ],
    '数字 & 布尔': [
        { label: 'Int8', value: 2 },
        { label: 'Int16', value: 3 },
        { label: 'Int32', value: 4 },
        { label: 'Int64', value: 5 },
        { label: 'Float', value: 10 },
        { label: 'Double', value: 11 },
        { label: 'Bool', value: 1 },
    ],
    'VarChar & JSON': [
        { label: 'VarChar', value: 21 },
        { label: 'JSON', value: 23 },
    ],
    数组: [{ label: 'Array', value: 22 }],
    其他: [
        { label: 'Geometry', value: 24 },
        { label: 'Timestamptz', value: 26 },
    ],
};

// 数据类型常量映射
export const DataTypeMap = {
    Bool: 1,
    Int8: 2,
    Int16: 3,
    Int32: 4,
    Int64: 5,
    Float: 10,
    Double: 11,
    VarChar: 21,
    Array: 22,
    JSON: 23,
    Geometry: 24,
    Timestamptz: 26,
    BinaryVector: 100,
    FloatVector: 101,
    Float16Vector: 102,
    BFloat16Vector: 103,
    SparseFloatVector: 104,
    Int8Vector: 105,
} as const;

// 索引参数模板
export const IndexTemplates = {
    // 通用向量索引参数
    vectorCommon: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        FLAT: { label: 'FLAT', group: '内存索引', params: {} },
        IVF_FLAT: { label: 'IVF_FLAT', group: '内存索引', params: { nlist: 1024 } },
        IVF_SQ8: { label: 'IVF_SQ8', group: '内存索引', params: { nlist: 1024 } },
        IVF_PQ: { label: 'IVF_PQ', group: '内存索引', params: { nlist: 1024, m: 0, nbits: 8 } },
        IVF_RABITQ: { label: 'IVF_RABITQ', group: '内存索引', params: { nlist: 1024 } },
        HNSW: { label: 'HNSW', group: '内存索引', params: { M: 16, efConstruction: 200 } },
        HNSW_SQ: { label: 'HNSW_SQ', group: '内存索引', params: { M: 16, efConstruction: 200 } },
        HNSW_PQ: { label: 'HNSW_PQ', group: '内存索引', params: { M: 16, efConstruction: 200, m: 0, nbits: 8 } },
        HNSW_PRQ: { label: 'HNSW_PRQ', group: '内存索引', params: { M: 16, efConstruction: 200, m: 0, nbits: 8 } },
        SCANN: { label: 'SCANN', group: '内存索引', params: { nlist: 1024 } },
        DISKANN: { label: 'DISKANN', group: '磁盘索引', params: {} },
        AISAQ: { label: 'AISAQ', group: '磁盘索引', params: {} },
        GPU_CAGRA: { label: 'GPU_CAGRA', group: 'GPU索引', params: {} },
        GPU_IVF_FLAT: { label: 'GPU_IVF_FLAT', group: 'GPU索引', params: { nlist: 1024 } },
        GPU_IVF_PQ: { label: 'GPU_IVF_PQ', group: 'GPU索引', params: { nlist: 1024, m: 0, nbits: 8 } },
    },
    // BinaryVector 索引
    binaryVector: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        BIN_FLAT: { label: 'BIN_FLAT', group: '内存索引', params: {} },
        BIN_IVF_FLAT: { label: 'BIN_IVF_FLAT', group: '内存索引', params: { nlist: 1024 } },
        IVF_RABITQ: { label: 'IVF_RABITQ', group: '内存索引', params: { nlist: 1024 } },
    },
    // SparseFloatVector 索引
    sparseVector: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        SPARSE_INVERTED_INDEX: { label: 'SPARSE_INVERTED_INDEX', group: '内存索引', params: { drop_ratio_build: 0.0 } },
    },
    // 标量索引
    scalarInverted: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        INVERTED: { label: 'INVERTED', group: '标量索引', params: {} },
    },
    scalarSorted: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        INVERTED: { label: 'INVERTED', group: '标量索引', params: {} },
        STL_SORT: { label: 'STL_SORT', group: '标量索引', params: {} },
    },
    scalarBitmap: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        BITMAP: { label: 'BITMAP', group: '标量索引(推荐)', params: {} },
        INVERTED: { label: 'INVERTED', group: '标量索引', params: {} },
    },
    varcharScalar: {
        AUTOINDEX: { label: 'AUTOINDEX', group: '自动索引', params: {} },
        INVERTED: { label: 'INVERTED', group: '标量索引', params: {} },
        NGRAM: { label: 'NGRAM', group: '标量索引', params: {} },
        BITMAP: { label: 'BITMAP', group: '标量索引', params: {} },
        Trie: { label: 'Trie', group: '标量索引', params: {} },
    },
} as const;

// 辅助函数：创建向量索引配置
const createVectorIndexConfig = (indexes: Record<string, IndexTemplateItem>, metrics: string[]): IndexConfig => ({
    category: '向量索引',
    indexes,
    metrics,
});

// 辅助函数：创建标量索引配置
const createScalarIndexConfig = (indexes: Record<string, IndexTemplateItem>): IndexConfig => ({
    category: '标量索引',
    indexes,
    metrics: [],
});

// 索引类型配置 - 根据字段数据类型分类
export const IndexConfigByDataType = {
    // 浮点向量类型（共享相同索引配置）
    [DataTypeMap.FloatVector]: createVectorIndexConfig(IndexTemplates.vectorCommon, ['COSINE', 'L2', 'IP']),
    [DataTypeMap.Float16Vector]: createVectorIndexConfig(IndexTemplates.vectorCommon, ['COSINE', 'L2', 'IP']),
    [DataTypeMap.BFloat16Vector]: createVectorIndexConfig(IndexTemplates.vectorCommon, ['COSINE', 'L2', 'IP']),
    [DataTypeMap.Int8Vector]: createVectorIndexConfig(IndexTemplates.vectorCommon, ['COSINE', 'L2', 'IP']),
    // BinaryVector
    [DataTypeMap.BinaryVector]: createVectorIndexConfig(IndexTemplates.binaryVector, ['HAMMING', 'JACCARD', 'MHJACCARD', 'TANIMOTO']),
    // SparseFloatVector
    [DataTypeMap.SparseFloatVector]: createVectorIndexConfig(IndexTemplates.sparseVector, ['IP', 'BM25']),

    // 标量类型
    [DataTypeMap.Int8]: createScalarIndexConfig(IndexTemplates.scalarSorted),
    [DataTypeMap.Int16]: createScalarIndexConfig(IndexTemplates.scalarSorted),
    [DataTypeMap.Int32]: createScalarIndexConfig(IndexTemplates.scalarSorted),
    [DataTypeMap.Int64]: createScalarIndexConfig(IndexTemplates.scalarSorted),
    [DataTypeMap.Float]: createScalarIndexConfig(IndexTemplates.scalarInverted),
    [DataTypeMap.Double]: createScalarIndexConfig(IndexTemplates.scalarInverted),
    [DataTypeMap.Bool]: createScalarIndexConfig(IndexTemplates.scalarBitmap),
    [DataTypeMap.VarChar]: createScalarIndexConfig(IndexTemplates.varcharScalar),
    [DataTypeMap.JSON]: createScalarIndexConfig(IndexTemplates.scalarInverted),
    [DataTypeMap.Array]: createScalarIndexConfig(IndexTemplates.scalarBitmap),
    [DataTypeMap.Geometry]: createScalarIndexConfig(IndexTemplates.scalarInverted),
    [DataTypeMap.Timestamptz]: createScalarIndexConfig(IndexTemplates.scalarSorted),
} as const;

// 向量类型集合
export const VectorTypes = new Set<number>([
    DataTypeMap.BinaryVector,
    DataTypeMap.FloatVector,
    DataTypeMap.Float16Vector,
    DataTypeMap.BFloat16Vector,
    DataTypeMap.SparseFloatVector,
    DataTypeMap.Int8Vector,
]);
