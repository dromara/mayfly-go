package sqlexec

// 进行中执行登记表的回归测试。
//
// 守的是「取消」这条通道的安全/正确性边界：
//  1. 只有发起者能取消自己的执行（否则任何持 token 的账号可中断他人查询）；
//  2. 登记时拿不到登录账号的执行谁都取消不了（异常登记不能变成免鉴权入口）；
//  3. 重复 execId 不会让旧条目失去可取消性，也不会让旧条目的收尾误删新条目；
//  4. 跨属主的 execId 碰撞必须被拒绝：登记的顶替行为不得绕过取消侧的属主校验；
//  5. 取消比结算晚到一步（长语句尾点取消）得到的是「刚刚结束」而非「查无此执行」；
//  6. 入口 handler 的登记必须早于取连接（冷连接拨号窗口会让抢跑取消落空）。
//
// 运行：cd server && go test -count=1 -race -run TestSqlExecRegistry ./internal/db/application/

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

// ctxWithAccount 用真实写入函数构造带登录账号的 ctx：
// 登记表的属主取自 contextx.GetLoginAccount，测试自造 key 会让 accountId 恒为 0，测不到属主分支
func ctxWithAccount(id uint64) context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: id, Username: "u"})
}

func register(ctx context.Context, execId string) (context.Context, func()) {
	runCtx, done, err := execRegistry.register(ctx, execId, &dto.DbSQLExecReq{ExecId: execId, DbId: 1, Db: "db1", SQL: "select 1"})
	if err != nil {
		// 同属主登记在测试里不存在碰撞面，报错即用例前提坏了，直接炸出来
		panic(err)
	}
	return runCtx, done
}

func TestSqlExecRegistryOwnerCheck(t *testing.T) {
	ctx, done := register(ctxWithAccount(101), "exec-owner")
	t.Cleanup(done)

	// 非属主取消必须失败，且不能真的把执行打断
	err := execRegistry.cancel(ctxWithAccount(202), "exec-owner", 202)
	require.Error(t, err)
	assert.NotPanics(t, func() {
		select {
		case <-ctx.Done():
			t.Fatal("非属主的取消请求不得中断他人执行")
		default:
		}
	})

	require.NoError(t, execRegistry.cancel(context.Background(), "exec-owner", 101))
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("属主取消后执行上下文必须已取消")
	}
}

// 登记时拿不到登录账号（如内部链路误带 execId）时，任何人都不能取消：
// 若按「accountId==0 即放行」处理，这条登记就成了免鉴权的中断入口
func TestSqlExecRegistryRejectsOwnerlessEntry(t *testing.T) {
	ctx, done, err := execRegistry.register(context.Background(), "exec-ownerless", &dto.DbSQLExecReq{ExecId: "exec-ownerless", DbId: 1, Db: "db1", SQL: "select 1"})
	require.NoError(t, err)
	t.Cleanup(done)

	require.Error(t, execRegistry.cancel(context.Background(), "exec-ownerless", 7))
	select {
	case <-ctx.Done():
		t.Fatal("无主执行不应被取消")
	default:
	}
}

func TestSqlExecRegistryUnknownAndEmptyId(t *testing.T) {
	// 空 execId 不登记：非交互入口（工单回放/导入）没有可取消语义，不占表位
	_, done := register(ctxWithAccount(1), "")
	done()
	require.Error(t, execRegistry.cancel(context.Background(), "", 1))
	require.Error(t, execRegistry.cancel(context.Background(), "never-registered", 1))
}

