package evaluator

import (
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/pkg/ioc"
)

// evaluatorRegistry 评估器注册表
var evaluatorRegistry []service.AlertEvaluator

// registerEvaluator 构造评估器并登记到 IoC 容器与评估器注册表。
// 形参为 service.AlertEvaluator 接口类型：评估器未实现接口时编译期即失败，
// 不存在运行时类型断言静默丢失注册的路径；实例的模块依赖由 ioc 装配阶段统一注入
func registerEvaluator(e service.AlertEvaluator) {
	ioc.Register(e)
	evaluatorRegistry = append(evaluatorRegistry, e)
}

// GetRegisteredEvaluators 返回所有已注册的评估器实例（IoC 装配完成后即为依赖注入完成态）
func GetRegisteredEvaluators() []service.AlertEvaluator {
	return evaluatorRegistry
}
