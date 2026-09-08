package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// MaskRule 脱敏规则仓库接口
type MaskRule interface {
	base.Repo[*entity.DbMaskRule]

	// GetPageList 分页获取规则
	GetPageList(condition *entity.MaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error)

	// ListEnabled 获取所有启用的规则
	ListEnabled() ([]*entity.DbMaskRule, error)
}

// MaskColumn 脱敏列标签仓库接口
type MaskColumn interface {
	base.Repo[*entity.DbMaskColumn]

	// GetPageList 分页获取列标签
	GetPageList(condition *entity.MaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error)

	// ListByInstance 获取指定实例的所有列标签
	ListByInstance(instanceId uint64) ([]*entity.DbMaskColumn, error)
}
