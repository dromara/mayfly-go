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
	"mayfly-go/pkg/i18n"
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
	// recent 刚结算的执行（execId → 结算记录）：尾点取消（语句刚好跑完、响应还在路上）
	// 时条目已注销，不记这一笔就只能回答「不存在」，操作者分不清是「语句跑完了」还是「取消落空了」
	recent map[string]recentSettle
}

var execRegistry = &sqlExecRegistry{running: make(map[string]*runningSqlExec), recent: make(map[string]recentSettle)}

// recentSettle 一笔刚结算记录：结算时刻 + 是否因取消而结束。
//
// 「因取消而结束」涵盖前端 abort 断连传导、cancel-exec、执行超时三条路径（runCtx 被 cancel）。
// 区分它是必须的：同一取消动作里 abort 走已有连接、几乎必然先于 cancel-exec 新建连接落地，
// 执行会先被 abort 停掉并结算——此时到达的 cancel-exec 若答「刚刚结束」，
// 等于对一个已经取消成功的动作报错；只有自然跑完的结算才该答「刚刚结束，无需取消」
type recentSettle struct {
	at        time.Time
	cancelled bool
	// accountId 结算条目的属主：recent 分支必须与 running 分支同一属主口径，
	// 否则这条无功能权限码的接口会凭 execId 沦为跨账号信息 oracle
	accountId uint64
}

// recentFinishTTL 结算后的条目保留可辨识时长。
//
// 只需覆盖「界面还没收到响应/还没重绘完」的窗口（百毫秒量级），30s 已给到操作者
// 看清结果再补点的余量；过期条目在登记/取消路径上顺手修剪，不另起后台协程
const recentFinishTTL = 30 * time.Second

// CodeExecJustFinished 「执行刚结算、无需取消」的错误码。
//
// 给一个可判定的错误码而不是让前端匹配提示文案：这是一个事实而非失败，中心层要按码
// 压掉红色 toast（结果区的状态迁移本身就是反馈），靠字符串区分会在换语言或改文案时静默失效。
// 取值在 int16 范围内，且与既有码（200/400/500/501/502/4001/4002）都不冲突
const CodeExecJustFinished int16 = 4003

