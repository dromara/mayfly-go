package agent

import (
	"mayfly-go/pkg/utils/collx"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// AgenticMessage 路径（eino v0.9 Typed API）泛型实例化别名收敛：
// agent 包内部统一引用以下别名，消息类型切换只需修改此处。
// contributor.AgentMiddleware 为跨包共享的中间件契约别名。

type (
	// agenticAgent Typed Agent（AgenticMessage 实例化）
	agenticAgent = adk.TypedAgent[*schema.AgenticMessage]
	// agenticRunner Typed Runner（AgenticMessage 实例化）
	agenticRunner = adk.TypedRunner[*schema.AgenticMessage]
)

// AgentEvent Agent 运行事件（AgenticMessage 实例化，导出别名供外部回调签名引用）
type AgentEvent = adk.TypedAgentEvent[*schema.AgenticMessage]

// agentEvent 包内部使用的非导出引用
//
//nolint:unused // 别名本身无开销，统一收敛引用点
type agentEvent = AgentEvent

type InternalMessageType string

const (
	InternalMessageTypeResume string = "resume" // 中断恢复
)

// InternalMessageExtra 内部消息内容
type InternalMessageExtra struct {
	Type    string `json:"type"`
	Content any    `json:"content"`
}

func NewInternalMessageExtra(t string, content any) collx.M {
	return collx.Kvs("type", t, "content", content)
}
