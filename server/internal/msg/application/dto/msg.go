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
