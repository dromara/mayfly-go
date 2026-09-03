package protocol

// ContentSegmentType 内容段类型
const (
	ContentSegmentInputText  = "input_text"
	ContentSegmentOutputText = "output_text"
	// 芯片引用段（对齐 tokhub typed ContentSegment：skill/mention 结构化贯穿
	// 发送/持久化/回显，前端按 type 渲染芯片样式而非纯文本）
	ContentSegmentSkill    = "skill"
	ContentSegmentResource = "resource"
)

// ContentSegment 类型化内容段
type ContentSegment struct {
	Type string `json:"type"`
	Text string `json:"text"`
	// Extra 芯片段元数据（resource: resourceType/id/code/ip/port/authCertName/username；
	// skill: id），随消息持久化使历史回显可恢复芯片样式
	Extra map[string]any `json:"extra,omitempty"`
}

// NewInputTextSegment 创建输入文本段
func NewInputTextSegment(text string) ContentSegment {
	return ContentSegment{Type: ContentSegmentInputText, Text: text}
}

// NewOutputTextSegment 创建输出文本段
func NewOutputTextSegment(text string) ContentSegment {
	return ContentSegment{Type: ContentSegmentOutputText, Text: text}
}
