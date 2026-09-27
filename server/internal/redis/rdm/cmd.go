package rdm

import (
	"strings"

	"github.com/spf13/cast"
)

// write cmd
var writeCmd = map[string]string{
	"APPEND":           "APPEND key value",
	"BLMOVE":           "BLMOVE source destination LEFT|RIGHT LEFT|RIGHT timeout",
	"BLPOP":            "BLPOP key [key ...] timeout",
	"BRPOP":            "BRPOP key [key ...] timeout",
	"BRPOPLPUSH":       "BRPOPLPUSH source destination timeout",
	"BZPOPMAX":         "BZPOPMAX key [key ...] timeout",
	"BZPOPMIN":         "BZPOPMIN key [key ...] timeout",
	"COPY":             "COPY source destination [DB destination-db] [REPLACE]",
	"DECR":             "DECR key",
	"DECRBY":           "DECRBY key decrement",
	"DEL":              "DEL key [key ...]",
	"EVAL":             "EVAL script numkeys key [key ...] arg [arg ...]",
	"EVALSHA":          "EVALSHA sha1 numkeys key [key ...] arg [arg ...]",
	"EXPIRE":           "EXPIRE key seconds",
	"EXPIREAT":         "EXPIREAT key timestamp",
	"FLUSHALL":         "FLUSHALL",
	"FLUSHDB":          "FLUSHDB",
	"GEOADD":           "GEOADD key [NX|XX] [CH] longitude latitude member [longitude latitude member ...]",
	"GETDEL":           "GETDEL key",
	"GETEX":            "GETEX key [EX seconds|PX milliseconds|EXAT timestamp|PERSIST]",
	"GETSET":           "GETSET key value",
	"HDEL":             "HDEL key field [field ...]",
	"HEXPIRE":          "HEXPIRE key seconds [NX | XX | GT | LT] FIELDS numfields field [field ...]",
	"HPERSIST":         "HPERSIST key FIELDS numfields field [field ...]",
	"HINCRBY":          "HINCRBY key field increment",
	"HINCRBYFLOAT":     "HINCRBYFLOAT key field increment",
	"HMSET":            "HMSET key field value [field value ...]",
	"HSET":             "HSET key field value",
	"HSETNX":           "HSETNX key field value",
	"INCR":             "INCR key",
	"INCRBY":           "INCRBY key increment",
	"INCRBYFLOAT":      "INCRBYFLOAT key increment",
	"LINSERT":          "LINSERT key BEFORE|AFTER pivot value",
	"LMOVE":            "LMOVE source destination LEFT|RIGHT LEFT|RIGHT",
	"LPOP":             "LPOP key",
	"LPUSH":            "LPUSH key value [value ...]",
	"LPUSHX":           "LPUSHX key value",
	"LREM":             "LREM key count value",
	"LSET":             "LSET key index value",
	"LTRIM":            "LTRIM key start stop",
	"MIGRATE":          "MIGRATE host port key destination-db timeout",
	"MOVE":             "MOVE key db",
	"MSET":             "MSET key value [key value ...]",
	"MSETNX":           "MSETNX key value [key value ...]",
	"PERSIST":          "PERSIST key",
	"PEXPIRE":          "PEXPIRE key milliseconds",
	"PEXPIREAT":        "PEXPIREAT key milliseconds-timestamp",
	"PSETEX":           "PSETEX key milliseconds value",
	"PUBLISH":          "PUBLISH channel message",
	"RENAME":           "RENAME key newkey",
	"RENAMENX":         "RENAMENX key newkey",
	"RESTORE":          "RESTORE key ttl serialized-value",
	"RPOP":             "RPOP key",
	"RPOPLPUSH":        "RPOPLPUSH source destination",
	"RPUSH":            "RPUSH key value [value ...]",
	"RPUSHX":           "RPUSHX key value",
	"SADD":             "SADD key member [member ...]",
	"SCRIPT":           "SCRIPT EXISTS script [script ...], 'SCRIPT FLUSH', 'SCRIPT KILL', SCRIPT LOAD script",
	"SDIFFSTORE":       "SDIFFSTORE destination key [key ...]",
	"SET":              "SET key value",
	"SETBIT":           "SETBIT key offset value",
	"SETEX":            "SETEX key seconds value",
	"SETNX":            "SETNX key value",
	"SETRANGE":         "SETRANGE key offset value",
	"SINTERSTORE":      "SINTERSTORE destination key [key ...]",
	"SMOVE":            "SMOVE source destination member",
	"SORT":             "SORT key [BY pattern] [LIMIT offset count] [GET pattern [GET pattern ...]] [ASC|DESC] [ALPHA] [STORE destination]",
	"SPOP":             "SPOP key",
	"SREM":             "SREM key member [member ...]",
	"SUNIONSTORE":      "SUNIONSTORE destination key [key ...]",
	"SWAPDB":           "SWAPDB index1 index2",
	"UNLINK":           "UNLINK key [key ...]",
	"XADD":             "XADD key ID field string [field string ...]",
	"XDEL":             "XDEL key ID [ID ...]",
	"XGROUP":           "XGROUP CREATE key groupname id|$ [MKSTREAM], XGROUP CREATECONSUMER key groupname consumername, XGROUP DELCONSUMER key groupname consumername, XGROUP DESTROY key groupname, XGROUP SETID key groupname id|$",
	"XTRIM":            "XTRIM key MAXLEN [~] count",
	"XACK":             "XACK key group name [ID ...]",
	"PFADD":            "PFADD key element [element ...]",
	"PFMERGE":          "PFMERGE destkey sourcekey [sourcekey ...]",
	"ZADD":             "ZADD key score member [score] [member]",
	"ZDIFFSTORE":       "ZDIFFSTORE destination numkeys key [key ...]",
	"ZINCRBY":          "ZINCRBY key increment member",
	"ZINTERSTORE":      "ZINTERSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]",
	"ZPOPMAX":          "ZPOPMAX key [count]",
	"ZPOPMIN":          "ZPOPMIN key [count]",
	"ZRANGESTORE":      "ZRANGESTORE dst src min max [BYSCORE|BYLEX] [REV] [LIMIT offset count]",
	"ZREM":             "ZREM key member [member ...]",
	"ZREMRANGEBYLEX":   "ZREMRANGEBYLEX key min max",
	"ZREMRANGEBYRANK":  "ZREMRANGEBYRANK key start stop",
	"ZREMRANGEBYSCORE": "ZREMRANGEBYSCORE key min max",
	"ZUNIONSTORE":      "ZUNIONSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]",
}

