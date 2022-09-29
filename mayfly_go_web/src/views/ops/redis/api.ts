import Api from '@/common/Api';

export const redisApi = {
    redisList : Api.create("/redis", 'get'),
    getRedisPwd: Api.create("/redis/{id}/pwd", 'get'),
    redisInfo: Api.create("/redis/{id}/info", 'get'),
    clusterInfo: Api.create("/redis/{id}/cluster-info", 'get'),
    saveRedis: Api.create("/redis", 'post'),
    delRedis: Api.create("/redis/{id}", 'delete'),
    // 获取权限列表
    scan: Api.create("/redis/{id}/{db}/scan", 'post'),
    getStringValue: Api.create("/redis/{id}/{db}/string-value", 'get'),
    saveStringValue: Api.create("/redis/{id}/{db}/string-value", 'post'),
    getHashValue: Api.create("/redis/{id}/{db}/hash-value", 'get'),
    hscan: Api.create("/redis/{id}/{db}/hscan", 'get'),
    hget: Api.create("/redis/{id}/{db}/hget", 'get'),
    hdel: Api.create("/redis/{id}/{db}/hdel", 'delete'),
    saveHashValue: Api.create("/redis/{id}/{db}/hash-value", 'post'),
    getSetValue: Api.create("/redis/{id}/{db}/set-value", 'get'),
    saveSetValue: Api.create("/redis/{id}/{db}/set-value", 'post'),
    del: Api.create("/redis/{id}/{db}/scan/{cursor}/{count}", 'delete'),
    delKey: Api.create("/redis/{id}/{db}/key", 'delete'),
    getListValue: Api.create("/redis/{id}/{db}/list-value", 'get'),
    saveListValue: Api.create("/redis/{id}/{db}/list-value", 'post'),
    setListValue: Api.create("/redis/{id}/{db}/list-value/lset", 'post'),
}