package application

import (
	"mayfly-go/internal/alert/application/evaluator"
	"mayfly-go/internal/alert/domain/service"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/logx"
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
	ioc.Register(new(machineTriageAppImpl))
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

func GetMachineTriage() MachineTriage {
	return ioc.Get[MachineTriage]()
}

// RegisterEvaluators 将所有已注册的评估器注册到评估器注册表，并对其指标元信息做启动自检。
// 新增评估器只需在 evaluator 包中添加实现并调用 registerEvaluator，无需修改此函数。
// 元信息缺陷（key 重复/展示文案缺失/阈值区间非法）会让保存校验、前端指标下拉、通知渲染
// 全线失真，因此 fail-fast 拒绝启动，而不是带缺陷运行
func RegisterEvaluators() {
	for _, e := range evaluator.GetRegisteredEvaluators() {
		if err := service.ValidateMetrics(e.Metrics()); err != nil {
			logx.Panicf("[alert] evaluator resourceType[%d] metrics invalid: %s", e.ResourceType(), err.Error())
		}
		service.RegisterEvaluator(e)
	}
}
