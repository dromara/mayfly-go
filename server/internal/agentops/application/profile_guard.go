// 档位派发守护：无人值守派发（cron / 事件 / 手动）允许使用哪些 Agent 装配档，单点判定。
//
// 为什么必须拦在保存期而不是留到运行期：RequireRunGrant 的档（如受控执行档）在没拿到一次性
// 运行许可时，capability_guard 会拒掉它的**全部**工具调用——这是 fail-closed 的设计意图。
// 而 cron/事件/手动这条派发链从不下发许可（只有提案经人工审批后由网关下发那一个动作的许可）。
// 结果是一条配了执行档的定时任务每次运行都「成功结束但一个工具也没调过」，产出一份空诊断：
// 没有报错、没有失败标记、执行历史看着一切正常，用户完全得不到信号（静默废任务）。
package application

import (
	"context"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/errorx"
)

// validateDispatchProfile 校验该档位可用于无人值守派发：
//   - 档位不存在 → 拒绝（不静默兜底成默认档：改写用户选定的档位比报错更难排查）
//   - 需一次性许可的档 → 拒绝，并说明它只能由审批后的提案执行驱动
//   - ai 侧目录未装配 → 放行（单测/降级启动场景无从判定，不把业务链路卡死）
func validateDispatchProfile(ctx context.Context, profileId string) error {
	catalog := headless.Profiles()
	if catalog == nil {
		return nil
	}
	info, ok := catalog.Find(profileId)
	if !ok {
		return errorx.NewBizI(ctx, imsg.ErrProfileUnknown, "id", profileId)
	}
	if info.RequireRunGrant {
		return errorx.NewBizI(ctx, imsg.ErrProfileRequiresGrant, "id", profileId)
	}
	return nil
}
