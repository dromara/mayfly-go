package dto

import "mayfly-go/pkg/utils/stringx"

// ************** 系统消息  **************

const SuccessSysMsgType = 1
const ErrorSysMsgType = 0
const InfoSysMsgType = 2

// websocket消息
type SysMsg struct {
	Type     int    `json:"type"`     // 消息类型
	Category int    `json:"category"` // 消息类别
	Title    string `json:"title"`    // 消息标题
	Msg      string `json:"msg"`      // 消息内容
}

func (sm *SysMsg) WithTitle(title string) *SysMsg {
	sm.Title = title
	return sm
}

func (sm *SysMsg) WithCategory(category int) *SysMsg {
	sm.Category = category
	return sm
}

func (sm *SysMsg) WithMsg(msg any) *SysMsg {
	sm.Msg = stringx.AnyToStr(msg)
	return sm
}

// 普通消息
func NewSysMsg(title string, msg any) *SysMsg {
	return &SysMsg{Type: InfoSysMsgType, Title: title, Msg: stringx.AnyToStr(msg)}
}

// 成功消息
func SuccessSysMsg(title string, msg any) *SysMsg {
	return &SysMsg{Type: SuccessSysMsgType, Title: title, Msg: stringx.AnyToStr(msg)}
}

// 错误消息
func ErrSysMsg(title string, msg any) *SysMsg {
	return &SysMsg{Type: ErrorSysMsgType, Title: title, Msg: stringx.AnyToStr(msg)}
}
