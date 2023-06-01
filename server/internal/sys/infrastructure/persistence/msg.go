package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type msgRepoImpl struct{}

func newMsgRepo() repository.Msg {
	return new(msgRepoImpl)
}

func (m *msgRepoImpl) GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity)
}

func (m *msgRepoImpl) Insert(account *entity.Msg) {
	biz.ErrIsNil(model.Insert(account), "新增消息记录失败")
}
