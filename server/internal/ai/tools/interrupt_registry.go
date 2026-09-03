package tools

import (
	"context"
	"sync"

	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/collx"
)

// InterruptExtension 中断类型扩展点（注册式，满足开闭原则）。
// 新增中断类型时只需：1. 实现本接口；2. 经中断类型扩展贡献者
// （agent/ext/interrupt）纳入 contributor 统一装配链路 —— 装配期由
// Registry.Activate 装载、宿主装配完成后冻结，可经配置按 Id 裁剪。
// agent.Run / event_mapper / api 层均通过本扩展点分发，无需感知具体中断类型。
type InterruptExtension interface {
	// ConvertResume 将通用的 InterruptResume 转换为该类型对应的恢复目标结构体
	ConvertResume(resume *InterruptResume) any

	// PrepareResumeCtx 在 runner.ResumeWithParams 前调用，可包装 ctx 或做类型特有的预处理
	// （如参数补全类型需要按 toolCallId 缓存 resume 数据、注入 Go context）
	PrepareResumeCtx(ctx context.Context, resume *InterruptResume) context.Context

	// ExtraEventMetadata 为前端中断事件补充该类型特有的 metadata（如参数补全的 paramType/options）
	ExtraEventMetadata(info InterruptMetadata) collx.M
}

// 中断类型扩展注册表：仅在装配期写入，冻结后只读。
//
// 与旧的「运行期可变全局单例」不同：
//   - 装载入口唯一（LoadInterruptExtension），仅由中断扩展贡献者在
//     Registry.Activate 时调用（统一装配链路，可经 disabledExtensions 裁剪）
//   - 宿主装配完成后 Freeze，运行期零变更 —— 并发读安全，
//     多实例部署下各实例装载相同清单，无状态水平扩容
var (
	interruptExtensionsMu sync.RWMutex
	interruptExtensions   = make(map[InterruptType]InterruptExtension)
	interruptFrozen       bool
)

// LoadInterruptExtension 装配期装载中断类型扩展（幂等，覆盖旧值）
//
// 冻结后调用仅告警跳过不报错：装配失败重试时扩展已装载，重复装载同一
// 扩展无副作用，保证装配幂等。
func LoadInterruptExtension(t InterruptType, ext InterruptExtension) {
	if ext == nil {
		return
	}
	interruptExtensionsMu.Lock()
	defer interruptExtensionsMu.Unlock()
	if interruptFrozen {
		logx.Warnf("[tools] interrupt registry frozen, ignore loading extension %s", t)
		return
	}
	interruptExtensions[t] = ext
}

// FreezeInterruptExtensions 冻结中断扩展注册表（宿主装配完成后调用，幂等）。
// 冻结后注册表只读，运行期零变更（多实例部署下各实例清单一致）。
func FreezeInterruptExtensions() {
	interruptExtensionsMu.Lock()
	defer interruptExtensionsMu.Unlock()
	interruptFrozen = true
}

// GetInterruptExtension 获取中断类型扩展
func GetInterruptExtension(t InterruptType) InterruptExtension {
	interruptExtensionsMu.RLock()
	defer interruptExtensionsMu.RUnlock()
	return interruptExtensions[t]
}

// ConvertInterruptResume 将 InterruptResume 转换为具体恢复目标（未注册类型返回原值）
func ConvertInterruptResume(resume *InterruptResume) any {
	if ext := GetInterruptExtension(resume.InterruptType); ext != nil {
		return ext.ConvertResume(resume)
	}
	return resume
}

// PrepareInterruptResumeCtx 对 resume 执行类型特有的预处理（未注册类型原样返回 ctx）
func PrepareInterruptResumeCtx(ctx context.Context, resume *InterruptResume) context.Context {
	if ext := GetInterruptExtension(resume.InterruptType); ext != nil {
		return ext.PrepareResumeCtx(ctx, resume)
	}
	return ctx
}

// ExtraInterruptEventMetadata 获取中断类型特有的前端事件 metadata（未注册类型返回 nil）
func ExtraInterruptEventMetadata(info InterruptMetadata) collx.M {
	if ext := GetInterruptExtension(info.GetType()); ext != nil {
		return ext.ExtraEventMetadata(info)
	}
	return nil
}

// IsInterruptContent 判断内部消息的 content 是否为中断元数据
// （api 层据此分发中断事件与普通内部事件，无需硬编码中断类型）
func IsInterruptContent(v any) bool {
	_, ok := v.(InterruptMetadata)
	return ok
}
