package persistence

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type msgRepoImpl struct {
	base.RepoImpl[*entity.Msg]
}

func newMsgRepo() repository.Msg {
	return &msgRepoImpl{}
}

func (m *msgRepoImpl) GetPageList(condition *entity.Msg, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Msg], error) {
	pd := model.NewModelCond(condition).OrderBy(orderBy...)
	return m.PageByCond(pd, pageParam)
}
