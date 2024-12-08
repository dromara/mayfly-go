import { redisApi } from './api';
// import showCmdExecBox from './components/CmdExecBox';

export class RedisInst {
    /**
     * 实例id
     */
    id: number;

    /**
     * db
     */
    db: number;

    /**
     * 执行命令
     * @param cmd 命令列表如：['SET', 'key', 'value']
     * @returns 执行结果
     */
    async runCmd(cmd: any[]) {
        // // 工单流程定义存在，并且为写入命令时，弹窗输入工单相关信息并提交
        // if (this.flowProcdef && writeCmd[cmd[0].toUpperCase()]) {
        //     showCmdExecBox({
        //         id: this.id,
        //         db: this.db,
        //         flowProcdef: this.flowProcdef,
        //         cmd,
        //     });
        //     // 报错，阻止后续继续执行
        //     throw new Error('提交工单执行');
        // }

        return await redisApi.runCmd.request({
            id: this.id,
            db: this.db,
            cmd,
        });
    }
}

// const writeCmd = {
//     APPEND: 'APPEND key value',
//     BLMOVE: 'BLMOVE source destination LEFT|RIGHT LEFT|RIGHT timeout',
//     BLPOP: 'BLPOP key [key ...] timeout',
//     BRPOP: 'BRPOP key [key ...] timeout',
//     BRPOPLPUSH: 'BRPOPLPUSH source destination timeout',
//     BZPOPMAX: 'BZPOPMAX key [key ...] timeout',
//     BZPOPMIN: 'BZPOPMIN key [key ...] timeout',
//     COPY: 'COPY source destination [DB destination-db] [REPLACE]',
//     DECR: 'DECR key',
//     DECRBY: 'DECRBY key decrement',
//     DEL: 'DEL key [key ...]',
//     EVAL: 'EVAL script numkeys key [key ...] arg [arg ...]',
//     EVALSHA: 'EVALSHA sha1 numkeys key [key ...] arg [arg ...]',
//     EXPIRE: 'EXPIRE key seconds',
//     EXPIREAT: 'EXPIREAT key timestamp',
//     FLUSHALL: 'FLUSHALL',
//     FLUSHDB: 'FLUSHDB',
//     GEOADD: 'GEOADD key [NX|XX] [CH] longitude latitude member [longitude latitude member ...]',
//     GETDEL: 'GETDEL key',
//     GETSET: 'GETSET key value',
//     HDEL: 'HDEL key field [field ...]',
//     HINCRBY: 'HINCRBY key field increment',
//     HINCRBYFLOAT: 'HINCRBYFLOAT key field increment',
//     HMSET: 'HMSET key field value [field value ...]',
//     HSET: 'HSET key field value',
//     HSETNX: 'HSETNX key field value',
//     INCR: 'INCR key',
//     INCRBY: 'INCRBY key increment',
//     INCRBYFLOAT: 'INCRBYFLOAT key increment',
//     LINSERT: 'LINSERT key BEFORE|AFTER pivot value',
//     LMOVE: 'LMOVE source destination LEFT|RIGHT LEFT|RIGHT',
//     LPOP: 'LPOP key',
//     LPUSH: 'LPUSH key value [value ...]',
//     LPUSHX: 'LPUSHX key value',
//     LREM: 'LREM key count value',
//     LSET: 'LSET key index value',
//     LTRIM: 'LTRIM key start stop',
//     MIGRATE: 'MIGRATE host port key destination-db timeout',
//     MOVE: 'MOVE key db',
//     MSET: 'MSET key value [key value ...]',
//     MSETNX: 'MSETNX key value [key value ...]',
//     PERSIST: 'PERSIST key',
//     PEXPIRE: 'PEXPIRE key milliseconds',
//     PEXPIREAT: 'PEXPIREAT key milliseconds-timestamp',
//     PSETEX: 'PSETEX key milliseconds value',
//     PUBLISH: 'PUBLISH channel message',
//     RENAME: 'RENAME key newkey',
//     RENAMENX: 'RENAMENX key newkey',
//     RESTORE: 'RESTORE key ttl serialized-value',
//     RPOP: 'RPOP key',
//     RPOPLPUSH: 'RPOPLPUSH source destination',
//     RPUSH: 'RPUSH key value [value ...]',
//     RPUSHX: 'RPUSHX key value',
//     SADD: 'SADD key member [member ...]',
//     SCRIPT: ['SCRIPT EXISTS script [script ...]', 'SCRIPT FLUSH', 'SCRIPT KILL', 'SCRIPT LOAD script'],
//     SDIFFSTORE: 'SDIFFSTORE destination key [key ...]',
//     SET: 'SET key value',
//     SETBIT: 'SETBIT key offset value',
//     SETEX: 'SETEX key seconds value',
//     SETNX: 'SETNX key value',
//     SETRANGE: 'SETRANGE key offset value',
//     SINTERSTORE: 'SINTERSTORE destination key [key ...]',
//     SMOVE: 'SMOVE source destination member',
//     SORT: 'SORT key [BY pattern] [LIMIT offset count] [GET pattern [GET pattern ...]] [ASC|DESC] [ALPHA] [STORE destination]',
//     SPOP: 'SPOP key',
//     SREM: 'SREM key member [member ...]',
//     SUNIONSTORE: 'SUNIONSTORE destination key [key ...]',
//     SWAPDB: 'SWAPDB index1 index2',
//     UNLINK: 'UNLINK key [key ...]',
//     XADD: 'XADD key ID field string [field string ...]',
//     XDEL: 'XDEL key ID [ID ...]',
//     XGROUP: [
//         'XGROUP CREATE key groupname id|$ [MKSTREAM]',
//         'XGROUP CREATECONSUMER key groupname consumername',
//         'XGROUP DELCONSUMER key groupname consumername',
//         'XGROUP DESTROY key groupname',
//         'XGROUP SETID key groupname id|$',
//     ],
//     XTRIM: 'XTRIM key MAXLEN [~] count',
//     ZADD: 'ZADD key score member [score] [member]',
//     ZDIFFSTORE: 'ZDIFFSTORE destination numkeys key [key ...]',
//     ZINCRBY: 'ZINCRBY key increment member',
//     ZINTERSTORE: 'ZINTERSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]',
//     ZPOPMAX: 'ZPOPMAX key [count]',
//     ZPOPMIN: 'ZPOPMIN key [count]',
//     ZRANGESTORE: 'ZRANGESTORE dst src min max [BYSCORE|BYLEX] [REV] [LIMIT offset count]',
//     ZREM: 'ZREM key member [member ...]',
//     ZREMRANGEBYLEX: 'ZREMRANGEBYLEX key min max',
//     ZREMRANGEBYRANK: 'ZREMRANGEBYRANK key start stop',
//     ZREMRANGEBYSCORE: 'ZREMRANGEBYSCORE key min max',
//     ZUNIONSTORE: 'ZUNIONSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]',
// };
