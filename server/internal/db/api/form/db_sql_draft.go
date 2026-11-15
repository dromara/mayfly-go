package form

// SqlDraft SQL 草稿（自然语言转 SQL）请求。
//
// 只传「用户这次说了什么」与编辑器里的材料：库结构、权限过滤、审批事实全部由后端
// 自己取——前端传结构等于开了个「伪造上下文」的口子，也让权限判定无处安放。
type SqlDraft struct {
	// DbId 数据源 id
	DbId uint64 `json:"dbId" validate:"required"`
	// Db 库名（可空，空则用资产默认库）
	Db string `json:"db" `
	// Intent 意图：draft（按描述生成）/ fix（按描述改写选中 SQL）
	Intent string `json:"intent"`
	// Prompt 需求描述
	Prompt string `json:"prompt" validate:"required"`
	// Selection 编辑器选区原文
	Selection string `json:"selection"`
	// AnchorStatement 光标所在完整语句
	AnchorStatement string `json:"anchorStatement"`
	// RefineOf 上一版生成的语句（多轮细化时回传；「重新生成」不带它）
	RefineOf string `json:"refineOf"`
	// IncludePlans 是否取执行计划做只读校验
	IncludePlans bool `json:"includePlans"`
}
