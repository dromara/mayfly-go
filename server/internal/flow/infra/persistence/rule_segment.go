package persistence

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type ruleSegmentRepoImpl struct {
	base.RepoImpl[*entity.RuleSegment]
}

func newRuleSegmentRepo() repository.RuleSegment {
	return &ruleSegmentRepoImpl{}
}

func (r *ruleSegmentRepoImpl) GetPageList(condition *entity.RuleSegmentQuery, orderBy ...string) (*model.PageResult[*entity.RuleSegment], error) {
	qd := model.NewCond().
		Eq("biz_type", condition.BizType).
		Like("name", condition.Name).
		OrderBy(orderBy...)
	return r.PageByCond(qd, condition.PageParam)
}

func (r *ruleSegmentRepoImpl) GetByRef(ref string) *entity.RuleSegment {
	// 以实体自身作为查询条件：Ref 是唯一列，等值命中即可，不必再拼条件对象
	segment := &entity.RuleSegment{Ref: ref}
	if err := r.GetByCond(segment); err != nil {
		return nil
	}
	return segment
}