// IsWriteCmd 判断命令是否写命令。
//
// Redis 命令名大小写不敏感，而命令控制台原样透传用户输入，因此这里统一大写化再查表：
// 否则 `hset foo bar baz` 会被判成读命令，写权限校验与工单审批同时失守
func IsWriteCmd(cmd any) bool {
	_, ok := writeCmd[strings.ToUpper(cast.ToString(cmd))]
	return ok
}

// dangerousCmd 可能阻塞实例、清空数据或绕过数据权限的命令，只允许拥有独立数据操作权限的账号执行
var dangerousCmd = map[string]string{
	"FLUSHALL":  "FLUSHALL [ASYNC|SYNC]",
	"FLUSHDB":   "FLUSHDB [ASYNC|SYNC]",
	"KEYS":      "KEYS pattern",
	"CONFIG":    "CONFIG GET|SET|RESETSTAT ...",
	"DEBUG":     "DEBUG ...",
	"SHUTDOWN":  "SHUTDOWN [NOSAVE|SAVE]",
	"SLAVEOF":   "SLAVEOF host port",
	"REPLICAOF": "REPLICAOF host port",
	"SWAPDB":    "SWAPDB index1 index2",
	"SCRIPT":    "SCRIPT EXISTS|FLUSH|KILL|LOAD ...",
	"EVAL":      "EVAL script numkeys key [key ...] arg [arg ...]",
	"EVALSHA":   "EVALSHA sha1 numkeys key [key ...] arg [arg ...]",
	"FUNCTION":  "FUNCTION DELETE|FLUSH|RESTORE ...",
	"CLUSTER":   "CLUSTER FAILZONE|FORGET|RESET|SAVECONFIG ...",
	"ACL":       "ACL DELUSER|SETUSER|SAVE ...",
	"MONITOR":   "MONITOR",
}

// IsDangerousCmd 判断命令是否为高危命令
func IsDangerousCmd(cmd any) bool {
	_, ok := dangerousCmd[strings.ToUpper(cast.ToString(cmd))]
	return ok
}
