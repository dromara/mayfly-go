import Api from '@/common/Api';

export const redisApi = {
    redisList: Api.newGet('/redis'),
    redisTags: Api.newGet('/redis/tags'),
    getRedisPwd: Api.newGet('/redis/{id}/pwd'),
    redisInfo: Api.newGet('/redis/{id}/info'),
    clusterInfo: Api.newGet('/redis/{id}/cluster-info'),
    testConn: Api.newPost('/redis/test-conn'),
    saveRedis: Api.newPost('/redis'),
    delRedis: Api.newDelete('/redis/{id}'),

    keyInfo: Api.newGet('/redis/{id}/{db}/key-info'),
    keyTtl: Api.newGet('/redis/{id}/{db}/key-ttl'),
    keyMemuse: Api.newGet('/redis/{id}/{db}/key-memuse'),
    renameKey: Api.newPost('/redis/{id}/{db}/rename-key'),
    expireKey: Api.newPost('/redis/{id}/{db}/expire-key'),
    persistKey: Api.newDelete('/redis/{id}/{db}/persist-key'),

    // 获取权限列表
    scan: Api.newPost('/redis/{id}/{db}/scan'),
    getString: Api.newGet('/redis/{id}/{db}/string-value'),
    setString: Api.newPost('/redis/{id}/{db}/string-value'),
    getHashValue: Api.newGet('/redis/{id}/{db}/hash-value'),
    hscan: Api.newGet('/redis/{id}/{db}/hscan'),
    hget: Api.newGet('/redis/{id}/{db}/hget'),
    hset: Api.newPost('/redis/{id}/{db}/hset'),
    hdel: Api.newDelete('/redis/{id}/{db}/hdel'),
    saveHashValue: Api.newPost('/redis/{id}/{db}/hash-value'),

    getSetValue: Api.newGet('/redis/{id}/{db}/set-value'),
    scard: Api.newGet('/redis/{id}/{db}/scard'),
    sscan: Api.newPost('/redis/{id}/{db}/sscan'),
    sadd: Api.newPost('/redis/{id}/{db}/sadd'),
    srem: Api.newPost('/redis/{id}/{db}/srem'),
    saveSetValue: Api.newPost('/redis/{id}/{db}/set-value'),

    del: Api.newDelete('/redis/{id}/{db}/scan/{cursor}/{count}'),
    delKey: Api.newDelete('/redis/{id}/{db}/key'),
    flushDb: Api.newDelete('/redis/{id}/{db}/flushdb'),

    lrem: Api.newPost('/redis/{id}/{db}/lrem'),
    getListValue: Api.newGet('/redis/{id}/{db}/list-value'),
    saveListValue: Api.newPost('/redis/{id}/{db}/list-value'),
    setListValue: Api.newPost('/redis/{id}/{db}/list-value/lset'),

    zcard: Api.newGet('/redis/{id}/{db}/zcard'),
    zscan: Api.newGet('/redis/{id}/{db}/zscan'),
    zrevrange: Api.newGet('/redis/{id}/{db}/zrevrange'),
    zadd: Api.newPost('/redis/{id}/{db}/zadd'),
    zrem: Api.newPost('/redis/{id}/{db}/zrem'),
};
