import Api from '@/common/Api';

export const redisApi = {
    redisList: Api.newGet("/redis"),
    getRedisPwd: Api.newGet("/redis/{id}/pwd"),
    redisInfo: Api.newGet("/redis/{id}/info"),
    clusterInfo: Api.newGet("/redis/{id}/cluster-info"),
    saveRedis: Api.newPost("/redis"),
    delRedis: Api.newDelete("/redis/{id}"),
    // 获取权限列表
    scan: Api.newPost("/redis/{id}/{db}/scan"),
    getStringValue: Api.newGet("/redis/{id}/{db}/string-value"),
    saveStringValue: Api.newPost("/redis/{id}/{db}/string-value"),
    getHashValue: Api.newGet("/redis/{id}/{db}/hash-value"),
    hscan: Api.newGet("/redis/{id}/{db}/hscan"),
    hget: Api.newGet("/redis/{id}/{db}/hget"),
    hdel: Api.newDelete("/redis/{id}/{db}/hdel"),
    saveHashValue: Api.newPost("/redis/{id}/{db}/hash-value"),
    getSetValue: Api.newGet("/redis/{id}/{db}/set-value"),
    saveSetValue: Api.newPost("/redis/{id}/{db}/set-value"),
    del: Api.newDelete("/redis/{id}/{db}/scan/{cursor}/{count}"),
    delKey: Api.newDelete("/redis/{id}/{db}/key"),
    getListValue: Api.newGet("/redis/{id}/{db}/list-value"),
    saveListValue: Api.newPost("/redis/{id}/{db}/list-value"),
    setListValue: Api.newPost("/redis/{id}/{db}/list-value/lset"),
}