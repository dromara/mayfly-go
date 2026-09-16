package application

import (
	"mayfly-go/internal/alert/application/evaluator"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(alertRuleAppImpl))
	ioc.Register(new(alertEventAppImpl))
	ioc.Register(new(alertEngineAppImpl))
	ioc.Register(new(alertSilenceAppImpl))
	ioc.Register(new(alertEscalationAppImpl))
	ioc.Register(new(alertInhibitionAppImpl))
	ioc.Register(new(alertNotifyPolicyAppImpl))
	ioc.Register(new(alertOverviewAppImpl))
	// 评估器通过 evaluator 包的 init() 自注册到 IoC
}

func GetAlertRuleApp() AlertRule {
	return ioc.Get[AlertRule]()
}

func GetAlertEventApp() AlertEvent {
	return ioc.Get[AlertEvent]()
}

func GetAlertEngine() AlertEngine {
	return ioc.Get[AlertEngine]()
}

func GetAlertSilenceApp() AlertSilence {
	return ioc.Get[AlertSilence]()
}

func GetAlertEscalationApp() AlertEscalation {
	return ioc.Get[AlertEscalation]()
}

func GetAlertInhibitionApp() AlertInhibition {
	return ioc.Get[AlertInhibition]()
}

func GetAlertNotifyPolicyApp() AlertNotifyPolicy {
	return ioc.Get[AlertNotifyPolicy]()
}

func GetAlertOverviewApp() AlertOverview {
	return ioc.Get[AlertOverview]()
}

// RegisterEvaluators 将所有已注册的评估器注册到评估器注册表
// 新增评估器只需在 evaluator 包中添加实现并调用 registerEvaluatorFactory，无需修改此函数
func RegisterEvaluators() {
	for _, e := range evaluator.GetAllEvaluators() {
		if evaluator, ok := e.(service.AlertEvaluator); ok {
			service.RegisterEvaluator(evaluator)
		}
	}
}
