package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// 机器任务配置
type MachineCronJob struct {
	model.Model

	Name            string     `json:"name" form:"name"`
	Key             string     `json:"key"`
	Cron            string     `json:"cron"`   // cron表达式
	Script          string     `json:"script"` // 任务内容
	Status          int        `json:"status" form:"status"`
	Remark          string     `json:"remark"` // 备注
	LastExecTime    *time.Time `json:"lastExecTime"`
	SaveExecResType int        `json:"saveExecResType"` // 记录执行结果类型
}

// 计划任务与机器关联信息
type MachineCronJobRelate struct {
	model.CreateModel

	CronJobId uint64
	MachineId uint64
}

// 机器任务执行记录
type MachineCronJobExec struct {
	model.DeletedModel

	CronJobId uint64    `json:"cronJobId" form:"cronJobId"`
	MachineId uint64    `json:"machineId" form:"machineId"`
	Status    int       `json:"status" form:"status"` // 执行状态
	Res       string    `json:"res"`                  // 执行结果
	ExecTime  time.Time `json:"execTime"`
}

const (
	MachineCronJobStatusEnable  = 1
	MachineCronJobStatusDisable = -1

	MachineCronJobExecStatusSuccess = 1
	MachineCronJobExecStatusError   = -1

	SaveExecResTypeNo      = -1 // 不记录执行日志
	SaveExecResTypeOnError = 1  // 执行错误时记录日志
	SaveExecResTypeYes     = 2  // 记录日志
)
