package sqlexec

// 进行中执行登记表的回归测试。
//
// 守的是「取消」这条通道的三条安全/正确性边界：
//  1. 只有发起者能取消自己的执行（否则任何持 token 的账号可中断他人查询）；
//  2. 登记时拿不到登录账号的执行谁都取消不了（异常登记不能变成免鉴权入口）；
//  3. 重复 execId 不会让旧条目失去可取消性，也不会让旧条目的收尾误删新条目。
//
// 运行：cd server && go test -count=1 -race -run TestSqlExecRegistry ./internal/db/application/

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
)

// ctxWithAccount 用真实写入函数构造带登录账号的 ctx：
// 登记表的属主取自 contextx.GetLoginAccount，测试自造 key 会让 accountId 恒为 0，测不到属主分支
func ctxWithAccount(id uint64) context.Context {
	return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: id, Username: "u"})
}

func register(ctx context.Context, execId string) (context.Context, func()) {
	return execRegistry.register(ctx, execId, &dto.DbSQLExecReq{ExecId: execId, DbId: 1, Db: "db1", SQL: "select 1"})
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
	ctx, done := execRegistry.register(context.Background(), "exec-ownerless", &dto.DbSQLExecReq{ExecId: "exec-ownerless", DbId: 1, Db: "db1", SQL: "select 1"})
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

	assert.Error(t, execRegistry.cancel(context.Background(), "exec-reuse", 1), "收尾后条目应已注销")
}

// TestSqlExecRegistryCancelWaitsForRegistration 取消比登记先到时必须等出结果，而不是报「不存在」。
//
// execId 在执行请求里带来，但登记发生在取到连接之后：用户在执行刚发起的一瞬点取消，
// 两条请求几乎同时落地。直接判「找不到」会让界面说「该执行已结束」而语句其实还在跑，
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
