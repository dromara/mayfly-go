package base

import (
	"context"
	"errors"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/gox"
	"sync"

	"mayfly-go/pkg/contextx"
)

// afterCommitKey 是待执行动作队列在 ctx 上的键。
//
// 用非导出结构体作键，避免与其他包的同类键相互覆盖
type afterCommitKey struct{}

// afterCommitQueue 收集「事务提交成功后才允许执行」的动作。
//
// 动作本身可能在提交后于其他协程里被读取，因此用互斥锁保护追加与取出的配对操作
type afterCommitQueue struct {
	mu      sync.Mutex
	actions []func(context.Context) error
}

func (q *afterCommitQueue) add(action func(context.Context) error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.actions = append(q.actions, action)
}

func (q *afterCommitQueue) drain() []func(context.Context) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	pending := q.actions
	q.actions = nil
	return pending
}

// AfterCommit 登记一个「仅当本事务提交成功后才执行」的动作。
//
// 存在这条入口的原因：外部副作用（对目标实例执行 SQL、下发机器/Redis 命令、调用模型、发通知）
// 既不可回滚，又会把本地事务的行锁持有时间拉长到秒级甚至分钟级。把它们写在事务里，会出现
// 「外部世界已经改了，而元数据库回滚/提交失败说没改」这种比重复执行更难查的半提交状态。
//
// 语义边界要说清楚：AfterCommit 只保证「不提前做」，不保证「一定做」——进程若在提交后、执行前
// 崩溃，动作就丢了。因此它适合丢了不致命的动作（通知、缓存刷新、连接清理）。凡「必须做且要
// 记账」的（审批通过后的业务回放），必须与状态位配合使用：在同一事务里把待处理状态一并落库，
// 使崩溃后的记录是「已完成但业务待处理」这种可被发现、可重驱动的状态，而不是无痕丢失
//
// ctx 未绑定事务时立即执行并返回其错误：此时没有需要等待的提交点
func AfterCommit(ctx context.Context, action func(ctx context.Context) error) error {
	if action == nil {
		return nil
	}
	if queue := afterCommitQueueFromCtx(ctx); queue != nil {
		queue.add(action)
		return nil
	}
	return action(ctx)
}

// withAfterCommitQueue 给 ctx 挂上待执行动作队列，供 Tx 在提交成功后取出执行
func withAfterCommitQueue(ctx context.Context) (context.Context, *afterCommitQueue) {
	queue := &afterCommitQueue{}
	return context.WithValue(ctx, afterCommitKey{}, queue), queue
}

func afterCommitQueueFromCtx(ctx context.Context) *afterCommitQueue {
	if queue, ok := ctx.Value(afterCommitKey{}).(*afterCommitQueue); ok {
		return queue
	}
	return nil
}

// runAfterCommit 出队并依次执行已登记的动作，由事务创建者在提交成功后调用。
//
// 传入的 ctx 可以是任意形态（事务创建者传的就是绑了事务的 txCtx），因为动作 ctx 由本函数
// 自己做两层剥离，不依赖调用点传对东西：
//   - 事务连接：Commit 只结束事务，不会清掉 Session 里那条 *gorm.DB，沿用它写入会撞
//     "sql: transaction has already been committed or rolled back"，因此用 gormx.WithoutTx 显式脱绑定；
//   - 已排空的队列：动作内部再登记「提交后动作」时，若仍带着原队列，会追加进一个不再被排空
//     的队列而被静默丢弃，因此把队列键也置空。
//
// 取消信号同样剥离：请求可能在动作执行期间返回，而副作用（尤其是对目标实例的写入）不能被
// 掐断成「执行了一半」；链路信息与登录账号则保留，否则这批写入既无法串联也查不到归属人。
//
// 所有动作都执行完、失败逐个收集后合并返回：动作之间互不相关，第一个失败不该剥夺后续动作的
// 执行机会（否则漏做的副作用只能靠日志追溯）
func runAfterCommit(baseCtx context.Context, queue *afterCommitQueue) error {
	actions := queue.drain()
	if len(actions) == 0 {
		return nil
	}
	runCtx := context.WithValue(contextx.Detach(gormx.WithoutTx(baseCtx)), afterCommitKey{}, nil)
	var errs []error
	for _, action := range actions {
		if err := runAction(runCtx, action); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// runAction 执行单个提交后动作，并把 panic 收敛成 error。
//
// 事务那一层的 recover 在这里已经用不上了：TxWith 里 panic 收口先于「结束事务」执行，而动作跑在
// 后者里。不接住 panic 的后果不是崩进程这么简单：它会从**已提交**的事务里穿出去，让调用方拿到
// 500 而误判整件事失败并重试，于是那些已经生效的副作用被再做一遍；排在后面的动作也会被无声
// 跳过。收敛成 error 才能走 ErrAfterCommit 这条「数据已生效，不许整体重试」的路
func runAction(ctx context.Context, action func(context.Context) error) (err error) {
	defer gox.Recover(func(e error) { err = e })
	return action(ctx)
}
