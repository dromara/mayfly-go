package dto

import "mayfly-go/pkg/utils/anyx"

// ************** 系统消息  **************

const SuccessSysMsgType = 1
const ErrorSysMsgType = 0
const InfoSysMsgType = 2

// websocket消息
type SysMsg struct {
	Type     int    `json:"type"`     // 消息类型
	Category string `json:"category"` // 消息类别
	Title    string `json:"title"`    // 消息标题
	Msg      string `json:"msg"`      // 消息内容

	ClientId string
}

func (sm *SysMsg) WithTitle(title string) *SysMsg {
	sm.Title = title
	return sm
}

func (sm *SysMsg) WithCategory(category string) *SysMsg {
	sm.Category = category
	return sm
}

func (sm *SysMsg) WithMsg(msg any) *SysMsg {
	sm.Msg = anyx.ToString(msg)
	return sm
}

func (sm *SysMsg) WithClientId(clientId string) *SysMsg {
	sm.ClientId = clientId
	return sm
}

// 普通消息
func InfoSysMsg(title string, msg any) *SysMsg {
	return &SysMsg{Type: InfoSysMsgType, Title: title, Msg: anyx.ToString(msg)}
}

// 成功消息
func SuccessSysMsg(title string, msg any) *SysMsg {
	return &SysMsg{Type: SuccessSysMsgType, Title: title, Msg: anyx.ToString(msg)}
}

// 错误消息
func ErrSysMsg(title string, msg any) *SysMsg {
	return &SysMsg{Type: ErrorSysMsgType, Title: title, Msg: anyx.ToString(msg)}
}
