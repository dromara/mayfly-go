import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type {
    Redis,
    RedisKeyMeta,
    RedisKeyTargetForm,
    RedisKeysForm,
    RedisTargetParam,
    RedisKeyRenameForm,
    RedisKeyTtlForm,
    RedisListParam,
    RedisMemberPage,
    RedisMemberQueryForm,
    RedisMemberWriteForm,
    RedisScanRes,
    RedisViewDescriptor,
    RedisCommandSpec,
    RedisViewOpForm,
} from './types';

export const redisApi = {
    redisList: Api.newGet<PageResult<Redis>, RedisListParam>('/redis'),
    /** INFO 按分段解析（后端 map[string]map[string]string）：{ Server: { redis_version: ... }, Keyspace: { db0: 'keys=5,...' } } */
    redisInfo: Api.newGet<Record<string, Record<string, string>>>('/redis/{id}/info'),
    clusterInfo: Api.newGet<Record<string, unknown>>('/redis/{id}/cluster-info'),
    testConn: Api.newPost<void>('/redis/test-conn'),
    saveRedis: Api.newPost<number>('/redis'),
    delRedis: Api.newDelete<void>('/redis/{id}'),

    /** 数据视角描述符：表格列、能力位、表单结构、操作清单的单一真源 */
    views: Api.newGet<RedisViewDescriptor[], RedisTargetParam>('/redis/{id}/{db}/views'),

    /** 实例命令目录：命令控制台的输入提示与执行前确认依据（由实例自描述，不是平台内置表） */
    commands: Api.newGet<RedisCommandSpec[], RedisTargetParam>('/redis/{id}/{db}/commands'),

    /** key 元信息（类型 / 编码 / TTL / 内存 / 成员数 / 可切换视角） */
    keyMeta: Api.newGet<RedisKeyMeta, RedisKeyTargetForm>('/redis/{id}/{db}/key-meta'),

    /** 成员分页读取 */
    keyValues: Api.newPost<RedisMemberPage, RedisMemberQueryForm>('/redis/{id}/{db}/key-values'),
    /** 成员写操作：新增 / 修改 / 删除（批量）统一由 op 表达 */
    putKeyValue: Api.newPut<unknown, RedisMemberWriteForm>('/redis/{id}/{db}/key-value'),

    /** 视角扩展操作（集合运算、位统计、GEO 检索等） */
    runKeyOp: Api.newPost<unknown, RedisViewOpForm>('/redis/{id}/{db}/key-op'),

    setKeyTtl: Api.newPut<void, RedisKeyTtlForm>('/redis/{id}/{db}/key-ttl'),
    renameKey: Api.newPut<void, RedisKeyRenameForm>('/redis/{id}/{db}/key-rename'),
    copyKey: Api.newPost<void, RedisKeyRenameForm>('/redis/{id}/{db}/key-copy'),
    delKeys: Api.newPost<{ delNum: number }, RedisKeysForm>('/redis/{id}/{db}/del-keys'),

    // 获取key列表
    scan: Api.newPost<RedisScanRes>('/redis/{id}/{db}/scan'),

    // 执行命令，返回命令原始结果（随命令变化，由调用方泛型指定）
    runCmd: Api.newPost<unknown>('/redis/{id}/{db}/run-cmd'),
};
