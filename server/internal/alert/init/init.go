package init

import (
	"mayfly-go/internal/alert/api"
	"mayfly-go/internal/alert/application"
	"mayfly-go/internal/alert/infra/cache"
	"mayfly-go/internal/alert/infra/notifier"
	"mayfly-go/internal/alert/infra/persistence"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
		ioc.Register(notifier.NewAlertNotifier())
		ioc.Register(cache.NewBreachTracker())
		ioc.Register(cache.NewConsecutiveCounter())
	})

	starter.AddInitFunc(Init)
}

func Init() {
	application.RegisterEvaluators()
	application.GetAlertEngine().StartEvalLoop()
	application.GetAlertEscalationApp().StartEscalationLoop()
}
