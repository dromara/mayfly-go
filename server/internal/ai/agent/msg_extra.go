package agent

import (
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/utils/collx"
)

// GetTurnId 获取turn id
func GetTurnId(msg *session.Message) string {
	return collx.M(msg.Extra).GetStr("turnId")
}

// SetTurnId 设置tern id
func SetTurnId(msg *session.Message, turnId string) {
	SetMessageExtra(msg, "turnId", turnId)
}

func SetActionId(msg *session.Message, actionId string) {
	SetMessageExtra(msg, "actionId", actionId)
}

func GetActionId(msg *session.Message) string {
	return collx.M(msg.Extra).GetStr("actionId")
}

func SetToolStatus(msg *session.Message, status string) {
	SetMessageExtra(msg, "toolStatus", status)
}

// SetMessageExtra 设置message extra
func SetMessageExtra(msg *session.Message, key string, value any) {
	m := collx.M(msg.Extra)
	msg.Extra = *m.Set(key, value)
}
