package ws

// 消息类型
type MsgType uint8

const (
	JsonMsg   MsgType = 1
	TextMsg   MsgType = 2
	BinaryMsg MsgType = 3
)

// 消息信息
type Msg struct {
	ToUserId   UserId // 用户id
	ToClientId string // 客户端id

	Type MsgType // 消息类型
	Data any
}
