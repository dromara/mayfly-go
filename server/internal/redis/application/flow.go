package application

import (
	flowapp "mayfly-go/internal/flow/application"
	"mayfly-go/pkg/ioc"
)

const (
	RedisRunWriteCmdFlowBizType = "redis_run_write_cmd_flow"
)

func InitRedisFlowHandler() {
	flowapp.RegisterBizHandler(RedisRunWriteCmdFlowBizType, ioc.Get[Redis]("RedisApp"))
}
