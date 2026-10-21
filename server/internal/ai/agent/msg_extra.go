package agent

import (
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/utils/collx"
)

// GetTurnId 获取turn id
func GetTurnId(msg *session.Message) string {
	return collx.M(msg.Extra).GetStr(session.ExtraKeyTurnId)
}

// SetTurnId 设置tern id
func SetTurnId(msg *session.Message, turnId string) {
	SetMessageExtra(msg, session.ExtraKeyTurnId, turnId)
}

func SetActionId(msg *session.Message, actionId string) {
	SetMessageExtra(msg, session.ExtraKeyActionId, actionId)
}

func GetActionId(msg *session.Message) string {
	return collx.M(msg.Extra).GetStr(session.ExtraKeyActionId)
}

func SetToolStatus(msg *session.Message, status string) {
	SetMessageExtra(msg, session.ExtraKeyToolStatus, status)
}

// GetToolStatus 读取工具调用的状态标记（error / interrupted / 空），与 SetToolStatus 成对
func GetToolStatus(msg *session.Message) string {
	if msg == nil {
		return ""
	}
	return collx.M(msg.Extra).GetStr(session.ExtraKeyToolStatus)
}

// SetMessageExtra 设置message extra
func SetMessageExtra(msg *session.Message, key string, value any) {
	m := collx.M(msg.Extra)
	msg.Extra = *m.Set(key, value)
}
