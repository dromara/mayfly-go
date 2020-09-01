package models

import (
	"github.com/astaxie/beego/orm"
	"mayfly-go/base"
	"mayfly-go/controllers/vo"
)

type Machine struct {
	base.Model
	Name string `orm:"column(name)"`
	// IP地址
	Ip string `orm:"column(ip)" json:"ip"`
	// 用户名
	Username string `orm:"column(username)" json:"username"`
	Password string `orm:"column(password)" json:"-"`
	// 端口号
	Port int `orm:"column(port)" json:"port"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(Machine))
}

func GetMachineById(id uint64) *Machine {
	machine := new(Machine)
	machine.Id = id
	err := base.GetBy(machine)
	if err != nil {
		return nil
	}
	return machine
}

// 分页获取机器信息列表
func GetMachineList(pageParam *base.PageParam) base.PageResult {
	m := new([]Machine)
	querySetter := base.QuerySetter(new(Machine)).OrderBy("-Id")
	return base.GetPage(querySetter, pageParam, m, new([]vo.MachineVO))
}

// 获取所有需要监控的机器信息列表
func GetNeedMonitorMachine() *[]orm.Params {
	return base.GetListBySql("SELECT id FROM t_machine WHERE need_monitor = 1")
}
