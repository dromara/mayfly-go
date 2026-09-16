package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newAlertRuleRepo())
	ioc.Register(newAlertEventRepo())
	ioc.Register(newAlertSilenceRepo())
	ioc.Register(newAlertEscalationRepo())
	ioc.Register(newAlertInhibitionRepo())
	ioc.Register(newAlertNotifyPolicyRepo())
	ioc.Register(newAlertNotifyLogRepo())
}
