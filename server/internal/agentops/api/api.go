package api

import "mayfly-go/pkg/ioc"

// InitIoc 注册 agentops 的 API 组件（实现 req.RouterApi，由 starter 自动挂载路由）
func InitIoc() {
	ioc.Register(new(AgentTask))
	ioc.Register(new(Run))
	// 档位目录读接口：表单选项与筛选器的数据源（与后端注册表同源）
	ioc.Register(new(Profile))
	// 触发源目录读接口：同理，新增触发源不需改前端即可被配置
	ioc.Register(new(TriggerSource))
	ioc.Register(new(Trigger))
	ioc.Register(new(Proposal))
	ioc.Register(new(Governance))
	ioc.Register(new(Orchestration))
}
