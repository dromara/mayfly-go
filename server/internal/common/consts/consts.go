package consts

import "time"

const (
	AdminId = 1

	MachineConnExpireTime = 60 * time.Minute
	DbConnExpireTime      = 120 * time.Minute
	RedisConnExpireTime   = 30 * time.Minute
	MongoConnExpireTime   = 30 * time.Minute

	/****  开发测试使用   ****/
	// MachineConnExpireTime = 4 * time.Minute
	// DbConnExpireTime      = 2 * time.Minute
	// RedisConnExpireTime   = 2 * time.Minute
	// MongoConnExpireTime   = 2 * time.Minute

	TagResourceTypeMachine = 1
	TagResourceTypeDb      = 2
	TagResourceTypeRedis   = 3
	TagResourceTypeMongo   = 4

	// 删除机器的事件主题名
	DeleteMachineEventTopic = "machine:delete"
)
