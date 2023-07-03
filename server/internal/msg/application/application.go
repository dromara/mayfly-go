package application

import (
	"mayfly-go/internal/msg/infrastructure/persistence"
)

var (
	msgApp = newMsgApp(persistence.GetMsgRepo())
)

func GetMsgApp() Msg {
	return msgApp
}
