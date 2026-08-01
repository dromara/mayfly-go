/**
 * Redis 模块类型定义
 * 对应后端: redis/domain/entity/redis.go
 */
import type { BaseModel, PageParam } from '@/types/common';

/** Redis 实体 (对应 entity.Redis) */
export interface Redis extends BaseModel {
    id: number;
    code: string;
    name: string;
    host: string;
    port?: number;
    mode: string;
    db: string;
    sshTunnelMachineId: number;
    remark: string;
}

/** Redis 列表查询参数 (对应后端 entity.RedisQuery) */
export interface RedisListParam extends PageParam {
    tagPath?: string;
    id?: number;
}

/** Redis 保存表单 */
export interface RedisSaveForm {
    id?: number | null;
    code?: string;
    tagCodePaths?: string[];
    name?: string | null;
    mode?: string;
    host?: string;
    username?: string | null;
    password?: string | null;
    redisNodePassword?: string | null;
    db?: string;
    remark?: string;
    sshTunnelMachineId?: number;
}

/** Redis key 信息 */
export interface RedisKeyInfo {
    key: string;
    type: string;
    ttl: number;
    size: number;
}

/** Redis scan 结果 (对应后端 vo.Keys) */
export interface RedisScanRes {
    /** 各节点 scan 游标 */
    cursor: Record<string, number>;
    keys: string[];
    dbSize: number;
}

/** Redis 信息 */
export interface RedisInfo {
    [key: string]: string;
}

/** Redis 集群信息 */
export interface RedisClusterInfo {
    clusterEnabled: boolean;
    nodes: RedisClusterNode[];
}

export interface RedisClusterNode {
    id: string;
    addr: string;
    flags: string;
    masterId: string;
    pingSent: number;
    pongRecv: number;
    configEpoch: number;
    linkState: string;
    slots: string[];
}
