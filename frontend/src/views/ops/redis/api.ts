import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type { Redis, RedisKeyInfo, RedisScanRes, RedisListParam } from './types';

export const redisApi = {
    redisList: Api.newGet<PageResult<Redis>, RedisListParam>('/redis'),
    redisTags: Api.newGet<Redis[]>('/redis/tags'),
    getRedisPwd: Api.newGet<Record<string, string>>('/redis/{id}/pwd'),
    redisInfo: Api.newGet<Record<string, unknown>>('/redis/{id}/info'),
    clusterInfo: Api.newGet<Record<string, unknown>>('/redis/{id}/cluster-info'),
    testConn: Api.newPost<void>('/redis/test-conn'),
    saveRedis: Api.newPost<void>('/redis'),
    delRedis: Api.newDelete<void>('/redis/{id}'),

    keyInfo: Api.newGet<RedisKeyInfo>('/redis/{id}/{db}/key-info'),
    keyTtl: Api.newGet<number>('/redis/{id}/{db}/key-ttl'),
    keyMemuse: Api.newGet<number>('/redis/{id}/{db}/key-memuse'),

    // 获取key列表
    scan: Api.newPost<RedisScanRes>('/redis/{id}/{db}/scan'),

    // 执行命令，返回命令原始结果（随命令变化，由调用方泛型指定）
    runCmd: Api.newPost<unknown>('/redis/{id}/{db}/run-cmd'),
};

export function splitargs(line: string) {
    var ret: string[] = [];
    if (!line || typeof line.length !== 'number') {
        return ret;
    }

    var len = line.length;
    var pos = 0;
    while (true) {
        // skip blanks
        while (pos < len && isspace(line[pos])) {
            pos += 1;
        }

        if (pos === len) {
            break;
        }

        var inq = false; // if we are in "quotes"
        var insq = false; // if we are in "single quotes"
        var done = false;
        var current = '';

        while (!done) {
            var c = line[pos];
            if (inq) {
                if (c === '\\' && pos + 1 < len) {
                    pos += 1;

                    switch (line[pos]) {
                        case 'n':
                            c = '\n';
                            break;
                        case 'r':
                            c = '\r';
                            break;
                        case 't':
                            c = '\t';
                            break;
                        case 'b':
                            c = '\b';
                            break;
                        case 'a':
                            c = 'a';
                            break;
                        default:
                            c = line[pos];
                            break;
                    }
                    current += c;
                } else if (c === '"') {
                    // closing quote must be followed by a space or
                    // nothing at all.
                    if (pos + 1 < len && !isspace(line[pos + 1])) {
                        throw new Error("Expect '\"' followed by a space or nothing, got '" + line[pos + 1] + "'.");
                    }
                    done = true;
                } else if (pos === len) {
                    throw new Error('Unterminated quotes.');
                } else {
                    current += c;
                }
            } else if (insq) {
                if (c === '\\' && line[pos + 1] === "'") {
                    pos += 1;
                    current += "'";
                } else if (c === "'") {
                    // closing quote must be followed by a space or
                    // nothing at all.
                    if (pos + 1 < len && !isspace(line[pos + 1])) {
                        throw new Error('Expect "\'" followed by a space or nothing, got "' + line[pos + 1] + '".');
                    }
                    done = true;
                } else if (pos === len) {
                    throw new Error('Unterminated quotes.');
                } else {
                    current += c;
                }
            } else {
                if (pos === len) {
                    done = true;
                } else {
                    switch (c) {
                        case ' ':
                        case '\n':
                        case '\r':
                        case '\t':
                            done = true;
                            break;
                        case '"':
                            inq = true;
                            break;
                        case "'":
                            insq = true;
                            break;
                        default:
                            current += c;
                            break;
                    }
                }
            }
            if (pos < len) {
                pos += 1;
            }
        }
        ret.push(current);
        current = '';
    }

    return ret;
}

function isspace(ch: string) {
    return ch === ' ' || ch === '\t' || ch === '\n' || ch === '\r' || ch === '\v' || ch === '\f';
}
