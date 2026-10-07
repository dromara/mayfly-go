package config

import (
	"cmp"
	sysapp "mayfly-go/internal/sys/application"
)

// ConfigKeyMongo Mongo 数据面相关配置在系统配置表中的 key。
const ConfigKeyMongo string = "MongoConfig"

// 默认值：数据面所有硬性上限与超时都从这里取，不允许在 handler 里写死数字。
const (
	// defaultExecTl 单条 Mongo 操作的服务端执行时间上限（秒）。
	// 没有它，一个全集合扫描或 $where 慢查询会永久占住 goroutine，客户端断开也不会取消。
	defaultExecTl = 60
	// defaultLimit 未指定 limit 时使用的条数。绝不能按「不限制」处理：不限条数会把整个集合读进内存。
	defaultLimit = 50
	// defaultMaxResultSet 允许查询的最大结果集条数，超出即截断并回 truncated 标记。
	defaultMaxResultSet = 500
	// defaultPoolSize 单个 mongo 实例的连接池大小。
	// 旧实现写死 1，等于同一实例上所有操作串行；driver 自身会做多路复用，无需我们收窄。
	defaultPoolSize = 10
)

type Mongo struct {
	ExecTl       int // 单条操作执行时间上限（秒），超过即取消
	DefaultLimit int // 查询未指定 limit 时的默认条数
	MaxResultSet int // 允许查询的最大结果集条数
	PoolSize     int // 单实例连接池大小
}

func GetMongo() *Mongo {
	c := sysapp.GetConfigApp().GetConfig(ConfigKeyMongo)
	jm := c.GetJsonM()

	mongoConf := new(Mongo)
	mongoConf.ExecTl = cmp.Or(jm.GetInt("execTl"), defaultExecTl)
	mongoConf.DefaultLimit = cmp.Or(jm.GetInt("defaultLimit"), defaultLimit)
	mongoConf.MaxResultSet = cmp.Or(jm.GetInt("maxResultSet"), defaultMaxResultSet)
	mongoConf.PoolSize = cmp.Or(jm.GetInt("poolSize"), defaultPoolSize)

	return mongoConf
}

// Limit 归一化请求条数：未指定或非法时取默认值，并始终不超过最大结果集。
//
// 旧实现只在 limit != 0 时校验上限，limit=0 会落到 SetLimit(0)（即不限条数），
// 由调用方一个疏忽换来整集合入内存。这里收口为「永远有上限」。
func (m *Mongo) Limit(limit int64) int64 {
	if limit <= 0 {
		limit = int64(m.DefaultLimit)
	}
	if max := int64(m.MaxResultSet); max > 0 && limit > max {
		return max
	}
	return limit
}