// 同一 execId 重复登记（前端复用旧 id 直接重跑）：旧条目必须被取消并让位，
// 而旧条目的收尾函数不得把新条目从表里误删
func TestSqlExecRegistryReRegisterKeepsNewEntry(t *testing.T) {
	oldCtx, oldDone := register(ctxWithAccount(1), "exec-reuse")
	newCtx, newDone := register(ctxWithAccount(1), "exec-reuse")

	select {
	case <-oldCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("重复登记时旧执行必须被取消，否则它将再也无法被中断")
	}

	// 旧条目的收尾先执行：只能删除自己，不能删掉刚登记的新条目
	oldDone()
	require.NoError(t, execRegistry.cancel(context.Background(), "exec-reuse", 1))
	select {
	case <-newCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("旧条目的收尾不得误删新条目，新执行必须仍可取消")
	}
	newDone()

	// 收尾后新条目必须已注销并留下自己的结算记录（而非被旧条目的收尾误删后查无此执行）。
	// 不能再用「cancel 必须报错」断言注销：上面那发合法取消已把新条目的 runCtx 取消，
	// 按结算原因语义，因取消而结算之后再到的取消答成功（见 CancelAfterCancelledSettleSucceeds）
	execRegistry.mu.Lock()
	settle, settled := execRegistry.recent["exec-reuse"]
	_, stillRunning := execRegistry.running["exec-reuse"]
	execRegistry.mu.Unlock()
	assert.False(t, stillRunning, "收尾后条目应已注销")
	require.True(t, settled, "新条目的收尾必须留下结算记录，证明它未被旧收尾误删")
	assert.True(t, settle.cancelled, "新条目是被合法取消停掉的，结算原因必须如实记录")
}

// 跨属主的 execId 碰撞必须被拒绝：顶替行为会取消对方正在跑的执行，
// 等于绕过 cancel 的属主校验，变成伪造 execId 中断他人执行的后门。
// 登记已前置到资源鉴权之前，这道门是碰撞路径上唯一的属主关卡
func TestSqlExecRegistryRejectsCrossOwnerCollision(t *testing.T) {
	victimCtx, victimDone := register(ctxWithAccount(101), "exec-collide")
	t.Cleanup(victimDone)

	_, _, err := execRegistry.register(ctxWithAccount(202), "exec-collide", &dto.DbSQLExecReq{ExecId: "exec-collide", DbId: 1, Db: "db1", SQL: "select 1"})
	require.Error(t, err, "跨属主碰撞必须拒绝登记")

	select {
	case <-victimCtx.Done():
		t.Fatal("跨属主碰撞不得取消他人执行")
	default:
	}
	// 原条目完好：属主仍可正常取消
	require.NoError(t, execRegistry.cancel(context.Background(), "exec-collide", 101))
}

// 旧条目刚结算（recent 已记录）后同 execId 重新登记：取消必须命中 running 里的新条目，
// 不能拿旧结算记录误报「刚刚结束」——waitEntry 每轮先查进行表再查刚结算表就为这个
func TestSqlExecRegistryRunningWinsOverRecent(t *testing.T) {
	_, done := register(ctxWithAccount(1), "exec-reborn")
	done() // 写入 recent

	newCtx, newDone := register(ctxWithAccount(1), "exec-reborn")
	t.Cleanup(newDone)

	require.NoError(t, execRegistry.cancel(context.Background(), "exec-reborn", 1), "必须命中进行中的新条目")
	select {
	case <-newCtx.Done():
	default:
		t.Fatal("新条目必须被真正取消，而非被旧结算记录遣返")
	}
}

// 执行已被取消停掉并结算后，再到的取消必须答成功：同一取消动作里 abort 走已有连接、
// 几乎必然先于 cancel-exec 新建连接落地，执行会先被 abort 停掉——
// 此时 cancel-exec 若答「刚刚结束」，等于对一个已经取消成功的动作报错
func TestSqlExecRegistryCancelAfterCancelledSettleSucceeds(t *testing.T) {
	_, done := register(ctxWithAccount(101), "exec-aborted")
	require.NoError(t, execRegistry.cancel(context.Background(), "exec-aborted", 101), "第一发取消必须命中进行表")
	done() // 执行因 ctx 被取消而结算

	require.NoError(t, execRegistry.cancel(context.Background(), "exec-aborted", 101), "取消已因取消而结算的执行应答成功")
}

