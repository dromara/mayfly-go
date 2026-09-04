package form

type ChatMsgType string

const (
	ChatMsgTypeText            ChatMsgType = "text"            // 文本消息
	ChatMsgTypeInterruptResume ChatMsgType = "interruptResume" // 恢复中断
	ChatMsgTypeStop            ChatMsgType = "stop"            // 显式停止运行中 turn（唯一真正中断 agent 的途径）
	ChatMsgTypeAttach          ChatMsgType = "attach"          // 订阅运行中 turn（刷新/断线重连后续流）
)

// ChatSegment 富文本段（芯片引用：资源/技能等），随文本消息一起发送
// 对齐前端 input/chipRegistry 的 ChipSegment 结构
type ChatSegment struct {
	Type  string         `json:"type"`  // 段类型：input_text / skill / resource
	Text  string         `json:"text"`  // 文本内容（芯片为展示标签）
	Extra map[string]any `json:"extra"` // 芯片附加数据（如 resource 的 id/type）
}

// ChatAttachment 用户消息附件元数据（内容已由前端在发送前经统一文件服务上传落
// local/S3，随消息发送的仅是轻量引用，经 TurnItem payload 持久化用于历史回显）
type ChatAttachment struct {
	Name    string `json:"name"`           // 文件名
	Kind    string `json:"kind"`           // 附件种类：image | text | file
	Mime    string `json:"mime,omitempty"` // MIME 类型
	Size    int64  `json:"size,omitempty"` // 字节大小
	FileKey string `json:"fileKey"`        // 文件服务 key（内容已落 t_sys_file）
}

// ChatRequest 客户端发送的聊天消息（EventMsg 协议）
// Content 按类型校验：text/interruptResume 必填，stop/attach 无需（api 层手动校验）
type ChatRequest struct {
	ConversationId uint64           `json:"conversationId"` // 会话 ID（0 或 -1 表示新建）
	Code           string           `json:"code"`           // 会话编码（可选，与 conversationId 二选一）
	Type           ChatMsgType      `json:"type"`
	Content        string           `json:"content"`
	Segments       []ChatSegment    `json:"segments"`              // 可选：富文本段（含资源/技能芯片引用），为空时纯文本发送
	Attachments    []ChatAttachment `json:"attachments,omitempty"` // 可选：附件元数据（fileKey 引用，展示/预览用，不参与 LLM 输入）
}

// CreateConversationRequest 创建会话请求
type CreateConversationRequest struct {
	Title string `json:"title"`
}

// RenameConversationRequest 重命名会话请求
type RenameConversationRequest struct {
	Id    uint64 `json:"id" binding:"required"`
	Title string `json:"title" binding:"required"`
}
