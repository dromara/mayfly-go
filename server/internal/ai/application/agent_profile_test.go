package application

import (
	"context"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/pkg/i18n"
	"testing"

	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/internal/ai/tools"

	"github.com/stretchr/testify/require"
)

func TestBuiltinProfilesRegistered(t *testing.T) {
	for _, id := range []string{ProfileDiagnostician, ProfileExecutor, ProfileReviewer, ProfileAuditor} {
		p, ok := GetProfile(id)
		require.True(t, ok, "内置档应已注册: %s", id)
		require.Equal(t, id, p.Id)
	}

	// 诊断/审查/审批档为只读策略；执行档不限制
	diag, _ := GetProfile(ProfileDiagnostician)
	require.True(t, diag.ToolPolicy.IsRestricted())
	require.True(t, diag.ToolPolicy.Allows("DbQueryData"))
	require.False(t, diag.ToolPolicy.Allows("ExecSql"))

	exec, _ := GetProfile(ProfileExecutor)
	require.False(t, exec.ToolPolicy.IsRestricted(), "执行档不限制能力（受多重运行期约束）")
}

func TestRegisterAndGetProfile(t *testing.T) {
	const id = "__test_profile__"
	t.Cleanup(func() {
		profileMu.Lock()
		delete(profiles, id)
		delete(profileAgents, id)
		profileMu.Unlock()
	})

	// 空/无 Id 忽略
	RegisterProfile(nil)
	RegisterProfile(&AgentProfile{})

	RegisterProfile(&AgentProfile{
		Id:          id,
		Name:        "test",
		ToolPolicy:  tools.ReadOnlyPolicy(),
		Instruction: "you are a test",
	})
	p, ok := GetProfile(id)
	require.True(t, ok)
	require.Equal(t, "test", p.Name)

	// ListProfiles 含新注册档，且返回副本（改副本不影响注册表）
	list := ListProfiles()
	found := false
	for _, item := range list {
		if item.Id == id {
			found = true
			item.Name = "mutated"
		}
	}
	require.True(t, found)
	p2, _ := GetProfile(id)
	require.Equal(t, "test", p2.Name, "ListProfiles 应返回副本")

	// 覆盖注册
	RegisterProfile(&AgentProfile{Id: id, Name: "override"})
	p3, _ := GetProfile(id)
	require.Equal(t, "override", p3.Name)
}

func TestGetProfileAgent_UnknownProfile(t *testing.T) {
	// 未知档在构建 Agent（需运行时装配）前即返回错误，无需 DB/LLM 基础设施
	_, err := GetProfileAgent(context.Background(), "__no_such_profile__")
	require.Error(t, err)
	require.Contains(t, err.Error(), i18n.T(imsg.ErrUnknownAgentProfile, "profileId", "__no_such_profile__"))
}

func TestResetProfile_NoPanic(t *testing.T) {
	// 失效不存在的档 / 内置档缓存均安全（幂等）
	ResetProfile("__no_such_profile__")
	ResetProfile(ProfileDiagnostician)
	ResetAllProfiles()
}

func TestProfileIdConstantsAlignWithHeadlessContract(t *testing.T) {
	// application 再导出常量须与中立契约包一致（消费方以 headless.* 引用）
	require.Equal(t, headless.ProfileDiagnostician, ProfileDiagnostician)
	require.Equal(t, headless.ProfileExecutor, ProfileExecutor)
	require.Equal(t, headless.ProfileReviewer, ProfileReviewer)
	require.Equal(t, headless.ProfileAuditor, ProfileAuditor)
}

// TestProfileCatalogAdapter 档位目录适配器的契约（headless.ProfileCatalog 端口的真实实现）。
//
// 消费方（agentops 档位守护）测的是 fake 目录，真适配器一旦漏字段/漏档位只能在这里抓：
// 漏一档就是「后端注册了但界面永远选不到」，漏 RequireRunGrant 就是静默废任务缺陷复发
func TestProfileCatalogAdapter(t *testing.T) {
	catalog := profileCatalogImpl{}

	infos := catalog.List()
	byId := make(map[string]headless.ProfileInfo, len(infos))
	for i := 1; i < len(infos); i++ {
		require.Less(t, infos[i-1].Id, infos[i].Id, "目录必须按 id 升序（下拉顺序稳定性依赖此约定）")
	}
	for _, info := range infos {
		byId[info.Id] = info
	}

	// 内置四档全部可见，且展示名/说明非空
	for _, id := range []string{ProfileDiagnostician, ProfileExecutor, ProfileReviewer, ProfileAuditor} {
		info, ok := byId[id]
		require.True(t, ok, "内置档 %s 必须出现在目录里", id)
		require.NotEmpty(t, info.Name, "档 %s 缺少展示名", id)
		require.NotEmpty(t, info.Description, "档 %s 缺少说明", id)
	}

	// 注册表新增一档后目录必须同步看到（避免适配器只返硬编码清单，重回双真源）
	RegisterProfile(&AgentProfile{Id: "catalog-probe", Name: "探针档"})
	t.Cleanup(func() { profileMu.Lock(); delete(profiles, "catalog-probe"); profileMu.Unlock() })
	probe, ok := catalog.Find("catalog-probe")
	require.True(t, ok, "RegisterProfile 注册的自定义档必须立即可被目录发现")
	require.Equal(t, "探针档", probe.Name)
	require.False(t, probe.RequireRunGrant, "未声明许可要求的档默认不得拦截无人值守派发")

	grantOnly, ok := catalog.Find(ProfileExecutor)
	require.True(t, ok)
	require.True(t, grantOnly.RequireRunGrant, "受控执行档必须声明需一次性许可，否则前端不会置灰、派发也不会被拦")

	readOnly, ok := catalog.Find(ProfileDiagnostician)
	require.True(t, ok)
	require.False(t, readOnly.RequireRunGrant)

	_, ok = catalog.Find("__no_such_profile__")
	require.False(t, ok, "未知档必须返回 false（派发校验依赖它拒掉任意字符串档位）")
}