// 自然跑完的结算之后再到的取消答「刚刚结束」而非成功：操作者没取消过它，
// 不能拿一句「已请求取消」掩盖「语句其实完整跑完了」这个事实
func TestSqlExecRegistryNaturalSettleIsNotCancelSuccess(t *testing.T) {
	_, done := register(ctxWithAccount(101), "exec-natural")
	done() // runCtx 未被取消：自然结算

	err := execRegistry.cancel(context.Background(), "exec-natural", 101)
	require.Error(t, err)
	bizErr, ok := err.(*errorx.BizError)
	require.True(t, ok)
	assert.Equal(t, CodeExecJustFinished, bizErr.Code())
}

// recent 分支必须与 running 分支同一属主口径：非属主对一个已结算的他人 execId 发取消，
// 绝不能答成功也不得泄露「如何结束」。否则 cancel-exec（未挂功能权限码）会凭 execId
// 成为跨账号的三值信息 oracle：探知他人某执行是否结算过、是被取消还是自然跑完
func TestSqlExecRegistryRecentSettleRejectsNonOwner(t *testing.T) {
	// 属主 101 的执行自然结算进 recent
	_, done := register(ctxWithAccount(101), "exec-settled-other")
	done()
	// 另起一个 101 的进行中执行，用于对比「非属主命中 running」的回答
	_, runningDone := register(ctxWithAccount(101), "exec-running-other")
	defer runningDone()

	// 非属主 202 取消已结算的他人执行：必须报错，且绝不能是成功或 4003
	errSettled := execRegistry.cancel(context.Background(), "exec-settled-other", 202)
	require.Error(t, errSettled, "非属主取消他人已结算执行必须被拒")
	if bizErr, ok := errSettled.(*errorx.BizError); ok {
		assert.NotEqual(t, CodeExecJustFinished, bizErr.Code(), "非属主不得收到刚结束分流码")
	}

	// 关键口径一致性：非属主命中「已结算」与命中「进行中」必须是同一种回答（均为未授权），
	// 否则 recent 分支就比 running 分支多泄露了「该执行如何结束」这个事实
	errRunning := execRegistry.cancel(context.Background(), "exec-running-other", 202)
	assert.Equal(t, errRunning.Error(), errSettled.Error(), "recent 与 running 分支对非属主必须同口径")
}

// TestSqlExecRegistryCancelWaitsForRegistration 取消比登记先到时必须等出结果，而不是报「不存在」。
//
// execId 在执行请求里带来，登记发生在执行请求被服务端处理时：用户在执行刚发起的一瞬点取消，
// 两条请求分属不同连接，取消可能先落地。直接判「找不到」会让界面说「未找到进行中的执行」而语句其实还在跑，
// 用户以为取消失败只能等它自然结束。等待有上界，从未存在的 id 依旧如实报错（见下一条用例）
func TestSqlExecRegistryCancelWaitsForRegistration(t *testing.T) {
	const execId = "exec-race"

	go func() {
		time.Sleep(200 * time.Millisecond)
		_, done := register(ctxWithAccount(101), execId)
		t.Cleanup(done)
	}()

	require.NoError(t, execRegistry.cancel(ctxWithAccount(101), execId, 101), "取消必须等到登记完成再判定")
}

// 取消比结算晚到一步（典型：长语句尾点取消，响应还在路上）时，
// 必须得到「刚刚结束」而非「查无此执行」：两种事实对操作者的意义完全不同，
// 前者是「语句跑完了」，后者意味着取消可能落空、语句去向不明
func TestSqlExecRegistryJustFinishedDiffersFromUnknown(t *testing.T) {
	_, done := register(ctxWithAccount(101), "exec-settled")
	done()

	errSettled := execRegistry.cancel(context.Background(), "exec-settled", 101)
	errUnknown := execRegistry.cancel(context.Background(), "never-registered-at-all", 101)
	require.Error(t, errSettled)
	require.Error(t, errUnknown)
	assert.NotEqual(t, errSettled.Error(), errUnknown.Error(), "刚结束与查无此执行必须是两种回答")

	// 「刚刚结束」必须带独立分流码：前端靠它压掉红色 toast（这是事实而非失败），
	// 与真失败共用 400 会让中心层对不是错误的事实向操作者喊话
	bizErr, ok := errSettled.(*errorx.BizError)
	require.True(t, ok, "刚结束的错误必须是业务错误")
	assert.Equal(t, CodeExecJustFinished, bizErr.Code())
}

