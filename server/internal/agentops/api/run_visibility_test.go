package api

import (
	"context"
	"testing"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"

	"github.com/stretchr/testify/require"
)

// fakeCodeRegistry 权限码注册表测试桩（真实实现依赖 Redis/DB，判定逻辑只需码集合）
type fakeCodeRegistry struct {
	codes map[uint64][]string
}

func (f fakeCodeRegistry) SaveCodes(uint64, []string) {}

func (f fakeCodeRegistry) HasCode(userId uint64, code string) bool {
	for _, c := range f.codes[userId] {
		if c == code {
			return true
		}
	}
	return false
}

func (f fakeCodeRegistry) Remove(uint64) {}

// useRegistry 替换全局权限码注册表并在用例结束后还原（注册表是包级可变状态）
func useRegistry(t *testing.T, registry req.PermissionCodeRegistry) {
	t.Helper()
	prev := req.GetPermissionCodeRegistery()
	req.SetPermissionCodeRegistery(registry)
	t.Cleanup(func() { req.SetPermissionCodeRegistery(prev) })
}

func runCtxOf(accountId uint64) *req.Ctx {
	return &req.Ctx{MetaCtx: contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: accountId, Username: "u"})}
}

// TestCanQueryAllRuns 「谁能看全量运行」的判定契约。
//
// 运行列表 GET 按平台惯例不挂权限码，可见面全靠这里：判错一次就是任意登录用户
// 读到他人运行的提示词与结论，而且不会有任何报错
func TestCanQueryAllRuns(t *testing.T) {
	useRegistry(t, fakeCodeRegistry{codes: map[uint64][]string{
		9: {PermissionRunQueryAll},
	}})

	require.True(t, canQueryAllRuns(runCtxOf(consts.AdminId)), "超级管理员放行全量")
	require.True(t, canQueryAllRuns(runCtxOf(9)), "持 queryAll 码的账号放行全量")
	require.False(t, canQueryAllRuns(runCtxOf(8)), "无码普通账号不得看全量")

	// 注册表未装配时按「无权限」处理（fail-closed），而不是默认放行
	useRegistry(t, nil)
	require.False(t, canQueryAllRuns(runCtxOf(8)))
}

// TestScopeRunsToCaller 查询条件的收窄契约：含「请求方伪造 owner」这一攻击形态。
func TestScopeRunsToCaller(t *testing.T) {
	useRegistry(t, fakeCodeRegistry{})

	own := &entity.Run{}
	scopeRunsToCaller(runCtxOf(8), own)
	require.Equal(t, uint64(8), own.OwnerAccountId, "无码账号必须被强制收窄到自己的运行")

	// 伪造 ownerAccountId 不能越权：收窄发生在查询前并覆写该字段
	forged := &entity.Run{OwnerAccountId: consts.AdminId}
	scopeRunsToCaller(runCtxOf(8), forged)
	require.Equal(t, uint64(8), forged.OwnerAccountId, "伪造的 owner 必须被覆写，否则按他人 id 就能读到其运行")

	// 授权账号保留自己按 owner 筛选的能力（收窄不能顺手抹掉合法过滤条件）
	useRegistry(t, fakeCodeRegistry{codes: map[uint64][]string{9: {PermissionRunQueryAll}}})
	kept := &entity.Run{OwnerAccountId: 3}
	scopeRunsToCaller(runCtxOf(9), kept)
	require.Equal(t, uint64(3), kept.OwnerAccountId, "持码账号显式指定的 owner 过滤必须保留")

	full := &entity.Run{}
	scopeRunsToCaller(runCtxOf(9), full)
	require.Zero(t, full.OwnerAccountId, "持码账号不指定 owner 时查全量")
}

// TestRunCallerIdRejectsAnonymous 无登录态必须中断，不能退化成账号 0。
//
// NewCond 的 Eq 会忽略零值条件：若把「拿不到账号」当 0 处理，「按 owner 收窄」会静默变成「不过滤」
func TestRunCallerIdRejectsAnonymous(t *testing.T) {
	anonymous := &req.Ctx{MetaCtx: context.Background()}
	require.Panics(t, func() { runCallerId(anonymous) }, "无登录态必须中断请求")
	require.Panics(t, func() { canQueryAllRuns(anonymous) })
}
