package scheduler

import (
	"github.com/robfig/cron/v3"
	"mayfly-go/base"
)

var c = cron.New()

func Start() {
	c.Start()
}

func Stop() {
	c.Stop()
}

func GetCron() *cron.Cron {
	return c
}

func AddFun(spec string, cmd func()) cron.EntryID {
	id, err := c.AddFunc(spec, cmd)
	if err != nil {
		panic(base.NewBizErr("添加任务失败：" + err.Error()))
	}
	return id
}
