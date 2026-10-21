package persistence

import (
	"context"
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type alertRuleRepoImpl struct {
	base.RepoImpl[*entity.AlertRule]
}

var _ repository.AlertRule = (*alertRuleRepoImpl)(nil)

func newAlertRuleRepo() repository.AlertRule {
	return &alertRuleRepoImpl{}
}

// alertRuleColumns 整表单覆盖式更新时写入的列。
//
// 必须显式选列：GORM 的 Updates(struct) 会跳过结构体里的所有零值字段，而 priority 的 P0、
// remark 的清空、labels 的清空、notify_config 的移除都是「零值即用户意图」的写法。
// 一旦选列，自动维护的修改审计列也会被一并限制，必须同时点名否则不再写入。
// 新增实体字段时请同步补到本清单，遗漏由 TestAlertRuleColumnsCoverAllBusinessColumns 兜住
var alertRuleColumns = []string{
	"name", "status", "priority", "remark",
	"resource_type", "scope_type", "scope_value",
	"alert_condition", "notify_config",
	"eval_interval", "trigger_count", "recovery_count",
	"labels",
	"update_time", "modifier_id", "modifier",
}

func (r *alertRuleRepoImpl) UpdateAllColumns(ctx context.Context, rule *entity.AlertRule) error {
	return r.UpdateById(ctx, rule, alertRuleColumns...)
}

func (r *alertRuleRepoImpl) GetAlertRuleList(condition *entity.AlertRuleQuery, orderBy ...string) (*model.PageResult[*entity.AlertRule], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Eq("status", condition.Status).
		Eq("resource_type", condition.ResourceType)

	keyword := condition.Keyword
	if keyword != "" {
		keyword = "%" + keyword + "%"
		qd.And("name like ? or remark like ?", keyword, keyword)
	}

	return r.PageByCond(qd, condition.PageParam)
}

func (r *alertRuleRepoImpl) ListEnabled() ([]*entity.AlertRule, error) {
	return r.SelectByCond(model.NewCond().Eq("status", entity.AlertRuleStatusEnable))
}

func (r *alertRuleRepoImpl) GetRuleStats(ctx context.Context) (total, enabled, disabled int64, err error) {
	type row struct {
		Status int8
		Cnt    int64
	}
	var rows []row
	err = gormx.DB(ctx).Model(&entity.AlertRule{}).
		Select("status, COUNT(*) as cnt").
		Where("is_deleted = 0").
		Group("status").
		Scan(&rows).Error
	for _, row := range rows {
		total += row.Cnt
		switch row.Status {
		case entity.AlertRuleStatusEnable:
			enabled = row.Cnt
		case entity.AlertRuleStatusDisable:
			disabled = row.Cnt
		}
	}
	return
}

func (r *alertRuleRepoImpl) CountGroupByPriority(ctx context.Context) (map[string]int64, error) {
	type priorityRow struct {
		Priority entity.AlertPriority
		Cnt      int64
	}
	var rows []priorityRow
	err := gormx.DB(ctx).Model(&entity.AlertRule{}).
		Select("priority, COUNT(*) as cnt").
		Where("is_deleted = 0").
		Group("priority").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	priorityNames := map[entity.AlertPriority]string{
		entity.AlertPriorityCritical: "P0",
		entity.AlertPriorityHigh:     "P1",
		entity.AlertPriorityMedium:   "P2",
		entity.AlertPriorityLow:      "P3",
	}
	result := make(map[string]int64, len(rows))
	for _, row := range rows {
		result[priorityNames[row.Priority]] = row.Cnt
	}
	return result, nil
}

func (r *alertRuleRepoImpl) CountGroupByResourceType(ctx context.Context) (map[int8]int64, error) {
	type rtRow struct {
		ResourceType int8
		Cnt          int64
	}
	var rows []rtRow
	err := gormx.DB(ctx).Model(&entity.AlertRule{}).
		Select("resource_type, COUNT(*) as cnt").
		Where("is_deleted = 0").
		Group("resource_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[int8]int64, len(rows))
	for _, row := range rows {
		result[row.ResourceType] = row.Cnt
	}
	return result, nil
}
