package constant

import "time"

const (
	MachineConnExpireTime = 60 * time.Minute
	DbConnExpireTime      = 45 * time.Minute
	RedisConnExpireTime   = 30 * time.Minute
	MongoConnExpireTime   = 30 * time.Minute

/****  开发测试使用   ****/
// MachineConnExpireTime = 20 * time.Second
// DbConnExpireTime      = 20 * time.Second
// RedisConnExpireTime   = 20 * time.Second
// MongoConnExpireTime   = 20 * time.Second
)
