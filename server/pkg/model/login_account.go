package model

type LoginAccount struct {
	Id       uint64
	Username string

	// ClientUuid 客户端UUID
	ClientUuid string
}
