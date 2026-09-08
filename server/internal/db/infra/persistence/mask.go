package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type maskRuleRepoImpl struct {
	base.RepoImpl[*entity.DbMaskRule]
}

var _ repository.MaskRule = (*maskRuleRepoImpl)(nil)

func newMaskRuleRepo() repository.MaskRule {
	return &maskRuleRepoImpl{}
}

// GetPageList 分页获取规则
func (m *maskRuleRepoImpl) GetPageList(condition *entity.MaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error) {
	qd := model.NewCond().
		Like("name", condition.Name).
		Like("pattern", condition.Keyword).
		Eq("status", condition.Status).
		OrderBy(append([]string{"weight DESC"}, orderBy...)...)
	return m.PageByCond(qd, condition.PageParam)
}

// ListEnabled 获取所有启用的规则
func (m *maskRuleRepoImpl) ListEnabled() ([]*entity.DbMaskRule, error) {
	return m.SelectByCond(model.NewModelCond(&entity.DbMaskRule{Status: entity.MaskRuleStatusEnabled}))
}

type maskColumnRepoImpl struct {
	base.RepoImpl[*entity.DbMaskColumn]
}

var _ repository.MaskColumn = (*maskColumnRepoImpl)(nil)

func newMaskColumnRepo() repository.MaskColumn {
	return &maskColumnRepoImpl{}
}

// GetPageList 分页获取列标签
func (m *maskColumnRepoImpl) GetPageList(condition *entity.MaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error) {
	qd := model.NewCond().
		Eq("instance_id", condition.InstanceId).
		Eq("db_name", condition.DbName).
		Eq("table_name", condition.TableName).
		Eq("column_name", condition.ColumnName).
		OrderBy(orderBy...)
	return m.PageByCond(qd, condition.PageParam)
}

// ListByInstance 获取指定实例的所有列标签
func (m *maskColumnRepoImpl) ListByInstance(instanceId uint64) ([]*entity.DbMaskColumn, error) {
	return m.SelectByCond(model.NewCond().Eq("instance_id", instanceId))
}
