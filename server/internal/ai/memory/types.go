package memory

import "mayfly-go/internal/ai/session"

// ExtractMemoryReq 提取记忆的请求参数
type ExtractMemoryReq struct {
	UserId string             // 用户ID
	Msgs   []*session.Message // 消息列表
}
