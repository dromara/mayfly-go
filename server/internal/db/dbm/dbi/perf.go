package dbi

// 性能诊断（可选能力）：慢查询聚合与锁等待链。
//
// 与 SessionInspector 分开建模而不是塞进它：会话内省在 mysql/pg 都稳定可得，
// 而慢查询依赖 performance_schema / pg_stat_statements 这类「可能没开」的采集源，
// 锁等待链又依赖 8.0 的 data_lock_waits / pg 的 pg_blocking_pids。
// 三者可用性不同，合成一个接口会让「能看会话但看不到慢查询」的实例被迫谎报能力。
//
// 判定口径统一为「事实 + 明确的不可用原因」：拿不到数据源时返回 ErrPerfUnavailable，
// 绝不返回空切片——空结果在界面上与「没有慢查询」不可区分，等于把观测缺口伪装成好消息。

import (
	"context"
	"errors"
	"fmt"
)

// ErrPerfUnavailable 该实例拿不到该类性能数据（采集源未开启/版本不支持）
var ErrPerfUnavailable = errors.New("performance data unavailable")

// NewPerfUnavailable 构造带原因的不可用错误，原因会原样展示给运维
func NewPerfUnavailable(kind, reason string) error {
	return fmt.Errorf("%w: %s (%s)", ErrPerfUnavailable, kind, reason)
}

// perfHintError 在「不可用」之上附带启用步骤。
//
// 步骤属于方言知识（PG 要预加载 + CREATE EXTENSION，MySQL 要开 performance_schema），
// 由方言给出、应用层原样透传——放在应用层按类型拼会让两处方言知识各说各话。
type perfHintError struct {
	err  error
	hint string
}

func (e *perfHintError) Error() string { return e.err.Error() }
func (e *perfHintError) Unwrap() error { return e.err }

// NewPerfUnavailableWithHint 同 NewPerfUnavailable，另带启用步骤
func NewPerfUnavailableWithHint(kind, reason, hint string) error {
	return &perfHintError{err: NewPerfUnavailable(kind, reason), hint: hint}
}

// PerfHint 取方言给出的启用步骤；不是带提示的不可用错误时返回空串
func PerfHint(err error) string {
	var target *perfHintError
	if errors.As(err, &target) {
		return target.hint
	}
	return ""
}

// SlowQuery 一条慢查询聚合（按归一化语句聚合，不是单次执行）
type SlowQuery struct {
	// Digest 归一化语句或摘要标识；方言只能给出文本时与 SQL 相同
	Digest string `json:"digest"`
	// SQL 归一化后的语句文本（可能含 ? 占位符）
	SQL string `json:"sql"`
	// Schema 该语句所属库/模式
	Schema string `json:"schema,omitempty"`
	// Calls 统计窗口内的执行次数
	Calls int64 `json:"calls"`
	// AvgMs / MaxMs / TotalMs 平均、最大、累计耗时（毫秒）
	AvgMs   int64 `json:"avgMs"`
	MaxMs   int64 `json:"maxMs"`
	TotalMs int64 `json:"totalMs"`
	// RowsAvg 平均返回/影响行数；未知为 0
	RowsAvg int64 `json:"rowsAvg"`
}

// LockWait 一条阻塞关系：某个会话在等另一个会话持有的锁。
//
// 按「等待方 → 阻塞方」成对给出而不是只给被阻塞会话的数量：
// 处置时要先找到源头（是谁拿着锁不放），只有计数帮不上忙。
type LockWait struct {
	WaitSessionId string `json:"waitSessionId"`
	WaitUser      string `json:"waitUser,omitempty"`
	WaitDb        string `json:"waitDb,omitempty"`
	WaitQuery     string `json:"waitQuery,omitempty"`
	// WaitMs 已等待毫秒数；方言给不出时为 0
	WaitMs int64 `json:"waitMs"`
	// WaitTrx / BlockTrx 事务标识（mysql 为 innodb 事务 id，pg 为 xid），未知为空
	WaitTrx  string `json:"waitTrx,omitempty"`
	BlockTrx string `json:"blockTrx,omitempty"`

	BlockSessionId string `json:"blockSessionId"`
	BlockUser      string `json:"blockUser,omitempty"`
	BlockDb        string `json:"blockDb,omitempty"`
	BlockQuery     string `json:"blockQuery,omitempty"`

	// LockType 锁对象描述（表名/索引名 + 锁模式），各方言自由文本
	LockType string `json:"lockType,omitempty"`
}

// SlowQueryInspector 慢查询聚合能力
type SlowQueryInspector interface {
	// SlowQueries 返回按累计耗时降序的前 limit 条聚合
	SlowQueries(ctx context.Context, limit int) ([]*SlowQuery, error)
}

// LockWaitInspector 锁等待链能力
type LockWaitInspector interface {
	LockWaits(ctx context.Context) ([]*LockWait, error)
}
