package init

import "mayfly-go/internal/machine/application"

func Init() {
	application.GetMachineCronJobApp().InitCronJob()
}
