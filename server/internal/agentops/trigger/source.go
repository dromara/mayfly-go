// Package trigger 事件驱动触发源（TriggerSource Provider 注册表）。
//
// 收敛「外部事件 → 中立 OpsTriggerEvent → 匹配触发器 → 派生诊断 Run」的入站侧：
// 每种来源（告警、webhook、…）实现 TriggerSource 契约并向注册表注册一行即可接入，
// TriggerService 与 RunService 核心链路零改动（开闭原则，仿 ai/application/resource.Provider）。
//
// 依赖方向：本包仅依赖中立事件契约 internal/pkg/event，不依赖 alert/ai，维持无环。
package trigger

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/errorx"
)

// 触发源类型标识（别名到中立事件契约的常量，与 entity.Trigger.SourceType、event.OpsTriggerEvent.Source 同源对齐）
const (
	SourceTypeAlert   = event.OpsTriggerSourceAlert   // 告警联动（alert 引擎通知成功后发布中立事件）
	SourceTypeWebhook = event.OpsTriggerSourceWebhook // 外部监控 webhook 推送
)

// MatchCondition 触发器匹配条件（t_aiops_trigger.match_cond 的 JSON 形态）。
//
// 零值字段 = 不限定该维度；非空字段 = 白名单命中（多维度间 AND）。刻意只保留跨来源通用的
// 匹配维度；来源特有的复杂匹配可在对应 TriggerSource.Match 内自行扩展，不外溢到本契约。
type MatchCondition struct {
	ResourceType string   `json:"resourceType,omitempty"` // 资源类型（machine/db/...）
	ResourceIds  []uint64 `json:"resourceIds,omitempty"`  // 资源 id 白名单
	Severities   []string `json:"severities,omitempty"`   // 严重级白名单（如 P0..P3）
	Metrics      []string `json:"metrics,omitempty"`      // 指标名白名单
}

// ParseMatchCondition 严格解析匹配条件 JSON，是「什么算一个可用的条件」的唯一判据
// （保存校验与派发解析必须同用它，否则会出现「存得进、跑错」的强度差）。
//
// 三种结果各自的语义必须分清：
//   - 空串 → (nil, nil)：留空是产品语义「命中该来源全部事件」，不是解析失败。
//   - 非法 JSON / 含本契约之外的键 → (nil, error)：调用方须拒绝（保存侧）或跳过该触发器
//     （派发侧）。**绝不能退化成零值条件**：零值的每个维度都是「不限定」，MatchByFields 会判
//     全命中，于是键名拼错一次（如 resourceTyp）就等于「该来源每条事件都派生一次 LLM 诊断运行」，
//     告警风暴直接变成配额风暴，且日志里没有任何异常。
//   - 合法且只含已知键 → (*MatchCondition, nil)。
//
// 用 DisallowUnknownFields 而非 lenient Unmarshal：encoding/json 默认忽略未知字段，
// 那正是上面那条静默放大路径的入口。新增匹配维度只需给结构体加字段，本函数零改动。
func ParseMatchCondition(ctx context.Context, raw string) (*MatchCondition, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	dec := json.NewDecoder(strings.NewReader(s))
	dec.DisallowUnknownFields()
	cond := new(MatchCondition)
	if err := dec.Decode(cond); err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerMatchCondInvalid, "err", err.Error())
	}
	// 顶层尾随内容（`{}{}`）Decoder 不报错，需显式判定，否则半个条件会被静默丢弃
	if dec.More() {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerMatchCondTrailing)
	}
	return cond, nil
}

// TriggerSource 触发源契约：每种来源一个实现，负责「归一化 + 匹配」。
type TriggerSource interface {
	// Type 触发源类型标识（alert/webhook/...）
	Type() string
	// Normalize 将来源原始载荷归一化为中立事件。
	// alert：发布侧已归一化，此处做透传/校验补全；webhook：原始 JSON body → 事件。
	Normalize(ctx context.Context, raw any) (*event.OpsTriggerEvent, error)
	// Match 判断归一化事件是否命中触发器的匹配条件（cond 为 nil/零值视为全命中）
	Match(evt *event.OpsTriggerEvent, cond *MatchCondition) bool
}

// Registry 触发源注册表（并发安全；同类型后注册覆盖先注册，对齐 resource.App 覆盖约定）。
//
// 装配期注册、运行期只读：与工具/技能注册表同一「冻结式全局单例」口径，多实例各实例清单一致。
type Registry struct {
	mu      sync.RWMutex
	sources map[string]TriggerSource
}

// NewRegistry 创建空注册表（装配方按需注册来源）
func NewRegistry() *Registry {
	return &Registry{sources: make(map[string]TriggerSource)}
}

// Default 包级默认注册表：内置来源经各自 Install 注册于此，TriggerService 直接消费。
var Default = NewRegistry()

// Register 注册触发源；nil 或空 Type 直接忽略（防脏注册）
func (r *Registry) Register(s TriggerSource) {
	if s == nil || s.Type() == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sources[s.Type()] = s
}

// Get 按类型取触发源
func (r *Registry) Get(sourceType string) (TriggerSource, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.sources[sourceType]
	return s, ok
}

// Types 已注册的触发源类型（稳定排序，供前端下拉/校验）
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.sources))
	for t := range r.sources {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// MatchByFields 通用字段白名单匹配：cond 各维度为空=不限定，非空=须命中（多维度 AND）。
//
// 供各 TriggerSource 复用，保证匹配口径一致——新增匹配维度只改此处，各来源零改动。
// evt 或 cond 为 nil 时视为「无条件」全命中（触发器可不设条件，命中该来源全部事件）。
func MatchByFields(evt *event.OpsTriggerEvent, cond *MatchCondition) bool {
	if evt == nil || cond == nil {
		return true
	}
	if cond.ResourceType != "" && !strings.EqualFold(cond.ResourceType, evt.ResourceType) {
		return false
	}
	if len(cond.ResourceIds) > 0 && !containsUint(cond.ResourceIds, evt.ResourceId) {
		return false
	}
	if len(cond.Severities) > 0 && !containsString(cond.Severities, evt.Severity) {
		return false
	}
	if len(cond.Metrics) > 0 && !containsString(cond.Metrics, evt.Metric) {
		return false
	}
	return true
}

func containsUint(list []uint64, v uint64) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// containsString 大小写不敏感包含判定（严重级/指标名允许配置端书写差异）
func containsString(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(strings.TrimSpace(x), v) {
			return true
		}
	}
	return false
}
