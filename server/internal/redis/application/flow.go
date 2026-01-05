package application

import (
	flowapp "mayfly-go/internal/flow/application"
	"mayfly-go/pkg/ioc"
)

const (
	RedisRunCmdFlowBizType = "redis_run_cmd_flow"
)

func InitRedisFlowHandler() {
	flowapp.RegisterBizHandler(RedisRunCmdFlowBizType, ioc.Get[Redis]())
}
