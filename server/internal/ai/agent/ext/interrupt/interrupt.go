// Package interruptext 中断类型扩展（统一装配链路）
//
// 将 tools 定义的中断类型扩展（审批/参数补全）纳入 contributor 统一装配：
//
//   - 经 ChannelService 注册，可经配置 disabledExtensions 按 Id 裁剪
//     （interrupt_approval / interrupt_param_completion）
//   - 装配期由 Registry.Activate 激活装载，宿主装配完成后 tools 侧注册表
//     冻结，运行期零变更（多实例部署下各实例装载一致清单）
package interruptext

import (
	"context"

	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/tools"
)

// InterruptExtensionContributor 中断类型扩展贡献者（装配期激活型）
type InterruptExtensionContributor struct {
	id            string
	interruptType tools.InterruptType
	newExt        func() tools.InterruptExtension
}

var _ contributor.Activatable = (*InterruptExtensionContributor)(nil)

func (e *InterruptExtensionContributor) Id() string { return e.id }

// Activate 装配期激活：装载中断类型扩展进 tools 注册表（幂等，冻结后安全跳过）
func (e *InterruptExtensionContributor) Activate(_ context.Context) error {
	tools.LoadInterruptExtension(e.interruptType, e.newExt())
	return nil
}

// Install 注册全部中断类型扩展贡献者到 Builder
func Install(b *contributor.Builder) {
	b.RegisterService(&InterruptExtensionContributor{
		id:            "interrupt_approval",
		interruptType: tools.InterruptTypeApproval,
		newExt:        tools.NewApprovalExtension,
	})
	b.RegisterService(&InterruptExtensionContributor{
		id:            "interrupt_param_completion",
		interruptType: tools.InterruptTypeParamCompletion,
		newExt:        tools.NewParamCompletionExtension,
	})
}
