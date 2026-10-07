package repository

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

// RuleSegment 可复用条件组仓储
type RuleSegment interface {
	base.Repo[*entity.RuleSegment]

	GetPageList(condition *entity.RuleSegmentQuery, orderBy ...string) (*model.PageResult[*entity.RuleSegment], error)

	// GetByRef 按标识取条件组，取不到返回 nil。
	// 条件树通过 ref 引用它，因此求值与保存校验都要按标识回查，而不能只按 id
	GetByRef(ref string) *entity.RuleSegment
}
