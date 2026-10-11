package dbi

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// SessionInfo 数据库侧一条会话/执行的事实。
//
// 只放「各方言都确实拿得到」的字段，且一律为原始事实而非派生结论：
// 是否在等待锁由方言按自身系统表的判据填入 LockWait（判据差异极大：mysql 看 STATE 文本、
// pg 看 wait_event_type，clickhouse 无锁概念），上层不做字符串猜测，避免同一条判据在多处漂移。
type SessionInfo struct {
	// Id 会话/查询标识，用于 CancelSession；各方言形态不同（mysql 的连接 id、pg 的 pid、文本型 query id）
	Id string `json:"id"`
	// Db 该会话当前所在库/模式，未知时为空
	Db string `json:"db"`
	// User 建立该会话的数据库账号
	User string `json:"user"`
	// StartedAt 会话开始（或当前语句开始）时间；nil 表示该方言只给出「已执行多久」而没有时间点。
	//
	// 用指针而不是零值时间：`0001-01-01T00:00:00Z` 序列化出去后看起来像个真实时间戳，
	// 一旦被界面或导出取用就成了「会话从公元一年开始」的假信息
	StartedAt *time.Time `json:"startedAt,omitempty"`
	// ElapsedMs 当前语句已执行毫秒数；方言只能给出秒级精度时按秒换算
	ElapsedMs int64 `json:"elapsedMs"`
	// State 方言自己的会话状态（如 mysql 的 Sleep/Query、pg 的 active/idle in transaction）
	State string `json:"state"`
	// Query 该会话正在/最后执行的语句文本
	Query string `json:"query"`
	// LockWait 是否正处于等锁状态（判据由方言给出，见类型注释）
	LockWait bool `json:"lockWait"`
}

// SessionInspector 可选能力契约：由方言实现，列出实例上的活动会话并支持取消指定会话。
//
// 与 TableSearcher/MetadataNavigator 同属「可选能力 + 运行期类型断言」一族：
// 未实现的方言其 MetadataReader 代理返回 ErrUnsupportedKind，能力协商端点也不吐该 feature，
// 因此前端可以据此隐藏入口，不存在「显示了按钮却报不支持」的谎报路径。
type SessionInspector interface {
	// ListSessions 返回该实例当前的活动会话（实现应排除自身连接，避免列表里混入这条查询本身）
	ListSessions(ctx context.Context) ([]*SessionInfo, error)

	// CancelSession 请求数据库取消指定会话当前语句。
	//
	// id 来自 ListSessions 的返回值：实现必须先按本方言的标识形态校验再拼进语句
	// （KILL 一类命令大多不支持绑定参数），不合法的 id 一律拒绝而不是「执行报错」
	CancelSession(ctx context.Context, id string) error
}

// ErrInvalidSessionId 会话标识不符合该方言的形态，未向数据库发出任何取消请求
var ErrInvalidSessionId = errors.New("invalid session id")

// NewInvalidSessionIdErr 构造带上下文的非法会话标识错误（错误链保留 ErrInvalidSessionId 供上层判定）
func NewInvalidSessionIdErr(dbType, id string) error {
	return fmt.Errorf("%w: type %s, id %q", ErrInvalidSessionId, dbType, trimSessionIdForLog(id))
}

// trimSessionIdForLog 会话标识来自数据库系统表，但取消入口的 id 由请求带入，
// 长度不受控，直接拼进错误信息会让日志被超长字符串灌满
func trimSessionIdForLog(id string) string {
	const maxLogIdLen = 64
	if len(id) <= maxLogIdLen {
		return id
	}
	return id[:maxLogIdLen] + "..."
}

// ListSessions 代理方言的 SessionInspector 能力；方言未实现则返回 ErrUnsupportedKind。
func (m *MetadataReader) ListSessions(ctx context.Context) ([]*SessionInfo, error) {
	si, ok := m.provider.(SessionInspector)
	if !ok {
		return nil, fmt.Errorf("%w: sessions", ErrUnsupportedKind)
	}
	return si.ListSessions(ctx)
}

// CancelSession 代理方言的会话取消能力；方言未实现则返回 ErrUnsupportedKind。
//
// 只透传不判权限：谁能取消谁的会话属资源权限范畴，由上层（api/application）负责
func (m *MetadataReader) CancelSession(ctx context.Context, id string) error {
	si, ok := m.provider.(SessionInspector)
	if !ok {
		return fmt.Errorf("%w: sessions", ErrUnsupportedKind)
	}
	return si.CancelSession(ctx, id)
}
