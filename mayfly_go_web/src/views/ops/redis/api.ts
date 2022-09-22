import Api from '@/common/Api';

export const redisApi = {
    redisList : Api.create("/redis", 'get'),
    getRedisPwd: Api.create("/redis/{id}/pwd", 'get'),
    redisInfo: Api.create("/redis/{id}/info", 'get'),
    clusterInfo: Api.create("/redis/{id}/cluster-info", 'get'),
    saveRedis: Api.create("/redis", 'post'),
    delRedis: Api.create("/redis/{id}", 'delete'),
    // 获取权限列表
    scan: Api.create("/redis/{id}/scan", 'post'),
    getStringValue: Api.create("/redis/{id}/string-value", 'get'),
    saveStringValue: Api.create("/redis/{id}/string-value", 'post'),
    getHashValue: Api.create("/redis/{id}/hash-value", 'get'),
    hscan: Api.create("/redis/{id}/hscan", 'get'),
    hget: Api.create("/redis/{id}/hget", 'get'),
    hdel: Api.create("/redis/{id}/hdel", 'delete'),
    saveHashValue: Api.create("/redis/{id}/hash-value", 'post'),
    getSetValue: Api.create("/redis/{id}/set-value", 'get'),
    saveSetValue: Api.create("/redis/{id}/set-value", 'post'),
    del: Api.create("/redis/{id}/scan/{cursor}/{count}", 'delete'),
    delKey: Api.create("/redis/{id}/key", 'delete'),
    getListValue: Api.create("/redis/{id}/list-value", 'get'),
    saveListValue: Api.create("/redis/{id}/list-value", 'post'),
    setListValue: Api.create("/redis/{id}/list-value/lset", 'post'),
}