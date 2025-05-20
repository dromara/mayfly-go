package imsg

import "mayfly-go/pkg/i18n"

var En = map[i18n.MsgId]string{
	LogProcdefSave:   "ProcDef - Save",
	LogProcdefDelete: "ProcDef - Delete",

	ErrProcdefKeyExist:        "the process instance key already exists",
	ErrProcdefFlowNotExist:    "The process definition does not exist",
	ErrExistProcinstRunning:   "There is a running process instance that cannot be manipulated",
	ErrExistProcinstSuspended: "There is a pending process instance that cannot be manipulated",

	ErrUserTaskNodeCandidateNotEmpty: "The candidate of the user task node [{{.name}}] cannot be empty",

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
