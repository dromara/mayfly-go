package scheduler

import (
	"mayfly-go/pkg/biz"

	"github.com/robfig/cron/v3"
)

func init() {
	Start()
}

var cronService = cron.New()

func Start() {
	cronService.Start()
}

func Stop() {
	cronService.Stop()
}

func Remove(id cron.EntryID) {
	cronService.Remove(id)
}

func GetCron() *cron.Cron {
	return cronService
}

func AddFun(spec string, cmd func()) cron.EntryID {
	id, err := cronService.AddFunc(spec, cmd)
	biz.ErrIsNilAppendErr(err, "添加任务失败: %s")
	return id
}