// 过了 TTL 的结算记录不再被当成「刚刚结束」，且结算路径会顺手修剪过期条目
func TestSqlExecRegistryRecentExpires(t *testing.T) {
	execRegistry.mu.Lock()
	execRegistry.recent["exec-long-gone"] = recentSettle{at: time.Now().Add(-2 * recentFinishTTL)}
	execRegistry.mu.Unlock()

	errExpired := execRegistry.cancel(context.Background(), "exec-long-gone", 101)
	errUnknown := execRegistry.cancel(context.Background(), "never-registered-either", 101)
	require.Error(t, errExpired)
	assert.Equal(t, errUnknown.Error(), errExpired.Error(), "过期后的回答必须与从未登记一致")

	_, done := register(ctxWithAccount(1), "exec-prune-trigger")
	done()
	execRegistry.mu.Lock()
	_, still := execRegistry.recent["exec-long-gone"]
	execRegistry.mu.Unlock()
	assert.False(t, still, "过期条目应被结算路径修剪")
}

// 登记必须发生在取连接之前：取连接（冷池拨号/建隧道）可能超过取消侧的有界等待，
// 登记晚于取连接时，拨号期间到达的取消会误报「未找到进行中的执行」而语句随后照跑。
// 源码顺序断言：这个约束跨 api/application 两层，单靠登记表自身的用例钉不住
func TestExecSQLRegistersBeforeConnAcquisition(t *testing.T) {
	source, err := os.ReadFile("../../api/db.go")
	require.NoError(t, err)

	body := string(source)
	start := strings.Index(body, "func (d *Db) ExecSQL(")
	require.NotEqual(t, -1, start, "ExecSQL handler 应存在")
	rest := body[start+len("func (d *Db) ExecSQL("):]
	end := strings.Index(rest, "\nfunc ")
	require.NotEqual(t, -1, end, "应能找到 ExecSQL 的结束位置")
	handler := rest[:end]

	registerPos := strings.Index(handler, "RegisterExec(")
	connPos := strings.Index(handler, "GetDbConn(")
	require.NotEqual(t, -1, registerPos, "ExecSQL 必须登记交互式执行")
	require.NotEqual(t, -1, connPos, "ExecSQL 必须取连接")
	assert.Less(t, registerPos, connPos, "登记必须发生在取连接之前")
}

// ExecSQL 里 defer done() 必须注册在父超时 ctx 的 defer cancel() 之后：defer 是 LIFO，
// done 后注册才会先跑，从而在父 ctx 被 handler 自己取消之前采样结算原因。
// 顺序一旦颠倒，所有自然结束的执行都会被误记为「因取消而结束」，4003 分支静默变死代码，
// 而 TestSqlExecRegistryNaturalSettleIsNotCancelSuccess 不经过 handler 仍会绿——只有源码断言钉得住
func TestExecSQLDefersDoneAfterTimeoutCancel(t *testing.T) {
	source, err := os.ReadFile("../../api/db.go")
	require.NoError(t, err)

	body := string(source)
	start := strings.Index(body, "func (d *Db) ExecSQL(")
	require.NotEqual(t, -1, start, "ExecSQL handler 应存在")
	rest := body[start+len("func (d *Db) ExecSQL("):]
	end := strings.Index(rest, "\nfunc ")
	require.NotEqual(t, -1, end, "应能找到 ExecSQL 的结束位置")
	handler := rest[:end]

	timeoutCancelPos := strings.Index(handler, "defer cancel()")
	donePos := strings.Index(handler, "defer done()")
	require.NotEqual(t, -1, timeoutCancelPos, "ExecSQL 应 defer 父超时 ctx 的 cancel")
	require.NotEqual(t, -1, donePos, "ExecSQL 应 defer 登记收尾 done")
	assert.Less(t, timeoutCancelPos, donePos, "done 必须在 cancel 之后注册（LIFO 下先跑），否则自然结算会被误记为取消")
}
