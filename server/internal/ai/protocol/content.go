package protocol

// ContentSegmentType 内容段类型
const (
	ContentSegmentInputText  = "input_text"
	ContentSegmentOutputText = "output_text"
	// 芯片引用段（typed ContentSegment：skill/mention 结构化贯穿
	// 发送/持久化/回显，前端按 type 渲染芯片样式而非纯文本）
	ContentSegmentSkill    = "skill"
	ContentSegmentResource = "resource"
	// 图片段（多模态输入：附件内容经统一文件服务落 local/S3，extra.fileKey 承载
	// 引用贯穿持久化/回显，展示端以 /sys/files/{fileKey} 访问；LLM 侧由
	// application 层解析为 base64 data URL 后转 UserInputImage block，
	// 文本侧仅保留占位说明）
	ContentSegmentImage = "image"
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

// ImageUrlsOf 提取内容段中的图片文件引用列表（image 段 → session.Message.ImageUrls）。
// 引用为文件服务 fileKey（extra.fileKey，附件内容已落 t_sys_file），
// 须由调用方经文件服务解析为 base64 data URL（见 application.ResolveImageUrls）
// 后再进 LLM 请求
func ImageUrlsOf(segments []ContentSegment) []string {
	urls := make([]string, 0)
	for _, seg := range segments {
		if seg.Type != ContentSegmentImage {
			continue
		}
		if key, _ := seg.Extra["fileKey"].(string); key != "" {
			urls = append(urls, key)
		}
	}
	return urls
}
