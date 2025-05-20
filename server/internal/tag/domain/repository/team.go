package repository

import (
	"mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Team interface {
	base.Repo[*entity.Team]

	GetPageList(condition *entity.TeamQuery, orderBy ...string) (*model.PageResult[*entity.Team], error)
}
