package models

import (
	"github.com/astaxie/beego/orm"
	"time"
)

type MachineMonitor struct {
	Id         uint64    `orm:"column(id)" json:"id"`
	MachineId  uint64    `orm:"column(machine_id)" json:"machineId"`
	CpuRate    float32   `orm:"column(cpu_rate)" json:"cpuRate"`
	MemRate    float32   `orm:"column(mem_rate)" json:"memRate"`
	SysLoad    string    `orm:"column(sys_load)" json:"sysLoad"`
	CreateTime time.Time `orm:"column(create_time)" json:"createTime"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(MachineMonitor))
}
