package event

import (
	"context"
	"testing"
)

// 资源没有标签路径时不得越界 panic：那会把一个可观测的数据问题变成 500。
// EventBus 在此保持为 nil，顺带确认空路径分支根本不会去发布事件
func TestPublishResourceOpWithEmptyCodePath(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("an empty tag path must be skipped instead of panicking, recovered %v", recovered)
		}
	}()

	PublishResourceOp(context.Background(), nil)
	PublishResourceOp(context.Background(), []string{})
}
