package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(AlertRule))
	ioc.Register(new(AlertEvent))
	ioc.Register(new(AlertSilence))
	ioc.Register(new(AlertEscalation))
	ioc.Register(new(AlertInhibition))
	ioc.Register(new(AlertNotifyPolicy))
}
