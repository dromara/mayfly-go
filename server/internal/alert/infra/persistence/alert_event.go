package persistence

import (
	"context"
	"fmt"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/taskx"
	"time"
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

	// P0 优先级枚举值为 0，Eq 会忽略零值，故非 nil 时使用 EqStrict 强制生效
	if condition.Priority != nil {
		qd = qd.EqStrict("priority", *condition.Priority)
	}

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("rule_name like ? or resource_name like ? or metric like ?", keyword, keyword, keyword)
	}

	return r.PageByCond(qd, condition.PageParam)
}

func (r *alertEventRepoImpl) FindActive(ctx context.Context, ruleId, resourceId uint64) (*entity.AlertEvent, error) {
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

// activeEventGuard 跨实例互斥「同一(规则, 资源)的活跃事件创建」。
//
// 刻意不用 SELECT ... FOR UPDATE：SQLite 是本仓默认方言，其驱动对行锁子句是**静默丢弃**
// （源码注释 "SQLite3 does not support row-level locking"），所以那条 SQL 在默认部署下
// 根本没加锁，只是看起来安全；行锁的作用域也只到一个数据库实例，多实例各自加锁等于没加。
// 互斥统一交给 Redis 守卫（未配置 Redis 时自动降级为进程内互斥，即单机语义）
var activeEventGuard = &taskx.RunGuard[string]{LockTTL: activeEventLockTTL}

const (
	// activeEventLockTTL 临界区只有一查询一插入，TTL 取小值：实例被杀时最多阻塞这么久
	activeEventLockTTL = 10 * time.Second
	// activeEventLockWait 抢锁失败后的有界等待。等不到说明有实例卡在临界区里，
	// 此时报错交给下一轮评估重试，绝不无限阻塞评估协程
	activeEventLockWait = 2 * time.Second
	activeEventLockPoll = 20 * time.Millisecond
)

func activeEventGuardKey(ruleId, resourceId uint64) string {
	return fmt.Sprintf("alert-active-event:%d:%d", ruleId, resourceId)
}

// acquireActiveEventGuard 在 activeEventLockWait 内轮询抢锁
func acquireActiveEventGuard(key string) bool {
	deadline := time.Now().Add(activeEventLockWait)
	for {
		if activeEventGuard.Acquire(key) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(activeEventLockPoll)
	}
}

// CreateIfNotActive 检查并创建活跃事件，返回 (事件, 是否已存在活跃事件, 错误)。
//
// 查询与插入都在同一把跨实例锁内完成，因此不存在「两边都查到没有活跃事件、于是各插一条」的
// 检查-写入竞态窗口
func (r *alertEventRepoImpl) CreateIfNotActive(ctx context.Context, event *entity.AlertEvent) (*entity.AlertEvent, bool, error) {
	key := activeEventGuardKey(event.RuleId, event.ResourceId)
	if !acquireActiveEventGuard(key) {
		return nil, false, fmt.Errorf("timed out waiting for the active-event guard of rule %d resource %d", event.RuleId, event.ResourceId)
	}
	defer activeEventGuard.Release(key)

	existing, err := r.FindActive(ctx, event.RuleId, event.ResourceId)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return existing, true, nil
	}
	if err := r.Insert(ctx, event); err != nil {
		return nil, false, err
	}
	return event, false, nil
}

func (r *alertEventRepoImpl) CountGroupByStatus(ctx context.Context) (map[entity.AlertEventStatus]int64, error) {
	type row struct {
		Status entity.AlertEventStatus
		Cnt    int64
	}
	var rows []row
	sql := "SELECT status, COUNT(*) as cnt FROM t_alert_event WHERE is_deleted = 0 GROUP BY status"
	err := gormx.DB(ctx).Raw(sql).Scan(&rows).Error
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
func (r *alertEventRepoImpl) FindActiveByRuleId(ctx context.Context, ruleId uint64) ([]*entity.AlertEvent, error) {
	cond := model.NewCond().
		Eq("rule_id", ruleId).
		In("status", []entity.AlertEventStatus{
			entity.AlertEventStatusFiring,
			entity.AlertEventStatusAcknowledged,
		})
	return r.SelectByCond(cond)
}

// CleanupTerminal 删除已终结（Recovered/Closed）超过 retainDays 天的事件
func (r *alertEventRepoImpl) CleanupTerminal(ctx context.Context, retainDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -retainDays)
	result := gormx.DB(ctx).
		Where("status IN ? AND last_trigger_time < ?",
			[]entity.AlertEventStatus{entity.AlertEventStatusRecovered, entity.AlertEventStatusClosed},
			cutoff,
		).
		Delete(&entity.AlertEvent{})
	return result.RowsAffected, result.Error
}

// ---- 概览统计方法 ----

func (r *alertEventRepoImpl) GetAvgRecoveryTime(ctx context.Context) (int64, error) {
	type result struct {
		AvgSeconds float64
	}
	var row result
	err := gormx.DB(ctx).Raw(`SELECT AVG(TIMESTAMPDIFF(SECOND, first_trigger_time, recover_time)) as avg_seconds 
			FROM t_alert_event 
			WHERE status = ? AND recover_time IS NOT NULL AND is_deleted = 0`, entity.AlertEventStatusRecovered).Scan(&row).Error
	return int64(row.AvgSeconds), err
}

func (r *alertEventRepoImpl) CountTodayNotifications(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := gormx.DB(ctx).Model(&entity.AlertEvent{}).
		Where("last_notify_time >= ? AND is_deleted = 0", since).
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) CountPolicyMatched(ctx context.Context) (int64, error) {
	var count int64
	err := gormx.DB(ctx).Model(&entity.AlertEvent{}).
		Where("notify_count > 0 AND is_deleted = 0").
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) CountUnmatched(ctx context.Context) (int64, error) {
	var count int64
	err := gormx.DB(ctx).Model(&entity.AlertEvent{}).
		Where("status IN ? AND notify_count = 0 AND is_deleted = 0",
			[]entity.AlertEventStatus{entity.AlertEventStatusFiring, entity.AlertEventStatusAcknowledged}).
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) CountEscalated(ctx context.Context) (int64, error) {
	var count int64
	err := gormx.DB(ctx).Model(&entity.AlertEvent{}).
		Where("escalation_level > 0 AND is_deleted = 0").
		Count(&count).Error
	return count, err
}

func (r *alertEventRepoImpl) GetMaxEscalationLevel(ctx context.Context) (int64, error) {
	type maxLevel struct {
		MaxLvl int64
	}
	var ml maxLevel
	err := gormx.DB(ctx).Model(&entity.AlertEvent{}).
		Select("MAX(escalation_level) as max_lvl").
		Where("is_deleted = 0").
		Scan(&ml).Error
	return ml.MaxLvl, err
}

func (r *alertEventRepoImpl) GetTopByTriggerCount(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := gormx.DB(ctx).Where("is_deleted = 0").
		Order("trigger_count DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *alertEventRepoImpl) GetTopByDuration(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := gormx.DB(ctx).Where("is_deleted = 0").
		Order("CASE WHEN recover_time IS NULL THEN TIMESTAMPDIFF(SECOND, first_trigger_time, NOW()) ELSE TIMESTAMPDIFF(SECOND, first_trigger_time, recover_time) END DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *alertEventRepoImpl) GetTopByNotifyCount(ctx context.Context, limit int) ([]*entity.AlertEvent, error) {
	var events []*entity.AlertEvent
	err := gormx.DB(ctx).Where("is_deleted = 0").
		Order("notify_count DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
