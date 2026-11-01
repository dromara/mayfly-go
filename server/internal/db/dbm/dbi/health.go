package dbi

import "time"

// DbHealth 一次数据库实例健康采样的事实集合（告警评估器的数据源）。
//
// 刻意不引入独立的指标采集底座：实例可达性用现成的 Ping，会话事实复用 SessionInspector，
// 因此新增 DB 告警不需要「先建一套时序采集」。没有采集能力就不谎报指标：
// SessionsAvailable 与 PingErr 分开表达，正是为了区分「库连不上」「看不了会话」「会话为 0」三种事实
type DbHealth struct {
	// InstanceId 被采样的数据库实例
	InstanceId uint64

	// PingErr 建立/复用连接并 Ping 的失败原因；nil 表示可达
	PingErr error

	// Sessions 活动会话；仅当 SessionsAvailable 为 true 时有效
	Sessions []*SessionInfo

	// SessionsAvailable 该实例方言是否提供会话内省能力。
	// false 表示「看不了」，与 Sessions 为空（确实没有会话）必须可区分，
	// 否则方言能力缺失会被上层当成「实例健康」，把观测缺口伪装成好消息
	SessionsAvailable bool

	// DriftedTables 该实例下有基线且当前报为漂移的表数
	DriftedTables int
	// TLSCertDaysLeft 实例已配置证书中最早到期的剩余天数；nil 表示没配证书或证书解析不出。
	//
	// 用指针而非 0：0 会被读成「今天到期」，而事实可能是「根本没配 TLS」。
	// 负数表示已过期（仍如实给出，便于规则用同一指标同时表达「将过期」与「已过期」）。
	TLSCertDaysLeft *int

	// BaselineExists 该实例下是否建立过结构基线。
	// 没建过基线时无从判断漂移，与「有基线且结构未变」必须可区分：
	// 否则漂移指标会把「还没配基线」报成 0，被读成「结构很稳定」
	BaselineExists bool

	// RecoveryPrepared 该实例当前是否具备时间点恢复（PITR）前提：binlog / WAL 归档可用。
	//
	// 指针而非 bool：nil 表示「这个方言不提供探测」或「还没探测」，
	// 与「探测了但没开归档」必须可区分——后者才是需要处置的事实。
	RecoveryPrepared *bool
	// RecoveryReason RecoveryPrepared=false 时的原因（如 binlog 未开启），就绪时为空
	RecoveryReason string

	CheckedAt time.Time
}
