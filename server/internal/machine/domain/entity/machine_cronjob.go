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

	// 二期增强：超时/重试/结果通知
	TimeoutSeconds int    `json:"timeoutSeconds" gorm:"comment:单次执行超时秒数 0=用全局默认"`   // 命令执行超时
	RetryTimes     int8   `json:"retryTimes" gorm:"comment:失败重试次数(仅连接/超时重试)"`       // 失败重试次数
	NotifyType     int8   `json:"notifyType" gorm:"comment:结果通知 0不通知 1仅失败 2总是"`     // 通知方式
	NotifyTmplCode string `json:"notifyTmplCode" gorm:"size:64;comment:通知消息模板code"` // 绑定的消息模板编码（渠道由模板关联）
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

	// 结果通知方式
	CronJobNotifyNone   int8 = 0 // 不通知
	CronJobNotifyFail   int8 = 1 // 仅失败时通知
	CronJobNotifyAlways int8 = 2 // 总是通知
)
