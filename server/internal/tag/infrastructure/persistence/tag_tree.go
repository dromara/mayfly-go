package persistence

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type tagTreeRepoImpl struct {
	base.RepoImpl[*entity.TagTree]
}

func newTagTreeRepo() repository.TagTree {
	return &tagTreeRepoImpl{}
}

func (p *tagTreeRepoImpl) SelectByCondition(condition *entity.TagTreeQuery, toEntity any, orderBy ...string) {
	cond := model.NewCond().Like("name", condition.Name).
		Eq("id", condition.Id).
		In("code", condition.Codes).
		In("code_path", condition.CodePaths).
		In("type", condition.Types).
		OrderByAsc("type").OrderByAsc("code_path")

	if len(condition.CodePathLikes) > 0 {
		codePathLikesAnd := ""
		cocePathLikesParams := make([]any, 0)

		for i, v := range condition.CodePathLikes {
			if i == 0 {
				codePathLikesAnd = codePathLikesAnd + "code_path LIKE ?"
			} else {
				codePathLikesAnd = codePathLikesAnd + " OR code_path LIKE ?"
			}
			cocePathLikesParams = append(cocePathLikesParams, v+"%")
		}

		cond.And(codePathLikesAnd, cocePathLikesParams...)
	}

	p.SelectByCondToAny(cond, toEntity)
}
