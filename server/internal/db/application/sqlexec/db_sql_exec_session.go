package sqlexec

// 进行中的 SQL 执行登记表：支撑「服务端主动取消」。
//
// 前端 abort 只能断开 HTTP，数据库侧是否停下来取决于驱动是否响应 ctx 取消，
// 且平台完全看不到「谁在跑什么、跑了多久」。这里把每次交互式执行登记进来，
// 由 execId（前端本来就在生成并随请求带上，此前只进了日志）提供一条服务端中断入口。
//
// 为什么只中断自己的执行、不去 KILL 数据库会话：
// 连接池按语句借还连接，一次执行里的多条语句可能落在不同物理连接上，
// 执行侧拿不到稳定唯一的 backend 会话 id；若要 KILL 只能按「用户 + 库 + 语句文本」去猜匹配，
// 那会误杀同一账号并发跑的其他语句。停掉别人的会话属另一条通道：
// 活动会话列表 → 按会话 id 取消（见 api/db_session.go，权限码 db:session:cancel）

import (
	"context"
	"sync"
	"time"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/imsg"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
)

// runningSqlExec 一次进行中的执行
type runningSqlExec struct {
	cancel    context.CancelFunc
	execId    string
	dbId      uint64
	dbName    string
	accountId uint64
	sql       string
	startedAt time.Time
}

// sqlExecRegistry 进行中的执行集合。
//
// 只存内存：登记表的价值是「本实例正在跑什么」，跨实例取消另走会话通道；
// 进程重启后残留条目随之消失，因此不做持久化（重启即作废，不做恢复）
type sqlExecRegistry struct {
	mu      sync.Mutex
	running map[string]*runningSqlExec
}

var execRegistry = &sqlExecRegistry{running: make(map[string]*runningSqlExec)}

// register 登记一次执行并返回其可取消上下文与结束函数。
//
// execId 为空（非交互式入口，如工单回放、导入）时不登记，直接透传原 ctx，
// 避免把无法取消的执行塞进表里占位
func (r *sqlExecRegistry) register(ctx context.Context, execId string, req *dto.DbSQLExecReq) (context.Context, func()) {
	if execId == "" {
		return ctx, func() {}
	}
	runCtx, cancel := context.WithCancel(ctx)
	account := contextx.GetLoginAccount(ctx)
	var accountId uint64
	if account != nil {
		accountId = account.Id
	}

	entry := &runningSqlExec{
		cancel:    cancel,
		execId:    execId,
		dbId:      req.DbId,
		dbName:    req.Db,
		accountId: accountId,
		sql:       req.SQL,
		startedAt: time.Now(),
	}

	r.mu.Lock()
	// 同一 execId 重复登记说明前端复用了旧 id（如页面未刷新直接重跑）：丢弃旧条目并先取消它，
	// 否则旧执行失去被取消的能力，且结束时的注销会把新条目误删
	if prev, ok := r.running[execId]; ok {
		prev.cancel()
	}
	r.running[execId] = entry
	r.mu.Unlock()

	return runCtx, func() {
		cancel()
		r.mu.Lock()
		// 只删自己：重复登记场景下 map 里可能已被更新的条目覆盖，无条件 delete 会误删新条目
		if cur, ok := r.running[execId]; ok && cur == entry {
			delete(r.running, execId)
		}
		r.mu.Unlock()
	}
}

// cancelRegisterGrace 「取消」比登记先到时的有界等待。
//
// execId 随执行请求进来，但登记发生在取到连接之后：用户在执行刚发起的一百毫秒内点「取消」，
// 取消请求会先落地。不补这个窗口时，界面报「该执行已结束或不存在」而那条语句其实还在跑，
// 用户只能等它跑完或去会话列表里 KILL。上限取得很小：真正在跑的执行早已登记完成，
// 只有这种「几乎同时」的竞态需要等，拿不到就如实报错，不假装取消成功
const cancelRegisterGrace = time.Second

// cancel 按 execId 中断执行，accountId 为发起者：非属主不得取消他人执行。
//
// 找不到即视为已结束（不报错区分「已结束」与「从未存在」，两者对调用方的处置一致；
// 但也不允许借此静默成功——返回未找到的错误，让界面能如实反馈）
func (r *sqlExecRegistry) cancel(ctx context.Context, execId string, accountId uint64) error {
	entry, ok := r.waitEntry(ctx, execId)
	if !ok {
		return errorx.NewBizI(ctx, imsg.ErrSqlExecNotFoundToCancel)
	}
	// 无主执行（登记时拿不到登录账号）任何人都不能取消，避免异常登记变成免鉴权入口
	if entry.accountId == 0 || entry.accountId != accountId {
		return errorx.NewBizI(ctx, imsg.ErrSqlExecNotOwned)
	}
	entry.cancel()
	return nil
}

// waitEntry 取执行条目，未登记时最多等 cancelRegisterGrace，期间请求被取消即立刻放弃
func (r *sqlExecRegistry) waitEntry(ctx context.Context, execId string) (*runningSqlExec, bool) {
	deadline := time.After(cancelRegisterGrace)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		r.mu.Lock()
		entry, ok := r.running[execId]
		r.mu.Unlock()
		if ok {
			return entry, true
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-deadline:
			return nil, false
		case <-ticker.C:
		}
	}
}
