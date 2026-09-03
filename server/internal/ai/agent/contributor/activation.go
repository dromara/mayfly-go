package contributor

import (
	"context"

	"mayfly-go/pkg/logx"
)

// Activatable 可激活贡献者：装配期一次性激活
//
// 用于「无运行期调度通道、仅需在装配期装载副作用」的扩展，如中断类型扩展
// 装载、记忆提取装配。与旧的副作用装配（宿主手工调用 SetupXxx）相比：
//
//   - 统一纳入 Builder 装配链路，可经配置 disabledExtensions 按 Id 裁剪
//   - 激活发生在 WithFilter 裁剪之后，被裁剪的扩展不会激活
//   - 激活失败 fail-open（记日志不阻断），实现方须保证幂等（装配重试安全）
type Activatable interface {
	// Activate 装配期激活；实现方须幂等且不可假设其他贡献者已激活
	Activate(ctx context.Context) error
}

// Activate 激活全部装配期贡献者（按注册顺序遍历全部通道条目，fail-open）
//
// 由宿主在 WithFilter 裁剪后调用一次。返回首个激活错误（全部贡献者均会
// 尝试激活，错误仅用于宿主日志诊断，不阻断装配）。
func (r *Registry) Activate(ctx context.Context) error {
	if r == nil {
		return nil
	}
	var firstErr error
	for _, e := range r.entries {
		a, ok := e.c.(Activatable)
		if !ok {
			continue
		}
		if err := a.Activate(ctx); err != nil {
			logx.ErrorfContext(ctx, "[contributor] activate contributor %s error: %v", e.id, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		logx.InfofContext(ctx, "[contributor] contributor %s activated", e.id)
	}
	return firstErr
}
