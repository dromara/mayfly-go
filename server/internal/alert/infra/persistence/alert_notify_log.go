package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"time"
)

type alertNotifyLogRepoImpl struct {
	base.RepoImpl[*entity.AlertNotifyLog]
}

var _ repository.AlertNotifyLog = (*alertNotifyLogRepoImpl)(nil)

func newAlertNotifyLogRepo() repository.AlertNotifyLog {
	return &alertNotifyLogRepoImpl{}
}

func (r *alertNotifyLogRepoImpl) Create(log *entity.AlertNotifyLog) error {
	return global.Db.Create(log).Error
}

func (r *alertNotifyLogRepoImpl) GetByEventId(eventId uint64) ([]*entity.AlertNotifyLog, error) {
	var logs []*entity.AlertNotifyLog
	err := global.Db.Where("event_id = ?", eventId).Order("send_time DESC").Find(&logs).Error
	return logs, err
}

func (r *alertNotifyLogRepoImpl) GetTodayStats() (sent, success, failed int64, err error) {
	today := time.Now().Truncate(24 * time.Hour)

	// 一条 SQL 获取三个统计值
	type todayRow struct {
		Sent    int64
		Success int64
		Failed  int64
	}
	var row todayRow
	err = global.Db.Model(&entity.AlertNotifyLog{}).
		Select("COUNT(*) as sent, SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as success, SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) as failed").
		Where("send_time >= ?", today).
		Scan(&row).Error
	return row.Sent, row.Success, row.Failed, err
}

func (r *alertNotifyLogRepoImpl) GetByChannelStats(since time.Time) (map[string]*entity.ChannelStat, error) {
	var rows []struct {
		ChannelName string
		Sent        int64
		Success     int64
		Failed      int64
	}

	err := global.Db.Model(&entity.AlertNotifyLog{}).
		Select("channel_name, COUNT(*) as sent, SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as success, SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) as failed").
		Where("send_time >= ?", since).
		Group("channel_name").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*entity.ChannelStat, len(rows))
	for _, row := range rows {
		stat := &entity.ChannelStat{
			Sent:    row.Sent,
			Success: row.Success,
			Failed:  row.Failed,
		}
		if row.Sent > 0 {
			stat.Rate = float64(row.Success) / float64(row.Sent) * 100
		}
		result[row.ChannelName] = stat
	}
	return result, nil
}

func (r *alertNotifyLogRepoImpl) GetHourlyTrend(since time.Time) ([]*entity.NotifyTrendPoint, error) {
	// 一条 SQL 完成 24 小时趋势聚合
	var rows []struct {
		HourSlot string
		Sent     int64
		Success  int64
		Failed   int64
	}

	err := global.Db.Model(&entity.AlertNotifyLog{}).
		Select("DATE_FORMAT(send_time, '%Y-%m-%d %H:00') as hour_slot, COUNT(*) as sent, SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as success, SUM(CASE WHEN status = 2 THEN 1 ELSE 0 END) as failed").
		Where("send_time >= ?", since).
		Group("hour_slot").
		Order("hour_slot ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	// 构建 hour -> row 映射，用于填充空小时
	rowMap := make(map[string]*struct {
		HourSlot string
		Sent     int64
		Success  int64
		Failed   int64
	}, len(rows))
	for i := range rows {
		rowMap[rows[i].HourSlot] = &rows[i]
	}

	// 生成完整的 24 小时序列（含空小时填 0）
	result := make([]*entity.NotifyTrendPoint, 0, 24)
	now := time.Now()
	for i := 23; i >= 0; i-- {
		hour := now.Add(-time.Duration(i) * time.Hour).Truncate(time.Hour)
		hourKey := hour.Format("2006-01-02 15:04")
		hourLabel := hour.Format("15:04")

		if row, ok := rowMap[hourKey]; ok {
			result = append(result, &entity.NotifyTrendPoint{
				Hour:    hourLabel,
				Sent:    row.Sent,
				Success: row.Success,
				Failed:  row.Failed,
			})
		} else {
			result = append(result, &entity.NotifyTrendPoint{
				Hour: hourLabel,
			})
		}
	}

	return result, nil
}

func (r *alertNotifyLogRepoImpl) GetFailReasons(limit int) ([]*entity.FailReason, error) {
	var rows []*entity.FailReason
	err := global.Db.Model(&entity.AlertNotifyLog{}).
		Select("error_msg as reason, COUNT(*) as count").
		Where("status = 2 AND error_msg != ''").
		Group("error_msg").
		Order("count DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *alertNotifyLogRepoImpl) GetSendDelay() (avgDelay, maxDelay int64, err error) {
	type delayRow struct {
		AvgDelay float64
		MaxDelay float64
	}
	var row delayRow
	err = global.Db.Raw(`
		SELECT 
			AVG(TIMESTAMPDIFF(SECOND, e.last_trigger_time, l.send_time)) as avg_delay,
			MAX(TIMESTAMPDIFF(SECOND, e.last_trigger_time, l.send_time)) as max_delay
		FROM t_alert_notify_log l
		JOIN t_alert_event e ON l.event_id = e.id
		WHERE l.status = 1 AND e.is_deleted = 0
	`).Scan(&row).Error
	return int64(row.AvgDelay), int64(row.MaxDelay), err
}

func (r *alertNotifyLogRepoImpl) GetSuccessRateSince(since time.Time) (rate float64, err error) {
	type rateRow struct {
		Sent    int64
		Success int64
	}
	var row rateRow
	err = global.Db.Model(&entity.AlertNotifyLog{}).
		Select("COUNT(*) as sent, SUM(CASE WHEN status = 1 THEN 1 ELSE 0 END) as success").
		Where("send_time >= ?", since).
		Scan(&row).Error
	if err != nil {
		return 0, err
	}
	if row.Sent > 0 {
		return float64(row.Success) / float64(row.Sent) * 100, nil
	}
	return 0, nil
}
