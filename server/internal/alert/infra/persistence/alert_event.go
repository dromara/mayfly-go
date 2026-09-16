package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"time"

	"gorm.io/gorm/clause"
)

type alertEventRepoImpl struct {
	base.RepoImpl[*entity.AlertEvent]
}

var _ repository.AlertEvent = (*alertEventRepoImpl)(nil)

func newAlertEventRepo() repository.AlertEvent {
	return &alertEventRepoImpl{}
}

func (r *alertEventRepoImpl) GetAlertEventList(condition *entity.AlertEventQuery, orderBy ...string) (*model.PageResult[*entity.AlertEvent], error) {
	qd := model.NewCond().
		Eq("rule_id", condition.RuleId).
		Eq("resource_type", condition.ResourceType).
		Eq("resource_id", condition.ResourceId).
		Eq("status", condition.Status)

	// P0 优先级枚举值为 0，Eq 会忽略零值，故非 nil 时使用 Eq0 强制生效
	if condition.Priority != nil {
		qd = qd.Eq0("priority", *condition.Priority)
	}

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("rule_name like ? or resource_name like ? or metric like ?", keyword, keyword, keyword)
	}

	return r.PageByCond(qd, condition.PageParam)
}

func (r *alertEventRepoImpl) FindActive(ruleId, resourceId uint64) (*entity.AlertEvent, error) {
	cond := model.NewCond().
		Eq("rule_id", ruleId).
		Eq("resource_id", resourceId).
		In("status", []entity.AlertEventStatus{
			entity.AlertEventStatusFiring,
			entity.AlertEventStatusAcknowledged,
		})
	events, err := r.SelectByCond(cond)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}
	return events[0], nil
}

// CreateIfNotActive 在事务中原子地检查并创建活跃事件，防止并发创建重复事件。
// 使用 SELECT ... FOR UPDATE 行级锁，多实例部署时保证并发安全。
func (r *alertEventRepoImpl) CreateIfNotActive(event *entity.AlertEvent) (*entity.AlertEvent, bool, error) {
	var existing entity.AlertEvent
	tx := global.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 使用 FOR UPDATE 行级锁，防止多实例并发创建重复事件
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("rule_id = ? AND resource_id = ? AND status IN ? AND is_deleted = 0",
			event.RuleId, event.ResourceId,
			[]entity.AlertEventStatus{entity.AlertEventStatusFiring, entity.AlertEventStatusAcknowledged},
		).First(&existing).Error

	if err == nil {
		// 已存在活跃事件
		tx.Rollback()
		return &existing, true, nil
	}

	// 不存在，创建新事件
	if err := tx.Create(event).Error; err != nil {
		tx.Rollback()
		return nil, false, err
	}
	tx.Commit()
	return event, false, nil
}

func (r *alertEventRepoImpl) CountGroupByStatus() (map[entity.AlertEventStatus]int64, error) {
	type row struct {
		Status entity.AlertEventStatus
		Cnt    int64
	}
	var rows []row
	sql := "SELECT status, COUNT(*) as cnt FROM t_alert_event WHERE is_deleted = 0 GROUP BY status"
	err := global.Db.Raw(sql).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[entity.AlertEventStatus]int64)
	for _, r := range rows {
		result[r.Status] = r.Cnt
	}
	return result, nil
}

// FindActiveByRuleId 查找指定规则下所有活跃事件（Firing/Acknowledged）
func (r *alertEventRepoImpl) FindActiveByRuleId(ruleId uint64) ([]*entity.AlertEvent, error) {
	cond := model.NewCond().
		Eq("rule_id", ruleId).
		In("status", []entity.AlertEventStatus{
			entity.AlertEventStatusFiring,
			entity.AlertEventStatusAcknowledged,
		})
	return r.SelectByCond(cond)
}

// CleanupTerminal 删除已终结（Recovered/Closed）超过 retainDays 天的事件
func (r *alertEventRepoImpl) CleanupTerminal(retainDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retainDays)
	result := global.Db.
		Where("status IN ? AND last_trigger_time < ?",
			[]entity.AlertEventStatus{entity.AlertEventStatusRecovered, entity.AlertEventStatusClosed},
			cutoff,
		).
		Delete(&entity.AlertEvent{})
	return result.RowsAffected, result.Error
}

// ---- 概览统计方法 ----

func (r *alertEventRepoImpl) GetAvgRecoveryTime() (int64, error) {
	type result struct {
		AvgSeconds float64
	}
	var row result
	err := global.Db.Raw(`SELECT AVG(TIMESTAMPDIFF(SECOND, first_trigger_time, recover_time)) as avg_seconds 
			FROM t_alert_event 
			WHERE status = ? AND recover_time IS NOT NULL AND is_deleted = 0`, entity.AlertEventStatusRecovered).Scan(&row).Error
	return int64(row.AvgSeconds), err
}

func (r *alertEventRepoImpl) CountTodayNotifications(since time.Time) (int64, error) {
	var count int64
	err := global.Db.Model(&entity.AlertEvent{}).
		Where("last_notify_time >= ? AND is_deleted = 0", since).
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) CountPolicyMatched() (int64, error) {
	var count int64
	err := global.Db.Model(&entity.AlertEvent{}).
		Where("notify_count > 0 AND is_deleted = 0").
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) CountUnmatched() (int64, error) {
	var count int64
	err := global.Db.Model(&entity.AlertEvent{}).
		Where("status IN ? AND notify_count = 0 AND is_deleted = 0",
			[]entity.AlertEventStatus{entity.AlertEventStatusFiring, entity.AlertEventStatusAcknowledged}).
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) CountEscalated() (int64, error) {
	var count int64
	err := global.Db.Model(&entity.AlertEvent{}).
		Where("escalation_level > 0 AND is_deleted = 0").
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) GetMaxEscalationLevel() (int64, error) {
	type maxLevel struct {
		MaxLvl int64
	}
	var ml maxLevel
	err := global.Db.Model(&entity.AlertEvent{}).
		Select("MAX(escalation_level) as max_lvl").
		Where("is_deleted = 0").
		Scan(&ml).Error
	return ml.MaxLvl, err
}

func (r *alertEventRepoImpl) GetTopByTriggerCount(limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := global.Db.Where("is_deleted = 0").
		Order("trigger_count DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *alertEventRepoImpl) GetTopByDuration(limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := global.Db.Where("is_deleted = 0").
		Order("CASE WHEN recover_time IS NULL THEN TIMESTAMPDIFF(SECOND, first_trigger_time, NOW()) ELSE TIMESTAMPDIFF(SECOND, first_trigger_time, recover_time) END DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *alertEventRepoImpl) GetTopByNotifyCount(limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := global.Db.Where("is_deleted = 0").
		Order("notify_count DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
