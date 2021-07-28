import Api from '@/common/Api';

export const redisApi = {
    redisList : Api.create("/redis", 'get'),
    redisInfo: Api.create("/redis/{id}/info", 'get'),
    saveRedis: Api.create("/redis", 'post'),
    delRedis: Api.create("/redis/{id}", 'delete'),
    // 获取权限列表
    scan: Api.create("/redis/{id}/scan/{cursor}/{count}", 'get'),
    getStringValue: Api.create("/redis/{id}/string-value", 'get'),
    saveStringValue: Api.create("/redis/{id}/string-value", 'post'),
    getHashValue: Api.create("/redis/{id}/hash-value", 'get'),
    getSetValue: Api.create("/redis/{id}/set-value", 'get'),
    saveHashValue: Api.create("/redis/{id}/hash-value", 'post'),
    del: Api.create("/redis/{id}/scan/{cursor}/{count}", 'delete'),
    delKey: Api.create("/redis/{id}/key", 'delete'),
}