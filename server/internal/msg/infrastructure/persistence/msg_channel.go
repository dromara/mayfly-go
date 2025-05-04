package persistence

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type msgChannelRepoImpl struct {
	base.RepoImpl[*entity.MsgChannel]
}

func newMsgChannelRepo() repository.MsgChannel {
	return &msgChannelRepoImpl{}
}

func (m *msgChannelRepoImpl) GetPageList(condition *entity.MsgChannel, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	pd := model.NewCond().
		Eq("id", condition.Id).
		Like("code", condition.Code).
		OrderBy(orderBy...)
	return m.PageByCondToAny(pd, pageParam, toEntity)
}
