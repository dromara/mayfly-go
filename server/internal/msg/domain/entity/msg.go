package entity

import (
	"mayfly-go/pkg/model"
)

type Msg struct {
	model.ExtraData
	model.CreateModel

	Type        MsgType    `json:"type"`                    // 消息类型
	Subtype     MsgSubtype `json:"subtype" gorm:"size:100"` // 消息子类型
	Status      MsgStatus  `json:"status"`
	Msg         string     `json:"msg" gorm:"size:2000"`
	RecipientId int64      `json:"recipientId"` // 接收人id，-1为所有接收
}

func (a *Msg) TableName() string {
	return "t_sys_msg"
}

type MsgType int8

const (
	MsgTypeNotify MsgType = 1 // 通知
	MsgTypeTodo   MsgType = 2 // 代办
)

type MsgSubtype string

const (
	// sys
	MsgSubtypeUserLogin MsgSubtype = "user.login"

	// machine
	MsgSubtypeMachineFileUploadSuccess MsgSubtype = "machine.file.upload.success"
	MsgSubtypeMachineFileUploadFail    MsgSubtype = "machine.file.upload.fail"

	// db
	MsgSubtypeDbDumpFail          MsgSubtype = "db.dump.fail"
	MsgSubtypeSQLScriptRunFail    MsgSubtype = "db.sqlscript.run.fail"
	MsgSubtypeSQLScriptRunSuccess MsgSubtype = "db.sqlscript.run.success"
	// 数据库任务（迁移/同步/备份导出）执行失败：定时任务失败若无人知晓，等于功能静默失效
	MsgSubtypeDbTaskRunFail MsgSubtype = "db.task.run.fail"
	// 表结构漂移：基线建立后库结构被人改过，不报就是「上线时一致、半年后发现早已走形」
	MsgSubtypeDbSchemaDrift MsgSubtype = "db.schema.drift"

	// flow
	MsgSubtypeFlowUserTaskTodo MsgSubtype = "flow.usertask.todo" // 用户任务待办
)

type MsgStatus int8

const (
	MsgStatusRead   MsgStatus = 1  // 已读
	MsgStatusUnRead MsgStatus = -1 // 未读
)
