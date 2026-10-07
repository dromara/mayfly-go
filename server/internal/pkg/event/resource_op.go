package event

import (
	"context"

	"mayfly-go/pkg/global"
	"mayfly-go/pkg/logx"
)

// PublishResourceOp 记录一次资源操作事件（用于资源操作留痕与最近访问等订阅方）。
//
// 标签路径为空说明该资源没有归属任何标签节点，此时直接返回并留痕：
// 取 CodePath[0] 会让整个请求以越界 panic 收场，把一个可观测的数据问题变成 500
func PublishResourceOp(ctx context.Context, codePath []string) {
	if len(codePath) == 0 {
		logx.WarnfContext(ctx, "skip the resource operation event: the resource has no tag path to attribute the operation to")
		return
	}
	global.EventBus.Publish(ctx, EventTopicResourceOp, codePath[0])
}
