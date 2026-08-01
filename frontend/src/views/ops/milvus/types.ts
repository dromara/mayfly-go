/**
 * Milvus 模块类型定义
 * 对应后端: milvus/domain/entity/milvus.go
 */
import type { MachineAuthCert } from '@/views/ops/machine/types';
import type { BaseModel, PageParam } from '@/types/common';

/** Milvus 实例实体 (对应 entity.Milvus) */
export interface Milvus extends BaseModel {
    id: number;
    code: string;
    name: string;
    host: string;
    database: string;
    sshTunnelMachineId: number;
    remark?: string;
    authCerts?: MachineAuthCert[];
    selectAuthCert?: MachineAuthCert;
}

/** Milvus 实例列表查询参数 */
export interface MilvusListParam extends PageParam {
    tagPath?: string;
}

export interface IDatabase {
    id: string;
    name: string;
    create_time?: string;
    [key: string]: unknown;
}

export interface ICollection {
    name: string;
    description?: string;
    loaded?: boolean;
    Loaded?: boolean;
    LoadedPercentage?: number;
    created_time?: string;
    shardsNum?: number;
    aliases?: string[];
}

/** 分区信息 (对应后端 vo.PartitionVO) */
export interface IPartition {
    id?: number;
    name: string;
    createTime?: string;
    rowCount?: number;
    collectionName?: string;
}

export interface IIndex {
    collectionName: string;
    fieldName: string;
    indexName: string;
    indexType: string;
    metricType: string;
}

export interface IUser {
    name: string;
    username: string;
    roles?: string[];
}

export interface IRole {
    name: string;
}

/** 资源组节点信息 (对应 entity.NodeInfo) */
export interface IResourceGroupNode {
    NodeID?: number;
    Address?: string;
    HostName?: string;
    [key: string]: unknown;
}

/** 资源组信息 (listResourceGroups 返回名称数组, describeResourceGroup 返回 entity.ResourceGroup 结构) */
export interface IResourceGroup {
    name: string;
    Name?: string;
    Capacity?: number;
    capacity?: number;
    NumAvailableNode?: number;
    numAvailableNode?: number;
    availableNodes?: number;
    NumLoadedReplica?: Record<string, number>;
    numLoadedReplica?: Record<string, number>;
    Config?: { Requests?: { NodeNum?: number }; Limits?: { NodeNum?: number }; [key: string]: unknown };
    config?: { requests?: { nodeNum?: number }; limits?: { nodeNum?: number }; [key: string]: unknown };
    Nodes?: IResourceGroupNode[];
    nodes?: IResourceGroupNode[];
    [key: string]: unknown;
}

/** Milvus 健康检查详情 */
export interface MilvusHealthDetail {
    name: string;
    message?: string;
    healthy: boolean;
}

export interface IPrivilegeGroup {
    GroupName: string;
    Privileges: string[];
}

/** Milvus 权限项 (对应 entity.GrantItem, 角色权限详情) */
export interface IMilvusPrivilegeItem {
    Object?: string;
    ObjectName?: string;
    RoleName?: string;
    Grantor?: string;
    Privilege?: string;
    DbName?: string;
    [key: string]: unknown;
}

/** 角色详情 (对应 entity.Role, describeRole 返回) */
export interface IMilvusRoleDetail {
    RoleName?: string;
    Privileges?: IMilvusPrivilegeItem[];
    [key: string]: unknown;
}

/** Collection 字段结构 (对应 entity.Field 序列化, 大驼峰字段名) */
export interface IMilvusField {
    Name: string;
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

/** Collection 详情 (对应 entity.Collection 序列化, describeCollection 返回) */
export interface IMilvusCollectionDetail {
    ID?: number;
    Name?: string;
    name?: string;
    Schema?: { CollectionName?: string; Description?: string; AutoID?: boolean; Fields?: IMilvusField[]; EnableDynamicField?: boolean; [key: string]: unknown };
    schema?: { CollectionName?: string; Description?: string; AutoID?: boolean; Fields?: IMilvusField[]; EnableDynamicField?: boolean; [key: string]: unknown };
    Loaded?: boolean;
    ConsistencyLevel?: number;
    ShardNum?: number;
    description?: string;
    Properties?: Record<string, string>;
    [key: string]: unknown;
}

/** 查询结果 (对应 mvm.QueryResult) */
export interface IMilvusQueryResult {
    count?: number;
    fields?: string[];
    data?: Record<string, unknown>[];
    page?: number;
    pageSize?: number;
    total?: number;
}

/** 数据插入/导入结果 */
export interface IMilvusInsertResult {
    insertCount?: number;
}

/** Mock 样本数据生成结果 */
export interface IMilvusMockDataResult {
    data: Record<string, unknown>[];
}

/** 数据库详情 (对应 entity.Database, describeDatabase 返回) */
export interface IMilvusDatabaseDetail {
    Name?: string;
    Properties?: Record<string, string>;
    [key: string]: unknown;
}

/** 健康检查状态 (对应 vo.HealthStatusVO) */
export interface IMilvusHealthStatus {
    isHealthy?: boolean;
    IsHealthy?: boolean;
    reasons?: (string | { name?: string; Name?: string; message?: string; Message?: string; healthy?: boolean })[];
    Reasons?: (string | { name?: string; Name?: string; message?: string; Message?: string; healthy?: boolean })[];
    quotaStates?: Record<string, unknown>[];
    [key: string]: unknown;
}
