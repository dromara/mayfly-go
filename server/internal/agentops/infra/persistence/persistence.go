package persistence

import (
	"mayfly-go/pkg/ioc"
)

// InitIoc 注册 agentops 仓储实现
func InitIoc() {
	ioc.Register(newRunRepo())
	ioc.Register(newAgentTaskRepo())
	ioc.Register(newTriggerRepo())
	ioc.Register(newProposalRepo())
}
