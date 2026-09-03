package protocol

// InterruptEvent 中断事件
type InterruptEvent struct {
	ActionId    string         `json:"actionId"`
	Type        string         `json:"type"` // interrupt_approval | interrupt_param_completion 等
	Description string         `json:"description"`
	ToolName    string         `json:"toolName,omitempty"`
	ToolCallId  string         `json:"toolCallId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
