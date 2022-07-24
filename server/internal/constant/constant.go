package constant

import "time"

const (
	MachineConnExpireTime = 60 * time.Minute
	DbConnExpireTime      = 45 * time.Minute
	RedisConnExpireTime   = 30 * time.Minute
	MongoConnExpireTime   = 30 * time.Minute

/****  开发测试使用   ****/
// MachineConnExpireTime = 4 * time.Minute
// DbConnExpireTime      = 2 * time.Minute
// RedisConnExpireTime   = 2 * time.Minute
// MongoConnExpireTime   = 2 * time.Minute
)
