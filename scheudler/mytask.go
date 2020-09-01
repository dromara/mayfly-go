package scheduler

import (
	"github.com/siddontang/go/log"
	"mayfly-go/base"
	"mayfly-go/base/utils"
	"mayfly-go/machine"
	"mayfly-go/models"
)

func init() {
	SaveMachineMonitor()
}

func SaveMachineMonitor() {
	AddFun("@every 60s", func() {
		for _, m := range *models.GetNeedMonitorMachine() {
			m := m
			go func() {
				mm := machine.GetMonitorInfo(machine.GetCli(uint64(utils.GetInt4Map(m, "id"))))
				if mm != nil {
					err := base.Insert(mm)
					if err != nil {
						log.Error("保存机器监控信息失败: %s", err.Error())
					}
				}
			}()
		}
	})
}
