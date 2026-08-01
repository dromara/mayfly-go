/**
 * ES 模块类型定义
 * 对应后端: es/domain/entity/es_instance.go
 */
import type { MachineAuthCert } from '@/views/ops/machine/types';
import type { BaseModel, PageParam } from '@/types/common';

/** ES 实例实体 (对应 entity.EsInstance) */
export interface EsInstance extends BaseModel {
    id: number;
    code: string;
    name: string;
    protocol: string;
    host: string;
    port: number;
    network: string;
    version: string;
    remark?: string;
    sshTunnelMachineId: number;
    authCerts?: MachineAuthCert[];
    selectAuthCert?: MachineAuthCert;
}

/** ES 实例列表查询参数 */
export interface EsInstanceListParam extends PageParam {
    tagPath?: string;
}

/** ES 索引信息 */
export interface EsIndex {
    index: string;
    health: string;
    status: string;
    'docs.count': string;
    'store.size': string;
    pri: string;
    rep: string;
}

/** ES 导出进度 (对应后端 exportProgress) */
export interface EsExportProgress {
    total: number;
    processed: number;
    phase: string;
    done: boolean;
    error?: string;
}

/** ES 代理请求响应 (动态结构) */
export interface EsProxyRes {
    error?: unknown;
    failures?: unknown[];
    msg?: string;
    [key: string]: unknown;
}

/** 名称-值键值对 (用于 instInfo/clusterHealth 扁平化展示) */
export interface EsNameValue {
    name: string;
    value: unknown;
}

/** ES 节点统计信息 (/_nodes/stats 响应中的节点结构) */
export interface EsNodeStats {
    key: string;
    name: string;
    ip: string;
    timestamp: number;
    roles: string[];
    indices: {
        docs: { count: number; deleted: number };
        store: { size_in_bytes: number };
    };
    os: {
        mem: { total_in_bytes: number; used_in_bytes: number; used_percent: number };
        cpu: { percent: number };
    };
    jvm: {
        mem: { heap_max_in_bytes: number; heap_used_in_bytes: number; heap_used_percent: number };
    };
    fs: {
        total: { total_in_bytes: number; free_in_bytes: number };
    };
}

/** /_nodes/stats 响应结构 */
export interface EsNodesStatsRes {
    _nodes: { total: number; successful: number; failed: number };
    nodes: Record<string, Omit<EsNodeStats, 'key'>>;
}

/** ES mapping 字段定义 */
export interface EsMappingField {
    type?: string;
    fields?: Record<string, EsMappingField>;
    [key: string]: unknown;
}

/** /_cluster/state 响应结构 (仅建模 metadata.indices 部分) */
export interface EsClusterStateRes {
    metadata: {
        indices: Record<
            string,
            {
                mappings?: {
                    _doc?: { properties?: Record<string, EsMappingField> };
                    properties?: Record<string, EsMappingField>;
                };
            }
        >;
    };
}

/** 索引可分析字段信息 */
export interface EsIdxField {
    name: string;
    fields: string[];
}

/** _analyze 接口返回的分词结果 */
export interface EsAnalyzeToken {
    token: string;
    type: string;
    position: number;
    start_offset: number;
    end_offset: number;
}

/** _analyze 响应结构 */
export interface EsAnalyzeRes {
    tokens: EsAnalyzeToken[];
}

/** ES _search 请求参数 */
export interface EsSearchParam {
    sort?: Record<string, unknown>;
    query?: {
        bool?: {
            must?: Record<string, unknown>[];
            should?: Record<string, unknown>[];
            must_not?: Record<string, unknown>[];
            minimum_should_match?: number;
            [key: string]: unknown;
        };
        [key: string]: unknown;
    };
    aggs?: Record<string, unknown>;
    from: number;
    size: number;
    track_total_hits?: boolean;
    [key: string]: unknown;
}

/** _search hits 中的单条文档 */
export interface EsHit {
    _index?: string;
    _id: string;
    _score?: number;
    _source: Record<string, unknown>;
}

/** ES _search 响应结构 */
export interface EsSearchRes extends EsProxyRes {
    hits: {
        total: { value: number; relation?: string };
        hits: EsHit[];
    };
    took?: number;
    timed_out?: boolean;
}

/** /_stats 响应结构 (仅建模使用部分) */
export interface EsIndexStatsRes {
    indices: Record<
        string,
        {
            primaries?: { docs?: { count?: number } };
            health?: string;
            status?: string;
        }
    >;
}

/** /_count 响应结构 */
export interface EsCountRes {
    count?: number;
}

/** /_cat/indices 返回的索引信息 */
export interface EsIndexInfo {
    index: string;
    health?: string;
    status?: string;
    uuid?: string;
    pri?: string;
    rep?: string;
    'docs.count'?: string | number;
    'docs.deleted'?: string;
    'store.size'?: string;
    sc?: string;
    cd?: string;
    [key: string]: unknown;
}

/** ES 数据表格列定义 */
export interface EsColumn {
    title: string;
    width: number;
    key: string;
    dataKey?: string;
    class?: string;
    align?: string;
    hidden?: boolean;
    _filterd?: boolean;
    _show?: boolean;
}

/** ES 文档行数据 */
export type EsDoc = Record<string, unknown> & { _id: string; _score?: number; src?: string; _selected?: boolean };
