package repository

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/pkg/base"
	"time"
)

// AlertNotifyLog 通知日志仓库接口
type AlertNotifyLog interface {
	base.Repo[*entity.AlertNotifyLog]

	// Create 创建通知日志
	Create(ctx context.Context, log *entity.AlertNotifyLog) error

	// GetByEventId 获取事件的通知日志
	GetByEventId(ctx context.Context, eventId uint64) ([]*entity.AlertNotifyLog, error)

	// GetTodayStats 获取今日统计（发送数、成功数、失败数）
	GetTodayStats(ctx context.Context) (sent, success, failed int64, err error)

	// GetByChannelStats 按渠道统计今日发送情况
	GetByChannelStats(ctx context.Context, since time.Time) (map[string]*entity.ChannelStat, error)

	// GetHourlyTrend 批量获取最近 N 小时的趋势数据（一条 SQL 完成）
	GetHourlyTrend(ctx context.Context, since time.Time) ([]*entity.NotifyTrendPoint, error)

	// GetFailReasons 获取失败原因统计（按次数降序，限制条数）
	GetFailReasons(ctx context.Context, limit int) ([]*entity.FailReason, error)

	// GetSendDelay 获取发送延迟统计（平均/最大）
	GetSendDelay(ctx context.Context) (avgDelay, maxDelay int64, err error)

	// GetSuccessRateSince 获取指定时间以来的成功率
	GetSuccessRateSince(ctx context.Context, since time.Time) (rate float64, err error)
}
