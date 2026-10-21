package repository

import (
	"context"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// MaskRule 脱敏规则仓库接口
type MaskRule interface {
	base.Repo[*entity.DbMaskRule]

	// GetPageList 分页获取规则
	GetPageList(condition *entity.DbMaskRuleQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskRule], error)

	// ListEnabled 获取所有启用的规则
	ListEnabled() ([]*entity.DbMaskRule, error)

	// SaveAllColumns 整表单覆盖式保存：新建插入，更新时显式点名全部业务列。
	//
	// 保存脱敏规则必须走它而不是 base.Repo.Save：后者底层是 Updates(struct)，GORM 会跳过
	// 结构体里的所有零值字段，而「停用」的枚举值恰好是 0，隐式更新会让规则永远停不掉、
	// 接口却照样返回成功
	SaveAllColumns(ctx context.Context, rule *entity.DbMaskRule) error
}

// MaskColumn 脱敏列标签仓库接口
type MaskColumn interface {
	base.Repo[*entity.DbMaskColumn]

	// GetPageList 分页获取列标签
	GetPageList(condition *entity.DbMaskColumnQuery, orderBy ...string) (*model.PageResult[*entity.DbMaskColumn], error)

	// ListByInstance 获取指定实例的所有列标签
	ListByInstance(instanceId uint64) ([]*entity.DbMaskColumn, error)

	// SaveAllColumns 整表单覆盖式保存，语义与 MaskRule.SaveAllColumns 一致。
	//
	// 列标签的零值同样是用户意图：从「绑定规则」改回「直接指定算法」时 ruleId 要清 0，
	// 把限定范围从某个库/表放宽到全部时 db_name、table_name 要清成空串
	SaveAllColumns(ctx context.Context, tag *entity.DbMaskColumn) error
}
