package dto

import "mayfly-go/pkg/utils/collx"

// UserTaskOp 用户任务操作
type UserTaskOp struct {
	TaskId    uint64   // 任务id
	Candidate []string // 任务处理候选人
	Remark    string   // 备注
	Vars      collx.M  // 任务变量
	Handler   string   // 处理人
}
