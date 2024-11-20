package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogProcdefSave:   "ProcDef - Save",
	LogProcdefDelete: "ProcDef - Delete",

	ErrProcdefKeyExist:        "the process instance key already exists",
	ErrExistProcinstRunning:   "There is a running process instance that cannot be manipulated",
	ErrExistProcinstSuspended: "There is a pending process instance that cannot be manipulated",

	// procinst
	LogProcinstStart:  "Process - Start",
	LogProcinstCancel: "Process - Cancel",
	LogCompleteTask:   "Process - Completion of task",
	LogRejectTask:     "Process - Task rejection",
	LogBackTask:       "Process - Task rejection",

	ErrProcdefNotEnable:   "The process defines a non-enabled state",
	ErrProcinstCancelSelf: "You can only cancel processes you initiated",
	ErrProcinstCancelled:  "Process has been cancelled",
	ErrBizHandlerFail:     "Business process failure",
}
