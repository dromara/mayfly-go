package entity

import (
	"mayfly-go/pkg/model"
)

type Db struct {
	model.Model

	Code           string `orm:"column(code)" json:"code"`
	Name           string `orm:"column(name)" json:"name"`
	Database       string `orm:"column(database)" json:"database"`
	Remark         string `json:"remark"`
	InstanceId     uint64
	FlowProcdefKey *string `json:"flowProcdefKey"` // 审批流-流程定义key（有值则说明关键操作需要进行审批执行）,使用指针为了方便更新空字符串(取消流程审批)
}
