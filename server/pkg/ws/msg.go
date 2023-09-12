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
	ToUserId UserId
	Data     any

	Type MsgType // 消息类型
}

// ************** 系统消息  **************

const SuccessSysMsgType = 1
const ErrorSysMsgType = 0
const InfoSysMsgType = 2

// websocket消息
type SysMsg struct {
	Type   int    `json:"type"`  // 消息类型
	Title  string `json:"title"` // 消息标题
	SysMsg string `json:"msg"`   // 消息内容
}

// 普通消息
func NewSysMsg(title, msg string) *SysMsg {
	return &SysMsg{Type: InfoSysMsgType, Title: title, SysMsg: msg}
}

// 成功消息
func SuccessSysMsg(title, msg string) *SysMsg {
	return &SysMsg{Type: SuccessSysMsgType, Title: title, SysMsg: msg}
}

// 错误消息
func ErrSysMsg(title, msg string) *SysMsg {
	return &SysMsg{Type: ErrorSysMsgType, Title: title, SysMsg: msg}
}
