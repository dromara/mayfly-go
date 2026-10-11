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
	// BaselineExists 该实例下是否建立过结构基线。
	// 没建过基线时无从判断漂移，与「有基线且结构未变」必须可区分：
	// 否则漂移指标会把「还没配基线」报成 0，被读成「结构很稳定」
	BaselineExists bool

	CheckedAt time.Time
}
