package trigger

import (
	"context"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/model"
	"strings"
	"sync"
)

// Context 一次触发策略求值的上下文，负责按字段字典惰性解析取值。
// 同一次求值内字段结果被缓存，多个条件引用同一字段只计算一次
type Context struct {
	ctx context.Context

	// BizType 本次求值针对的业务场景
	BizType string
	// ProcdefId 所属流程定义 id，仅用于日志与试算回显
	ProcdefId uint64
	// Raw 业务侧提交的原始输入（一条 SQL、一条命令、一个路径等）
	Raw map[string]string
	// Attributes 业务侧已权威计算出的结构化事实（如解析后的语句类型、目标表），
	// 避开在求值路径上重复解析；同名字段以 Raw 为准
	Attributes map[string]any
	// CodePaths 本次操作所在资源的标签路径，供检查项按资源解析自己的配置数据
	// （如机器命令黑名单挂在机器标签上），避免每个调用方都为「取配置」写一遍前置查询
	CodePaths []string
	// Login 发起操作的用户，供通用检查项按人判定
	Login *model.LoginAccount
	// segments 可复用条件组的取数函数，条件树含 segment 节点时必须设置，
	// 否则引用无法展开，按「无法判定」报错而不是当成不成立
	segments SegmentResolver

	mu       sync.Mutex
	values   map[string]any
	resolved map[string]bool
	// Unknown 未能取到值的字段 key（未注册、缺失或派生失败）
	Unknown []string
}

// NewContext 创建求值上下文
func NewContext(ctx context.Context, bizType string, procdefId uint64, raw map[string]string, login *model.LoginAccount) *Context {
	if raw == nil {
		raw = map[string]string{}
	}
	return &Context{
		ctx:       ctx,
		BizType:   bizType,
		ProcdefId: procdefId,
		Raw:       raw,
		Login:     login,
		values:    map[string]any{},
		resolved:  map[string]bool{},
	}
}

// GoContext 返回派生字段可使用的标准上下文
func (tc *Context) GoContext() context.Context {
	return tc.ctx
}

// WithAttributes 附加业务侧已计算出的结构化事实，返回自身以便链式调用
func (tc *Context) WithAttributes(attributes map[string]any) *Context {
	tc.Attributes = attributes
	return tc
}

// WithCodePaths 附加资源标签路径，返回自身以便链式调用
func (tc *Context) WithCodePaths(codePaths []string) *Context {
	tc.CodePaths = codePaths
	return tc
}

// WithSegments 设置可复用条件组的取数函数，返回自身以便链式调用
func (tc *Context) WithSegments(resolver SegmentResolver) *Context {
	tc.segments = resolver
	return tc
}

// SegmentResolver 返回本次求值可用的条件组取数函数，未设置时返回 nil
func (tc *Context) SegmentResolver() SegmentResolver {
	return tc.segments
}

// Attribute 读取业务侧附加的结构化事实
func (tc *Context) Attribute(key string) (any, bool) {
	value, ok := tc.Attributes[key]
	return value, ok
}

// RawValue 读取原始输入项
func (tc *Context) RawValue(key string) (string, bool) {
	value, ok := tc.Raw[key]
	return value, ok
}

// FieldValue 按字段字典解析取值。
//
// 三个结果要分开看，不能合并成「取不到就不成立」：
//   - found=false 表示调用方确实没提供这个字段，条件按不成立处理并记入 Unknown
//   - err 非空表示派生过程本身失败（如需远程查询的字段超时或报错），
//     这属于「无法判定」，调用方必须 fail-closed，否则新场景一旦接入需查库的派生字段，
//     一次查询失败就会让规则静默放行
func (tc *Context) FieldValue(field TriggerField) (value any, found bool, err error) {
	tc.mu.Lock()
	if tc.resolved[field.Key] {
		cached := tc.values[field.Key]
		tc.mu.Unlock()
		return cached, true, nil
	}
	tc.mu.Unlock()

	value, found, err = tc.resolveFieldValue(field)
	tc.mu.Lock()
	defer tc.mu.Unlock()
	if err != nil || !found {
		tc.markUnknownLocked(field.Key)
		return nil, false, err
	}
	tc.values[field.Key] = value
	tc.resolved[field.Key] = true
	return value, true, nil
}

func (tc *Context) resolveFieldValue(field TriggerField) (any, bool, error) {
	if field.Resolve == nil {
		if value, ok := tc.RawValue(field.Key); ok {
			return value, true, nil
		}
		value, ok := tc.Attributes[field.Key]
		return value, ok, nil
	}

	value, err := field.Resolve(tc)
	if err != nil {
		return nil, false, err
	}
	if value == nil {
		// 派生函数返回 nil 表示「这会儿算不出来」（如会签总数为 0 时算不出完成率）。
		// 归成 found=true 会让条件静默判假且不记 Unknown，界面与日志都没有痕迹，
		// 排查时只会看到「审批通过了流程却没往下走」
		return nil, false, nil
	}
	return value, true, nil
}

