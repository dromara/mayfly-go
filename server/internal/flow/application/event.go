package application

import "mayfly-go/pkg/eventbus"

var flowEventBus eventbus.Bus[any] = eventbus.New[any]()

const (
	EventTopicFlowProcinstCreate    = "flow:procinst:create"    // 流程实例创建事件
	EventTopicFlowProcinstCancel    = "flow:procinst:cancel"    // 流程实例取消事件
	EventTopicFlowProcinstCompleted = "flow:procinst:completed" // 流程实例完成事件
)
