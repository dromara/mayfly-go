package dto

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/imsg"
	"mayfly-go/internal/msg/msgx"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

type MsgTmplSendEvent struct {
	TmplChannel *MsgTmplChannel
	Params      map[string]any // 模板占位符参数
	ReceiverIds []uint64       // 接收人id
}

type MsgTmplChannel struct {
	Tmpl     *entity.MsgTmpl
	Channels []*entity.MsgChannel
}

var (
	MsgChannelSite = &entity.MsgChannel{
		Type:   msgx.ChannelTypeSiteMsg,
		Status: entity.ChannelStatusEnable,
	}

	MsgChannelWs = &entity.MsgChannel{
		Type:   msgx.ChannelTypeWs,
		Status: entity.ChannelStatusEnable,
	}
)

var (
	MsgTmplLogin = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeUserLogin,
		entity.MsgStatusRead,
		imsg.LoginMsg,
		MsgChannelSite)

	MsgTmplMachineFileUploadSuccess = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeMachineFileUploadSuccess,
		entity.MsgStatusRead,
		imsg.MachineFileUploadSuccessMsg,
		MsgChannelSite, MsgChannelWs)

	MsgTmplMachineFileUploadFail = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeMachineFileUploadFail,
		entity.MsgStatusRead,
		imsg.MachineFileUploadFailMsg,
		MsgChannelSite, MsgChannelWs)

	MsgTmplDbDumpFail = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeDbDumpFail,
		entity.MsgStatusRead,
		imsg.DbDumpFailMsg,
		MsgChannelSite, MsgChannelWs)

	// MsgTmplDbTaskRunFail 数据库任务（迁移/同步/备份导出）执行失败的通知通道。
	//
	// 这些任务多数由 cron 触发：失败若只写进运行日志，没有界面会打开时就永远无人知晓，
	// 定时同步停摆几天才发现。故与 dump/脚本执行失败同级，站内 + ws 双通道推送。
	MsgTmplDbTaskRunFail = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeDbTaskRunFail,
		entity.MsgStatusRead,
		imsg.DbTaskRunFailMsg,
		MsgChannelSite, MsgChannelWs)

	// MsgTmplDbSchemaDrift 表结构偏离基线的通知。
	//
	// 漂移是「没人告知的结构变更」：不推送到人，基线就只是个存着的哈希值，
	// 等到下次发布对不上才发现。与任务失败同级（站内 + ws）
	MsgTmplDbSchemaDrift = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeDbSchemaDrift,
		entity.MsgStatusRead,
		imsg.DbSchemaDriftMsg,
		MsgChannelSite, MsgChannelWs)

	MsgTmplSQLScriptRunFail = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeSQLScriptRunFail,
		entity.MsgStatusRead,
		imsg.SQLScriptRunFailMsg,
		MsgChannelSite, MsgChannelWs)

	MsgTmplSQLScriptRunSuccess = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeSQLScriptRunSuccess,
		entity.MsgStatusRead,
		imsg.SQLScriptRunSuccessMsg,
		MsgChannelSite, MsgChannelWs)

	MsgTmplSQLScriptRunProgress = &MsgTmplChannel{
		Tmpl: &entity.MsgTmpl{
			ExtraData: model.ExtraData{
				Extra: collx.M{
					"category": "sqlScriptRunProgress",
				},
			},
		},
		Channels: []*entity.MsgChannel{MsgChannelWs},
	}

	// 数据文件导入进度消息模板：与 SQL 脚本执行进度同构（仅 Ws 通道、无持久化文案），
	// 前端按 uploadId 维护一个进度通知并在 terminated 时关闭
	MsgTmplDataImportProgress = &MsgTmplChannel{
		Tmpl: &entity.MsgTmpl{
			ExtraData: model.ExtraData{
				Extra: collx.M{
					"category": "dataImportProgress",
				},
			},
		},
		Channels: []*entity.MsgChannel{MsgChannelWs},
	}

	// 机器文件上传进度消息模板
	MsgTmplMachineFileUploadProgress = &MsgTmplChannel{
		Tmpl: &entity.MsgTmpl{
			ExtraData: model.ExtraData{
				Extra: collx.M{
					"category": "machineFileUploadProgress",
				},
			},
		},
		Channels: []*entity.MsgChannel{MsgChannelWs},
	}

	// 机器文件夹上传进度消息模板
	MsgTmplMachineFolderUploadProgress = &MsgTmplChannel{
		Tmpl: &entity.MsgTmpl{
			ExtraData: model.ExtraData{
				Extra: collx.M{
					"category": "machineFolderUploadProgress",
				},
			},
		},
		Channels: []*entity.MsgChannel{MsgChannelWs},
	}

	MsgTmplFlowUserTaskTodo = newMsgTmpl(entity.MsgTypeNotify,
		entity.MsgSubtypeFlowUserTaskTodo,
		entity.MsgStatusUnRead,
		imsg.FlowUserTaskTodoMsg,
		MsgChannelSite)
)

func newMsgTmpl(mtype entity.MsgType, subtype entity.MsgSubtype, status entity.MsgStatus, msgId i18n.MsgId, channels ...*entity.MsgChannel) *MsgTmplChannel {
	msgTmpl := &entity.MsgTmpl{}
	msgTmpl.SetExtraValue("msgId", msgId)
	msgTmpl.SetExtraValue("subtype", subtype)
	msgTmpl.SetExtraValue("type", mtype)
	msgTmpl.SetExtraValue("status", status)
	return &MsgTmplChannel{
		Tmpl:     msgTmpl,
		Channels: channels,
	}
}

// DbTaskRunFailEvent 构造一条「数据库任务执行失败」通知事件。
//
// 收件人取任务创建者而非当前登录账号：cron 触发的执行上下文里没有登录用户，
// 且失败只应通知到关心它的人，不做全员广播
func DbTaskRunFailEvent(taskName, errText string, receiverId uint64) *MsgTmplSendEvent {
	return &MsgTmplSendEvent{
		TmplChannel: MsgTmplDbTaskRunFail,
		Params:      collx.M{"taskName": taskName, "error": errText},
		ReceiverIds: []uint64{receiverId},
	}
}

// DbSchemaDriftEvent 构造一条「表结构偏离基线」通知。
//
// 收件人取基线创建者：漂移是否要紧只有建基线的人知道，不做全员广播
func DbSchemaDriftEvent(dbId uint64, dbName, detail string, count int, receiverId uint64) *MsgTmplSendEvent {
	return &MsgTmplSendEvent{
		TmplChannel: MsgTmplDbSchemaDrift,
		Params:      collx.M{"dbId": dbId, "dbName": dbName, "detail": detail, "count": count},
		ReceiverIds: []uint64{receiverId},
	}
}
