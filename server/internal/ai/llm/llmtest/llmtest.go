// Package llmtest 提供 llm.Provider / ChatModel 的测试替身
//
// 多个包（session 摘要、memory 记忆提取、agent 上下文）都需要验证「按当前生效模型
// 取值」这一契约，替身集中放这里，避免各包重复实现并各自漂移。
package llmtest

import (
	"context"
	"errors"
	"sync/atomic"

	"mayfly-go/internal/ai/llm"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var errStreamNotImplemented = errors.New("llmtest: Stream not implemented")

// Provider 可切换的 Provider 替身：运行期替换 Model / Window 即可模拟配置变更
type Provider struct {
	Model  model.AgenticModel // 当前模型实例（nil 时按未配置处理）
	Window int                // 当前上下文窗口
	Err    error              // 非 nil 时 ChatModel 返回该错误

	calls int32
}

var _ llm.Provider = (*Provider)(nil)

func (p *Provider) ChatModel(context.Context) (model.AgenticModel, error) {
	atomic.AddInt32(&p.calls, 1)
	if p.Err != nil {
		return nil, p.Err
	}
	if p.Model == nil {
		return nil, errors.New("llmtest: model not configured")
	}
	return p.Model, nil
}

func (p *Provider) ContextWindow() int { return p.Window }

// ChatModelCalls 返回 ChatModel 被请求的次数（用于断言「每次使用都重新取模型」）
func (p *Provider) ChatModelCalls() int { return int(atomic.LoadInt32(&p.calls)) }

// Model 可编程 ChatModel 替身：Generate 返回固定文本，Stream 未实现
type Model struct {
	Reply string // Generate 返回的助手文本
	Err   error  // 非 nil 时 Generate 返回该错误

	calls int32
}

var _ model.AgenticModel = (*Model)(nil)

func (m *Model) Generate(_ context.Context, _ []*schema.AgenticMessage, _ ...model.Option) (*schema.AgenticMessage, error) {
	atomic.AddInt32(&m.calls, 1)
	if m.Err != nil {
		return nil, m.Err
	}
	return &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			{Type: schema.ContentBlockTypeAssistantGenText, AssistantGenText: &schema.AssistantGenText{Text: m.Reply}},
		},
	}, nil
}

func (m *Model) Stream(context.Context, []*schema.AgenticMessage, ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	return nil, errStreamNotImplemented
}

// Calls 返回 Generate 被调用次数
func (m *Model) Calls() int { return int(atomic.LoadInt32(&m.calls)) }
