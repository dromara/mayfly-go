package imsg

import "mayfly-go/pkg/i18n"

var Zh_CN = map[i18n.MsgId]string{
	LogProcdefSave:   "流程定义-保存",
	LogProcdefDelete: "流程定义-删除",

	ErrProcdefKeyExist:        "该流程实例key已存在",
	ErrExistProcinstRunning:   "存在运行中的流程实例，无法操作",
	ErrExistProcinstSuspended: "存在挂起中的流程实例，无法操作",

	// procinst
	LogProcinstStart:  "流程-启动",
	LogProcinstCancel: "流程-取消",
	LogCompleteTask:   "流程-任务完成",
	LogRejectTask:     "流程-任务拒绝",
	LogBackTask:       "流程-任务驳回",

	ErrProcdefNotEnable:   "该流程定义非启用状态",
	ErrProcinstCancelSelf: "只能取消自己发起的流程",
	ErrProcinstCancelled:  "流程已取消",
	ErrBizHandlerFail:     "业务处理失败",
}
