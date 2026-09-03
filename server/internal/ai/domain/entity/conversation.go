package entity

import (
	"mayfly-go/pkg/model"
)

// Conversation AI 会话实体（替代 Session）
type Conversation struct {
	model.Model
	model.ExtraData

	Code                   string `gorm:"column:code;size:64;not null;comment:会话业务编码（UUID）" json:"code"`
	Title                  string `gorm:"column:title;size:255;not null;default:New Chat;comment:会话标题" json:"title"`
	Status                 int    `gorm:"column:status;not null;default:1;comment:1=active 2=archived" json:"status"`
	Compaction             string `gorm:"column:compaction;type:text;comment:对话压缩结果" json:"compaction"`
	CompactionMessageCount int    `gorm:"column:compaction_message_count;not null;default:0;comment:压缩时消息数" json:"compactionMessageCount"`
	TotalMessageCount      int    `gorm:"column:total_message_count;not null;default:0;comment:总消息数量" json:"totalMessageCount"`
	PromptTokens           int64  `gorm:"column:prompt_tokens;not null;default:0;comment:输入token数" json:"promptTokens"`
	CompletionTokens       int64  `gorm:"column:completion_tokens;not null;default:0;comment:输出token数" json:"completionTokens"`
	TotalTokens            int64  `gorm:"column:total_tokens;not null;default:0;comment:总token数" json:"totalTokens"`
}

// Conversation 状态常量
const (
	ConversationStatusActive   = 1
	ConversationStatusArchived = 2
)

func (c *Conversation) TableName() string {
	return "t_ai_conversation"
}
