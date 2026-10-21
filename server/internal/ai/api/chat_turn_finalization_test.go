package api

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"mayfly-go/internal/ai/protocol"
	"mayfly-go/internal/ai/tools"

	"github.com/stretchr/testify/require"
)

// turn 收尾决策决定用户在会话里看到什么：红色故障条、「已停止」提示、还是只有那条收尾气泡。
// 它的输入只是一个 error，因此可以（也必须）脱离 WS 连接与 LLM 单测——此前它内联在 handler 里不可测，
// 「用户自己点取消却被显示成系统故障」这条残留只能靠真机点出来。

// nodeRunError 复现真机实测到的错误原文形状：外层是 eino 的节点运行错误包装，内层是工具层前缀文案
func nodeRunError(inner string) error {
	return fmt.Errorf("[NodeRunError] transfer node failed, node path: [ChatModel ToolNode]: %s", inner)
}

func TestDecideTurnFinalization(t *testing.T) {
	for _, tt := range []struct {
		name        string
		err         error
		wantStatus  string
		wantPublish bool
	}{
		{
			name:        "正常结束",
			err:         nil,
			wantStatus:  protocol.TurnStatusSuccess,
			wantPublish: false,
		},
		{
			name:        "显式停止按 stopped 收尾且不下发错误",
			err:         context.Canceled,
			wantStatus:  protocol.TurnStatusStopped,
			wantPublish: false,
		},
		{
			name:        "被包装的停止信号仍识别为 stopped",
			err:         fmt.Errorf("run turn failed: %w", context.Canceled),
			wantStatus:  protocol.TurnStatusStopped,
			wantPublish: false,
		},
		{
			// e6 的根因断言：用户点「取消」不是系统故障。一旦这条被并回 default 分支，
			// 前端就会在取消瞬间多亮一行红色提示，而且原文是内部节点结构
			name:        "用户取消参数补全按 cancelled 收尾，绝不下发错误事件",
			err:         nodeRunError(tools.CancelledToolPrefix + " The user has cancelled the parameter completion for this tool."),
			wantStatus:  protocol.TurnStatusCancelled,
			wantPublish: false,
		},
		{
			name:        "用户拒绝审批同样不下发错误事件",
			err:         nodeRunError(tools.RejectedToolPrefix + " the user rejected this operation"),
			wantStatus:  protocol.TurnStatusCancelled,
			wantPublish: false,
		},
		{
			name:        "真实故障才下发错误事件",
			err:         errors.New("dial tcp 10.0.0.1:22: connect: connection refused"),
			wantStatus:  protocol.TurnStatusFailed,
			wantPublish: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := decideTurnFinalization(tt.err)
			require.Equal(t, tt.wantStatus, got.status, "turn 终态")
			require.Equal(t, tt.wantPublish, got.publishError, "是否下发 error 事件（前端红色提示的唯一触发源）")
		})
	}
}
