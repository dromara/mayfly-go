package models

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/controllers/vo"
)

type Machine struct {
	model.Model
	Name string `json:"name"`
	// IP地址
	Ip string `json:"ip"`
	// 用户名
	Username string `json:"username"`
	Password string `json:"-"`
	// 端口号
	Port int `json:"port"`
}

func GetMachineById(id uint64) *Machine {
	machine := new(Machine)
	machine.Id = id
	err := model.GetBy(machine)
	if err != nil {
		return nil
	}
	return machine
}

// 分页获取机器信息列表
func GetMachineList(pageParam *model.PageParam) model.PageResult {
	return model.GetPage(pageParam, new(Machine), new([]vo.MachineVO), "Id desc")
}

// 获取所有需要监控的机器信息列表
func GetNeedMonitorMachine() []map[string]interface{} {
	return model.GetListBySql("SELECT id FROM t_machine WHERE need_monitor = 1")
}
