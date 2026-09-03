package tools

import (
	"context"

	"mayfly-go/pkg/logx"
)

const (
	ToolStatusSuccess = "success"
	ToolStatusError   = "error"
	// 工具执行中间状态（中断）
	ToolStatusInterrupted = "interrupted" // 已中断，等待用户交互
)

// SafeOptionsFn 安全执行选项查询函数：捕获 panic 并记录日志，
// 避免选项构建失败（如依赖应用未初始化）导致中断流程中断
func SafeOptionsFn(ctx context.Context, name string, fn func() []CompletionOption) (options []CompletionOption) {
	defer func() {
		if r := recover(); r != nil {
			logx.WarnfContext(ctx, "[%s] panic: %v", name, r)
			options = nil
		}
	}()
	return fn()
}
