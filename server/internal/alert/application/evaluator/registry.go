package evaluator

import "mayfly-go/pkg/ioc"

// evaluatorRegistry 评估器类型注册表 —— 新增评估器时只需在对应文件的 init() 中追加一行
var evaluatorRegistry []any

// registerEvaluatorFactory 注册评估器的 IoC 类型（泛型实例）
func registerEvaluatorFactory[T any]() {
	instance := new(T)
	ioc.Register(instance)
	evaluatorRegistry = append(evaluatorRegistry, instance)
}

// GetAllEvaluators 从 IoC 中获取所有已注册的评估器实例
func GetAllEvaluators() []any {
	return evaluatorRegistry
}
