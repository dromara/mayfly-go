package persistence

import (
	"mayfly-go/internal/alert/domain/entity"
	"mayfly-go/internal/alert/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
)

type alertRuleRepoImpl struct {
	base.RepoImpl[*entity.AlertRule]
}

var _ repository.AlertRule = (*alertRuleRepoImpl)(nil)

func newAlertRuleRepo() repository.AlertRule {
	return &alertRuleRepoImpl{}
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

func (r *alertRuleRepoImpl) GetRuleStats() (total, enabled, disabled int64, err error) {
	type row struct {
		Status int8
		Cnt    int64
	}
	var rows []row
	err = global.Db.Model(&entity.AlertRule{}).
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

func (r *alertRuleRepoImpl) CountGroupByPriority() (map[string]int64, error) {
	type priorityRow struct {
		Priority entity.AlertPriority
		Cnt      int64
	}
	var rows []priorityRow
	err := global.Db.Model(&entity.AlertRule{}).
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

func (r *alertRuleRepoImpl) CountGroupByResourceType() (map[int8]int64, error) {
	type rtRow struct {
		ResourceType int8
		Cnt          int64
	}
	var rows []rtRow
	err := global.Db.Model(&entity.AlertRule{}).
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
