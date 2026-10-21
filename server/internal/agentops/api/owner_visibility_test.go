package api

import (
	"testing"

	"mayfly-go/internal/pkg/consts"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/pkg/i18n"
)

// TestAssertResourceOwned 任务/触发器归属断言的契约。
//
// 越权派发是本案最危险的入口：dispatchTask 以**任务 owner 的身份**入队运行，因此只持有
// task:run 权限码的账号遍历/猜一个 id，就能以 admin 身份读监控数据、并登记 owner=admin 的
// 变更提案。owner.go 的威胁模型注释明确写了要防这件事，但 Task/Trigger 聚合此前一处都没落地
// （只有 run 侧做了收窄）—— 这条测试就是防止再出现「一处做了三处没做」的不一致。
func TestAssertResourceOwned(t *testing.T) {
	useRegistry(t, fakeCodeRegistry{codes: map[uint64][]string{
		9: {PermissionRunAsOwner},
	}})

	require.NoError(t, assertResourceOwned(runCtxOf(7), 7), "自己的资源永远可操作")
	require.NoError(t, assertResourceOwned(runCtxOf(consts.AdminId), 7), "超级管理员可跨账号")
	require.NoError(t, assertResourceOwned(runCtxOf(9), 7), "持 impersonate 码者可跨账号")

	// 越权时报错必须与「资源不存在」是同一条消息，而不是权限拒绝：后者可被用来探测他人资源 id 是否存在。
	// 断言取 i18n 渲染结果而非英文字面量：文案已按请求语言本地化，写死英文等于把断言绑在语言上。
	err := assertResourceOwned(runCtxOf(7), 8)
	require.Error(t, err)
	require.Equal(t, i18n.T(imsg.ErrResourceNotFound), err.Error(), "越权必须报「资源不存在」")
	require.NotContains(t, err.Error(), i18n.T(imsg.ErrNotPermittedImpersonate), "越权不得回传权限拒绝文案，否则可探测资源存在性")

	// 权限注册表未装配时 fail-closed：只剩「自己的资源可操作」，不得默认放行跨账号
	useRegistry(t, nil)
	require.NoError(t, assertResourceOwned(runCtxOf(7), 7))
	require.Error(t, assertResourceOwned(runCtxOf(7), 8), "注册表缺失时不得放开跨账号操作")
}

// TestScopeOwnerToCaller 列表收窄必须覆写调用者传入的 owner 条件。
//
// 列表不收窄的话，非全局可见者虽改不了他人任务，却能读到其完整指令与调度配置——
// 指令里常含机器名/库名/业务口径，属于真实的信息泄露面。
func TestScopeOwnerToCaller(t *testing.T) {
	useRegistry(t, fakeCodeRegistry{codes: map[uint64][]string{}})

	// 伪造 ownerAccountId 必须被覆写为调用者自己（与 scopeRunsToCaller 同口径）
	scoped := uint64(123)
	scopeOwnerToCaller(runCtxOf(7), func(accountId uint64) { scoped = accountId })
	require.Equal(t, uint64(7), scoped)

	// 超管不收窄：保持调用者给出的原条件（0 = 全量）
	scoped = 0
	scopeOwnerToCaller(runCtxOf(consts.AdminId), func(accountId uint64) { scoped = accountId })
	require.Equal(t, uint64(0), scoped, "全局可见者不得被强制覆写查询条件")
}
