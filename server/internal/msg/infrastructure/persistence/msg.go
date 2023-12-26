package persistence

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type msgRepoImpl struct {
	base.RepoImpl[*entity.Msg]
}

func newMsgRepo() repository.Msg {
	return &msgRepoImpl{base.RepoImpl[*entity.Msg]{M: new(entity.Msg)}}
}

func (m *msgRepoImpl) GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
