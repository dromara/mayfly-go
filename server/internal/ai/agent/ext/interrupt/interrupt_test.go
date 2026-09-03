package interruptext

import (
	"context"
	"testing"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/tools"
)

// TestInstall_RegistersAllInterruptExtensions 装配完整性：全部中断类型扩展贡献者就位
func TestInstall_RegistersAllInterruptExtensions(t *testing.T) {
	b := contributor.NewBuilder()
	Install(b)

	ids := b.Build().ContributorIds()
	for _, want := range []string{"interrupt_approval", "interrupt_param_completion"} {
		found := false
		for _, id := range ids {
			if id == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected contributor %q registered, got %v", want, ids)
		}
	}
}

// TestActivate_LoadsExtensions 激活后中断扩展在 tools 注册表可用（幂等重复激活无害）
func TestActivate_LoadsExtensions(t *testing.T) {
	b := contributor.NewBuilder()
	Install(b)
	r := b.Build()

	// 激活两次验证幂等（冻结后重复装载安全跳过）
	if err := r.Activate(context.Background()); err != nil {
		t.Fatalf("activate: %v", err)
	}
	tools.FreezeInterruptExtensions()
	if err := r.Activate(context.Background()); err != nil {
		t.Fatalf("re-activate after freeze: %v", err)
	}

	if tools.GetInterruptExtension(tools.InterruptTypeApproval) == nil {
		t.Error("approval interrupt extension should be loaded after activate")
	}
	if tools.GetInterruptExtension(tools.InterruptTypeParamCompletion) == nil {
		t.Error("param completion interrupt extension should be loaded after activate")
	}
}

// TestActivate_WithFilter 被裁剪的中断扩展贡献者不激活
//
// 注：其他用例可能已装载全部扩展，此处仅验证激活调用未发生（不反证注册表为空）
func TestActivate_WithFilter(t *testing.T) {
	b := contributor.NewBuilder()
	Install(b)
	filtered := b.Build().WithFilter(map[string]struct{}{"interrupt_approval": {}})

	if filtered.ChannelCount(contributor.ChannelService) != 1 {
		t.Errorf("expected 1 service contributor after filter, got %d", filtered.ChannelCount(contributor.ChannelService))
	}
	if err := filtered.Activate(context.Background()); err != nil {
		t.Fatalf("activate: %v", err)
	}
}
