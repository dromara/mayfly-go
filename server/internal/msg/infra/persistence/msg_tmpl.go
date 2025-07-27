package persistence

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type msgTmplRepoImpl struct {
	base.RepoImpl[*entity.MsgTmpl]
}

func newMsgTmplRepo() repository.MsgTmpl {
	return &msgTmplRepoImpl{}
}

func (m *msgTmplRepoImpl) GetPageList(condition *entity.MsgTmpl, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MsgTmpl], error) {
	pd := model.NewCond().
		Eq("id", condition.Id).
		Like("code", condition.Code).
		OrderBy(orderBy...)
	return m.PageByCond(pd, pageParam)
}

type msgTmplChannelRepoImpl struct {
	base.RepoImpl[*entity.MsgTmplChannel]
}

func newMsgTmplChannelRepo() repository.MsgTmplChannel {
	return &msgTmplChannelRepoImpl{}
}
