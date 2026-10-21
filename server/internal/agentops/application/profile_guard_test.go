package application

import (
	"testing"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/trigger"
	"mayfly-go/internal/ai/pkg/headless"

	"context"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/pkg/i18n"
)

// fakeProfileCatalog 档位目录测试桩（headless.ProfileCatalog 端口的最小实现）
type fakeProfileCatalog struct {
	infos []headless.ProfileInfo
}

func (f fakeProfileCatalog) List() []headless.ProfileInfo { return f.infos }

func (f fakeProfileCatalog) Find(id string) (headless.ProfileInfo, bool) {
	for _, info := range f.infos {
		if info.Id == id {
			return info, true
		}
	}
	return headless.ProfileInfo{}, false
}

// withProfileCatalog 注入档位目录，用例结束后还原为「未装配」状态。
//
// 目录是进程级注册状态，不还原会污染同包其他用例（它们依赖「无目录即不拦」）
func withProfileCatalog(t *testing.T, infos ...headless.ProfileInfo) {
	t.Helper()
	headless.RegisterProfileCatalog(fakeProfileCatalog{infos: infos})
	t.Cleanup(func() { headless.RegisterProfileCatalog(nil) })
}

func TestValidateDispatchProfile(t *testing.T) {
	t.Run("目录未装配时放行", func(t *testing.T) {
		// 单测与降级启动会绕过 ai 装配：此时无从判定，不拦，避免把业务链路卡死
		require.NoError(t, validateDispatchProfile(context.Background(), "any-profile"))
	})

	withProfileCatalog(t,
		headless.ProfileInfo{Id: "diagnostician"},
		headless.ProfileInfo{Id: "executor", RequireRunGrant: true},
	)

	t.Run("未知档位必须拒绝", func(t *testing.T) {
		// 不静默兜底成默认档：改写用户选定的档位比报错更难排查
		requireMsgId(t, validateDispatchProfile(context.Background(), "not-registered"), imsg.ErrProfileUnknown, "id", "not-registered")
	})

	t.Run("需一次性许可的档不能无人值守派发", func(t *testing.T) {
		// 这类档拿不到 run grant 时护栏会拒掉全部工具调用：任务每次运行都「成功但什么都没做」
		requireMsgId(t, validateDispatchProfile(context.Background(), "executor"), imsg.ErrProfileRequiresGrant, "id", "executor")
	})

	t.Run("只读诊断档放行", func(t *testing.T) {
		require.NoError(t, validateDispatchProfile(context.Background(), "diagnostician"))
	})
}

// TestSaveTaskRejectsGrantOnlyProfile 档位守护必须接在任务保存期：
// 只靠派发期拦截，用户建完任务看不出问题，要等到执行历史里出现一条空诊断才发现
func TestSaveTaskRejectsGrantOnlyProfile(t *testing.T) {
	withProfileCatalog(t,
		headless.ProfileInfo{Id: "diagnostician"},
		headless.ProfileInfo{Id: "executor", RequireRunGrant: true},
	)
	taskSvc, _ := setupTaskSvc(t)
	ctx := runTestCtx()

	grantOnly := validTask()
	grantOnly.ProfileId = "executor"
	requireMsgId(t, taskSvc.SaveTask(ctx, grantOnly), imsg.ErrProfileRequiresGrant, "id", "executor")

	unknown := validTask()
	unknown.ProfileId = "ghost-profile"
	requireMsgId(t, taskSvc.SaveTask(ctx, unknown), imsg.ErrProfileUnknown, "id", "ghost-profile")

	ok := validTask()
	ok.ProfileId = "diagnostician"
	require.NoError(t, taskSvc.SaveTask(ctx, ok), "只读诊断档正常保存")
}

// TestSaveTriggerRejectsGrantOnlyProfile 触发器同理：事件驱动派发也不下发许可
func TestSaveTriggerRejectsGrantOnlyProfile(t *testing.T) {
	withProfileCatalog(t,
		headless.ProfileInfo{Id: "diagnostician"},
		headless.ProfileInfo{Id: "executor", RequireRunGrant: true},
	)
	svc, _, _ := setupTriggerSvc(t)
	ctx := runTestCtx()

	tr := &entity.Trigger{
		Name: "wh", SourceType: trigger.SourceTypeWebhook, OwnerAccountId: 7,
		Instruction: "诊断", Enabled: entity.TriggerEnabled, ProfileId: "executor",
	}
	requireMsgId(t, svc.SaveTrigger(ctx, tr), imsg.ErrProfileRequiresGrant, "id", "executor")
}

// requireMsgId 断言错误文案就是该 msgId 的渲染结果。
//
// 存在理由：文案已按请求语言本地化，测试里再按英文字面量子串匹配等于把断言绑在语言上——
// 服务端默认语言一改就假失败。改成比 msgId 渲染值，既钉住「是哪一条判定拒的」又与语言无关。
func requireMsgId(t *testing.T, err error, msgId i18n.MsgId, attrs ...any) {
	t.Helper()
	require.Error(t, err)
	require.Equal(t, i18n.T(msgId, attrs...), err.Error())
}
