// Package keyvalue 实现 Redis「数据视角」处理器与注册中心。
//
// 一个视角 = 一种数据类型的读写语义（hash 的 field/value、zset 的 score/member、位图的 bit 偏移……）。
// 所有对 Redis 命令的了解都收敛在对应处理器文件内，对上只暴露统一的读写契约：
//
//	新增一种数据类型 = 新增一个 handler 文件 + init() 里 Register，
//	既有处理器、应用服务、API 与前端组件均零改动（开闭原则）
package keyvalue

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"mayfly-go/internal/redis/domain/entity"

	"github.com/redis/go-redis/v9"
)

// Handler 数据视角的必选契约
type Handler interface {
	// Descriptor 视角自描述信息（能力位、列结构、表单结构、扩展操作）
	Descriptor() *entity.ViewDescriptor

	// Size 该视角下 key 的元素总数，用于展示与分页边界
	Size(ctx context.Context, cmd redis.Cmdable, key string) (int64, error)

	// Load 按条件读取成员分页
	Load(ctx context.Context, cmd redis.Cmdable, q *entity.MemberQuery) (*entity.MemberPage, error)

	// BuildWrite 构造成员写命令（只构造不执行），返回按序执行的命令列表，
	// 每条命令的第一个元素必须是大写命令名，供审批流按命令名匹配写命令
	BuildWrite(ctx context.Context, cmd redis.Cmdable, w *entity.MemberWrite) ([][]any, error)
}

// Detector 可选能力：从存储内容上判断该 key 是否应以本视角展示（如 HyperLogLog 实为特定编码的 string）
type Detector interface {
	// Detect 命中则返回 true；实现者必须自行消化读取失败（返回 false），不得中断主流程
	Detect(ctx context.Context, cmd redis.Cmdable, key string) bool
}

// SupportProbe 可选能力：按当前实例实际支持的命令裁剪描述符（如 hash 字段级过期需 Redis 7.4+）。
//
// 描述符是进程级共享的，实现者必须返回调整后的副本；未实现该接口的视角视为与实例能力无关。
// 裁剪时除了切片字段，指针字段（Form/UpdateForm）也要一并复制，否则改一个视角会串到共享原件
type SupportProbe interface {
	// ProbeSupport 返回该实例上真正可用的描述符（可以就是入参本身）
	ProbeSupport(ctx context.Context, cmd redis.Cmdable, desc *entity.ViewDescriptor) *entity.ViewDescriptor
}

// OpPlanner 可选能力：视角特有的、非成员级的扩展操作（集合运算、位统计、GEO 距离等）。
//
// 与 BuildWrite 一致：只构造命令不执行，写命令的审批流与权限校验因此始终只有一条通路
type OpPlanner interface {
	// PlanOp 构造扩展操作的命令列表，未声明的操作名返回错误
	PlanOp(ctx context.Context, cmd redis.Cmdable, req *entity.OpRequest) ([][]any, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Handler)
)

// Register 注册视角处理器（无状态，全实例共享）
func Register(h Handler) {
	if h == nil {
		panic("keyvalue: register nil handler")
	}
	desc := h.Descriptor()
	if desc == nil || desc.View == "" {
		panic("keyvalue: register handler without view")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exist := registry[desc.View]; exist {
		panic(fmt.Sprintf("keyvalue: duplicate view handler [%s]", desc.View))
	}
	registry[desc.View] = h
}

// Get 获取指定视角处理器，未注册返回 nil
func Get(view string) Handler {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[view]
}

// Descriptors 返回可服务指定原生类型的全部视角描述符（默认视角优先，其后按标识排序，保证前端顺序稳定）
func Descriptors(keyType entity.KeyType) []*entity.ViewDescriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()

	descs := make([]*entity.ViewDescriptor, 0, 4)
	for _, h := range registry {
		if servesType(h, keyType) {
			descs = append(descs, h.Descriptor())
		}
	}
	sort.Slice(descs, func(i, j int) bool {
		if descs[i].Default != descs[j].Default {
			return descs[i].Default
		}
		return descs[i].View < descs[j].View
	})
	return descs
}

// WithSupport 逐个视角应用其处理器的实例能力探测，供按连接下发描述符的入口使用
func WithSupport(ctx context.Context, cmd redis.Cmdable, descs []*entity.ViewDescriptor) []*entity.ViewDescriptor {
	out := make([]*entity.ViewDescriptor, 0, len(descs))
	for _, desc := range descs {
		if probe, ok := Get(desc.View).(SupportProbe); ok {
			out = append(out, probe.ProbeSupport(ctx, cmd, desc))
			continue
		}
		out = append(out, desc)
	}
	return out
}

// AllDescriptors 返回全部已注册视角描述符（按标识排序），供前端渲染类型选择器等全局列表
func AllDescriptors() []*entity.ViewDescriptor {
	registryMu.RLock()
	defer registryMu.RUnlock()

	descs := make([]*entity.ViewDescriptor, 0, len(registry))
	for _, h := range registry {
		descs = append(descs, h.Descriptor())
	}
	sort.Slice(descs, func(i, j int) bool { return descs[i].View < descs[j].View })
	return descs
}

// Resolve 解析实际生效的视角处理器：
//   - 指定了 view 则校验其确实可服务该原生类型（防止越权构造他人视角的写命令）；
//   - 未指定则先按内容自动探测（Detector），探测不到用默认视角
func Resolve(ctx context.Context, cmd redis.Cmdable, keyType entity.KeyType, view, key string) (Handler, error) {
	descs := Descriptors(keyType)
	if len(descs) == 0 {
		return nil, ErrUnsupportedType(ctx, keyType)
	}

	if view != "" {
		h := Get(view)
		if h == nil || !servesType(h, keyType) {
			return nil, ErrUnsupportedView(ctx, view, keyType)
		}
		return h, nil
	}

	for _, desc := range descs {
		h := Get(desc.View)
		if d, ok := h.(Detector); ok && d.Detect(ctx, cmd, key) {
			return h, nil
		}
	}
	return Get(defaultViewOf(descs)), nil
}

func defaultViewOf(descs []*entity.ViewDescriptor) string {
	for _, desc := range descs {
		if desc.Default {
			return desc.View
		}
	}
	return descs[0].View
}

func servesType(h Handler, keyType entity.KeyType) bool {
	for _, t := range h.Descriptor().Types {
		if t == keyType {
			return true
		}
	}
	return false
}

// scanMatch 关键字 → scan MATCH 模式，空关键字匹配全部
func scanMatch(keyword string) string {
	if keyword == "" {
		return "*"
	}
	var escaped strings.Builder
	for _, char := range keyword {
		switch char {
		case '\\', '*', '?', '[', ']', '{', '}':
			// glob 元字符一律用 \ 取消特殊语义：\x 在 Redis 的 glob 里就是字面量 x
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(char)
	}
	return "*" + escaped.String() + "*"
}

// pageCursor 解析游标分页的游标，空串表示游标归零（从头开始）
func pageCursor(cursor string) uint64 {
	if cursor == "" {
		return 0
	}
	var val uint64
	fmt.Sscanf(cursor, "%d", &val)
	return val
}
