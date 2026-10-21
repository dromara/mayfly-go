package persistence

import (
	"context"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// 整表单覆盖式保存时更新分支写入的列。
//
// 必须显式选列：GORM 的 Updates(struct) 会跳过结构体里的所有零值字段，而 status 的停用、
// rule_id 的解绑、db_name/table_name 的放宽、params/remark 的清空都是「零值即用户意图」的写法。
// 一旦选列，自动维护的修改审计列也会被一并限制，必须同时点名否则不再写入。
// 新增实体字段时请同步补到清单，遗漏由 TestMaskColumnsCoverAllBusinessColumns 兜住
var (
	maskRuleColumns = []string{
		"name", "match_type", "pattern", "algorithm", "params", "status", "weight", "remark",
		"update_time", "modifier_id", "modifier",
	}
	maskColumnColumns = []string{
		"instance_id", "db_name", "table_name", "column_name", "action", "rule_id", "algorithm", "params", "remark",
		"update_time", "modifier_id", "modifier",
	}
)

type maskRuleRepoImpl struct {
	base.RepoImpl[*entity.DbMaskRule]
}

var _ repository.MaskRule = (*maskRuleRepoImpl)(nil)

func newMaskRuleRepo() repository.MaskRule {
	return &maskRuleRepoImpl{}
}

func (m *maskRuleRepoImpl) SaveAllColumns(ctx context.Context, rule *entity.DbMaskRule) error {
	if rule.Id == 0 {
		return m.Insert(ctx, rule)
	}
	return m.UpdateById(ctx, rule, maskRuleColumns...)
}

// GetPageList 分页获取规则
func (m *maskRuleRepoImpl) GetPageList(condition *entity.DbMaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error) {
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

func (m *maskColumnRepoImpl) SaveAllColumns(ctx context.Context, tag *entity.DbMaskColumn) error {
	if tag.Id == 0 {
		return m.Insert(ctx, tag)
	}
	return m.UpdateById(ctx, tag, maskColumnColumns...)
}

// GetPageList 分页获取列标签
func (m *maskColumnRepoImpl) GetPageList(condition *entity.DbMaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error) {
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
