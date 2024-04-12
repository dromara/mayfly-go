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

	ResourceTypeMachine int8 = 1
	ResourceTypeDb      int8 = 2
	ResourceTypeRedis   int8 = 3
	ResourceTypeMongo   int8 = 4

	// 删除机器的事件主题名
	DeleteMachineEventTopic = "machine:delete"
)
