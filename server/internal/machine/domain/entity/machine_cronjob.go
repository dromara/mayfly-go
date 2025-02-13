package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// 机器任务配置
type MachineCronJob struct {
	model.Model

	Name            string     `json:"name" form:"name" gorm:"not null;size:255;comment:名称"` // 名称
	Key             string     `json:"key" gorm:"not null;size:32;comment:key"`              // key
	Cron            string     `json:"cron" gorm:"not null;size:255;comment:cron表达式"`        // cron表达式
	Script          string     `json:"script" gorm:"type:text;comment:脚本内容"`                 // 任务内容
	Status          int        `json:"status" form:"status" gorm:"comment:状态"`               // 状态
	Remark          string     `json:"remark" gorm:"size:255;comment:备注"`                    // 备注
	LastExecTime    *time.Time `json:"lastExecTime" gorm:"comment:最后执行时间"`                   // 最后执行时间
	SaveExecResType int        `json:"saveExecResType" gorm:"comment:保存执行记录类型"`              // 记录执行结果类型
}

// MachineCronJobExec 机器任务执行记录
type MachineCronJobExec struct {
	model.DeletedModel

	CronJobId   uint64    `json:"cronJobId" form:"cronJobId" gorm:"not null;"`
	MachineCode string    `json:"machineCode" form:"machineCode" gorm:"size:50;"`
	Status      int       `json:"status" form:"status"`  // 执行状态
	Res         string    `json:"res" gorm:"size:4000;"` // 执行结果
	ExecTime    time.Time `json:"execTime"`
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
