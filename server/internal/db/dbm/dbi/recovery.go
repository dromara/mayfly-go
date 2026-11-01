package dbi

import "context"

// 恢复能力探测：回答「这台库现在能不能做时间点恢复（PITR）、能回溯到哪」。
//
// 为什么单开一个能力而不是塞进 Health：PITR 的前提是数据库侧的归档开关
// （MySQL binlog、PostgreSQL WAL 归档），那是**部署形态**的事实，与当前是否健康无关；
// 混进健康检查会让「库很健康但根本做不了 PITR」这件事完全看不见。
//
// 返回结构里 Prepared=false 必须带 Reason：平台自身的备份是逻辑 dump，
// 只能恢复到备份时刻；没有 binlog/WAL 就永远做不了中间点恢复，这一点必须说清，
// 否则运维会以为「平台有备份 = 能恢复到任意时刻」。

// RecoveryReadiness 恢复能力事实
type RecoveryReadiness struct {
	// Prepared 数据库侧是否具备时间点恢复的前提（binlog / WAL 归档已开启且能看到位点）
	Prepared bool `json:"prepared"`
	// Mode 恢复方式：binlog / wal_archive / none
	Mode string `json:"mode"`
	// Reason Prepared=false 时的原因（就绪时为空）
	Reason string `json:"reason,omitempty"`
	// OldestPoint 可回溯的最早位点（MySQL 最老 binlog 文件、PG 最老归档 WAL 名），拿不到为空
	OldestPoint string `json:"oldestPoint,omitempty"`
	// LatestPoint 最新位点（MySQL 当前 binlog:pos、PG 当前 LSN）
	LatestPoint string `json:"latestPoint,omitempty"`
	// Retention 归档保留时长（如 binlog 过期秒数换算的可读值），拿不到为空
	Retention string `json:"retention,omitempty"`
}

// RecoveryInspector 恢复能力探测（可选能力，由方言声明 FeatRecovery 后实现）
type RecoveryInspector interface {
	RecoveryReadiness(ctx context.Context) (*RecoveryReadiness, error)
}