// register 登记一次执行并返回其可取消上下文与结束函数。
//
// execId 为空（非交互式入口，如工单回放、导入）时不登记，直接透传原 ctx，
// 避免把无法取消的执行塞进表里占位。
//
// 同一 execId 碰撞分两种：同属主是前端复用旧 id 重跑，顶替并先取消旧条目；
// 跨属主则拒绝——顶替会取消对方正在跑的执行，等于绕过 cancel 的属主校验，
// 变成任意持 token 账号伪造 execId 中断他人执行的后门
func (r *sqlExecRegistry) register(ctx context.Context, execId string, req *dto.DbSQLExecReq) (context.Context, func(), error) {
	if execId == "" {
		return ctx, func() {}, nil
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
	if prev, ok := r.running[execId]; ok {
		if prev.accountId != accountId {
			r.mu.Unlock()
			cancel()
			return nil, nil, errorx.NewBizI(ctx, imsg.ErrSqlExecIdConflict)
		}
		// 同属主重复登记（前端复用旧 id，如页面未刷新直接重跑）：丢弃旧条目并先取消它，
		// 否则旧执行失去被取消的能力，且结束时的注销会把新条目误删
		prev.cancel()
	}
	r.running[execId] = entry
	r.mu.Unlock()

	return runCtx, func() {
		// 先于 cancel() 判定结算原因：defer done() 会无条件调 cancel() 释放资源，
		// 若事后再看 runCtx.Err() 则恒非 nil，自然跑完也会被误记成「因取消而结算」
		settledByCancel := runCtx.Err() != nil
		cancel()
		r.mu.Lock()
		// 只删自己：重复登记场景下 map 里可能已被更新的条目覆盖，无条件 delete 会误删新条目
		if cur, ok := r.running[execId]; ok && cur == entry {
			delete(r.running, execId)
			// 记下结算时刻与结算原因：尾点取消据此如实回答，
			// 且「已被取消停掉」的结算之后再到的取消会答成功而非报错。
			// 结算含执行前失败（取连接/鉴权被拒）：对取消者而言该 execId 下确实不再有执行
			r.recent[execId] = recentSettle{at: time.Now(), cancelled: settledByCancel, accountId: entry.accountId}
		}
		r.pruneRecentLocked()
		r.mu.Unlock()
	}, nil
}

// cancelRegisterGrace 「取消」比登记先到时的有界等待。
//
// execId 随执行请求进来，登记发生在执行请求被服务端处理时：用户在执行刚发起的一瞬点「取消」，
// 取消请求可能先于执行请求落地（两条请求分属不同连接，排队/调度顺序无保证）。
// 不补这个窗口时，界面报「未找到进行中的该执行」而那条语句其实还在跑，
// 用户只能等它跑完或去会话列表里 KILL。上限取得很小：登记已前置到取连接之前（见 api/db.go ExecSQL），
// 真正在跑的执行几乎必然早已登记，只有这种「几乎同时」的竞态需要等，拿不到就如实报错，不假装取消成功
const cancelRegisterGrace = time.Second

// cancel 按 execId 中断执行，accountId 为发起者：非属主不得取消他人执行。
//
// 找不到时分三种事实分开回答：TTL 内因取消而结算的答成功（取消意图已达成，
// 停它的可能是同一动作里的 abort）；TTL 内自然跑完的是「刚刚结束，无需取消」；
// 其余（早已结束/其他实例/进程重启/从未存在）才报「未找到进行中的执行」。
// 后两者都不算取消成功，但也不允许静默成功，让界面能如实反馈
func (r *sqlExecRegistry) cancel(ctx context.Context, execId string, accountId uint64) error {
	entry, settle, lookup := r.waitEntry(ctx, execId)
	switch lookup {
	case execLookupJustFinished:
		// 与 running 分支同一属主口径：非属主（含无主结算）一律拒绝。
		// 少了这道校验，cancel-exec 会凭 execId 变成跨账号的三值信息 oracle——
		// 探知他人某执行是否结算过、是被取消还是自然跑完，而这条接口并未挂功能权限码
		if settle.accountId == 0 || settle.accountId != accountId {
			return errorx.NewBizI(ctx, imsg.ErrSqlExecNotOwned)
		}
		// 执行是被取消停掉的（典型：同一取消动作里 abort 断连先于本请求落地）：
		// 取消意图已达成，答成功，界面照旧报「已请求取消」
		if settle.cancelled {
			return nil
		}
		// 自然跑完的结算：带分流码返回，这是「事实」而非「失败」，前端按码压掉红色 toast
		// （见 CodeExecJustFinished）；与真失败共用 400 会让中心层对一个不是错误的事实向操作者喊话
		return errorx.NewBizCode(CodeExecJustFinished, i18n.TC(ctx, imsg.ErrSqlExecJustFinished))
	case execLookupMissing:
		return errorx.NewBizI(ctx, imsg.ErrSqlExecNotFoundToCancel)
	}
	// 无主执行（登记时拿不到登录账号）任何人都不能取消，避免异常登记变成免鉴权入口
	if entry.accountId == 0 || entry.accountId != accountId {
		return errorx.NewBizI(ctx, imsg.ErrSqlExecNotOwned)
	}
	entry.cancel()
	return nil
}

// execLookupState 取消侧对 execId 的查找结论
type execLookupState int

const (
	// execLookupMissing 有界等待后仍未找到。取零值：这是唯一不解引用 entry 的最保守回答，
	// 将来新增返回路径若漏赋 state，得到的是「查无此执行」而非对 nil entry 解引用崩溃
	execLookupMissing execLookupState = iota
	// execLookupJustFinished 执行在 TTL 内刚结算（如长语句尾点取消），无需再等
	execLookupJustFinished
	// execLookupRunning 命中进行中的条目（唯一会解引用 entry 的状态，故不取零值）
	execLookupRunning
)

// waitEntry 取执行条目，未登记时最多等 cancelRegisterGrace，期间请求被取消即立刻放弃。
//
// 每轮先查进行表再查刚结算表：同一 execId 被复用重跑时，新条目一旦登记必须优先命中，
// 不能拿旧条目的结算记录误报「刚刚结束」
func (r *sqlExecRegistry) waitEntry(ctx context.Context, execId string) (*runningSqlExec, recentSettle, execLookupState) {
	deadline := time.After(cancelRegisterGrace)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		r.mu.Lock()
		entry, ok := r.running[execId]
		settle, settled := r.recent[execId]
		settled = settled && !ok && time.Since(settle.at) < recentFinishTTL
		r.mu.Unlock()
		if ok {
			return entry, recentSettle{}, execLookupRunning
		}
		if settled {
			return nil, settle, execLookupJustFinished
		}
		select {
		case <-ctx.Done():
			return nil, recentSettle{}, execLookupMissing
		case <-deadline:
			return nil, recentSettle{}, execLookupMissing
		case <-ticker.C:
		}
	}
}

// pruneRecentLocked 清理过期的刚结算条目，调用方须持锁：
// 每次结算都全量扫一遍，表里最多只会留下 TTL 窗口内的条目，
// 不需要另起后台协程也不会膨胀
func (r *sqlExecRegistry) pruneRecentLocked() {
	for id, s := range r.recent {
		if time.Since(s.at) >= recentFinishTTL {
			delete(r.recent, id)
		}
	}
}
