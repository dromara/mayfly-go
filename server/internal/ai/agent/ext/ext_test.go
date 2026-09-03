package ext

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/agent/contributor"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// containsId 判断 id 是否在清单中
func containsId(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// fakeOverrideTool 同名占位工具（验证宿主安装器覆盖协议）
type fakeOverrideTool struct{ name string }

func (f *fakeOverrideTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: f.name, Desc: "fake override"}, nil
}

func (f *fakeOverrideTool) InvokableRun(ctx context.Context, arguments string, opts ...tool.Option) (string, error) {
	return "fake", nil
}

// fakeOverrideContributor 贡献同名工具的宿主扩展
type fakeOverrideContributor struct{ tool *fakeOverrideTool }

func (f *fakeOverrideContributor) Id() string { return "fake_override" }

func (f *fakeOverrideContributor) Tools(ctx context.Context, tc *contributor.ToolContributionContext) ([]tool.BaseTool, error) {
	return []tool.BaseTool{f.tool}, nil
}

// TestInstallAll_RegistersAllBuiltinExtensions 装配完整性：
// 全部内置扩展就位；sessionManager 未装配时 session_history 跳过（降级一致）
func TestInstallAll_RegistersAllBuiltinExtensions(t *testing.T) {
	b := contributor.NewBuilder()
	InstallAll(b, Deps{})

	ids := b.Build().ContributorIds()
	for _, want := range []string{"memory", "skill_injection", "skill_tools", "environment", "context_budget_footer", "safety_middleware",
		"interrupt_approval", "interrupt_param_completion"} {
		if !containsId(ids, want) {
			t.Errorf("expected contributor %q registered, got %v", want, ids)
		}
	}
	if containsId(ids, "session_history") {
		t.Errorf("nil sessionManager should skip session_history, got %v", ids)
	}
	// memory_extraction 依赖 memoryManager，nil 时跳过
	if containsId(ids, "memory_extraction") {
		t.Errorf("nil memoryManager should skip memory_extraction, got %v", ids)
	}
}

// TestRegisterHostInstaller_OverrideBuiltinTool 宿主级安装器覆盖协议：
// 内置扩展先注册、宿主扩展后注册，同名工具后注册者胜出（插件替换内置无需改宿主）
func TestRegisterHostInstaller_OverrideBuiltinTool(t *testing.T) {
	hostInstallers = nil
	t.Cleanup(func() { hostInstallers = nil })

	fake := &fakeOverrideTool{name: "skill_read"}
	RegisterHostInstaller(func(b *contributor.Builder) {
		b.RegisterTool(&fakeOverrideContributor{tool: fake})
	})

	b := contributor.NewBuilder()
	InstallAll(b, Deps{})
	RunHostInstallers(b)

	tools := b.Build().BuildTools(context.Background(), nil)
	for _, bt := range tools {
		ti, err := bt.Info(context.Background())
		if err != nil {
			t.Fatalf("get tool info: %v", err)
		}
		if ti.Name == "skill_read" && bt != tool.BaseTool(fake) {
			t.Fatalf("host installer tool should override builtin skill_read")
		}
	}

	// 宿主贡献者本身也应出现在清单中
	if !containsId(b.Build().ContributorIds(), "fake_override") {
		t.Errorf("host installer contributor should be registered")
	}
}

// TestRunHostInstallers_EmptyNoop 未注册宿主安装器时安全空操作
func TestRunHostInstallers_EmptyNoop(t *testing.T) {
	hostInstallers = nil
	t.Cleanup(func() { hostInstallers = nil })

	b := contributor.NewBuilder()
	InstallAll(b, Deps{})
	before := len(b.Build().ContributorIds())
	RunHostInstallers(b)
	if after := len(b.Build().ContributorIds()); after != before {
		t.Errorf("empty host installers should be noop: before=%d after=%d", before, after)
	}
}
