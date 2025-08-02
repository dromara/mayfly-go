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
	MsgSubtypeSqlScriptRunFail    MsgSubtype = "db.sqlscript.run.fail"
	MsgSubtypeSqlScriptRunSuccess MsgSubtype = "db.sqlscript.run.success"

	// flow
	MsgSubtypeFlowUserTaskTodo MsgSubtype = "flow.usertask.todo" // 用户任务待办
)

type MsgStatus int8

const (
	MsgStatusRead   MsgStatus = 1  // 已读
	MsgStatusUnRead MsgStatus = -1 // 未读
)