// markUnknown 记录无法取值的字段 key（自行加锁）
func (tc *Context) markUnknown(fieldKey string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.markUnknownLocked(fieldKey)
}

// markUnknownLocked 追加无法判定的字段名，同一个字段只记一次（调用方需已持有锁）。
//
// 试算会把该场景的字段字典全部解析一遍（补上只被检查项消费的解析结果），同一个取不出值的
// 字段会被再次问到；不去重的话「未能判定」清单里同一项出现多次，管理员会以为是多处故障
func (tc *Context) markUnknownLocked(fieldKey string) {
	for _, known := range tc.Unknown {
		if known == fieldKey {
			return
		}
	}
	tc.Unknown = append(tc.Unknown, fieldKey)
}

// ResolvedFields 返回本次求值实际取到值的字段，供试算回显「后端把输入解析成了什么」
func (tc *Context) ResolvedFields() map[string]any {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	fields := make(map[string]any, len(tc.values))
	for key, value := range tc.values {
		fields[key] = value
	}
	return fields
}

// Finding 策略求值产出一条结论。Title 为前端与错误消息使用的 i18n key
type Finding struct {
	// Source 结论来源：检查项 key、"custom.when"、"custom.unless" 或 "engine"
	Source   string          `json:"source"`
	Severity entity.Severity `json:"severity"`
	Title    string          `json:"title"`
	Detail   map[string]any  `json:"detail,omitempty"`

	// Summary 服务端提示用的消息 id，为 0 表示该结论没有可独立展示的原因（如引擎兜底）
	Summary i18n.MsgId `json:"-"`
}

// Decision 策略求值的最终结论
type Decision struct {
	// Severity 所有结论中最严格的处置级别
	Severity entity.Severity `json:"severity"`
	Findings []*Finding      `json:"findings"`
	// Unknown 未能取到值的字段 key
	Unknown []string `json:"unknown,omitempty"`
	// Exempted 命中 unless 豁免，此时 Severity 被降为不处置
	Exempted bool `json:"exempted"`

	// Matched 表示本次求值是否命中了某条规则（区别于「未配置规则走兜底」）
	Matched bool `json:"matched"`
	// ProcdefId、ProcdefName 生效的流程定义，供拦截提示与试算回显定位来源
	ProcdefId   uint64 `json:"procdefId"`
	ProcdefName string `json:"procdefName"`
}

// Reason 把命中的规则摘要拼成一句可直接进错误消息的原因。
//
// 只有带服务端消息 id 的结论会进入原因：引擎兜底这类内部状态对人没有解释力，
// 把「无法判定」和「命中了哪条规则」混在一句提示里反而误导
func (d *Decision) Reason(ctx context.Context) string {
	if d == nil {
		return ""
	}
	parts := make([]string, 0, len(d.Findings))
	for _, finding := range d.Findings {
		if finding.Summary == 0 {
			continue
		}
		parts = append(parts, i18n.TC(ctx, finding.Summary))
	}
	return strings.Join(parts, "、")
}

// Notices 返回不阻断执行但需要让操作者知道的结论（提醒级），以及无法判定被兜底的情况。
//
// 这些结论不会改变放行结果，因此可以安全地随成功响应回传，让「仅提醒」这一级别有出口，
// 而不是配了却只进服务端日志
func (d *Decision) Notices() []*Finding {
	if d == nil {
		return nil
	}
	notices := make([]*Finding, 0, len(d.Findings))
	for _, finding := range d.Findings {
		if finding.Severity == entity.SeverityWarning {
			notices = append(notices, finding)
		}
	}
	return notices
}

// RequiresApproval 判断结论是否要求提交审批工单
func (d *Decision) RequiresApproval() bool {
	return d != nil && d.Severity == entity.SeverityRequired
}

// IsBlocked 该结论是否需要阻断本次操作。
//
// 「仅提醒」与「不处置」不阻断。这个口径是三个资源入口共用的：写在三处就会有三份语义，
// 新场景接入时最容易漏掉的正是其中一份
func (d *Decision) IsBlocked() bool {
	return d != nil && (d.IsForbidden() || d.RequiresApproval())
}

// IsForbidden 判断结论是否禁止执行
func (d *Decision) IsForbidden() bool {
	return d != nil && d.Severity == entity.SeverityForbidden
}
