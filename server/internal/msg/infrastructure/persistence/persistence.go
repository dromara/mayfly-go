package persistence

import "mayfly-go/internal/msg/domain/repository"

var (
	msgRepo = newMsgRepo()
)

func GetMsgRepo() repository.Msg {
	return msgRepo
}
