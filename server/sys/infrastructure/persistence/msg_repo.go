package persistence

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/sys/domain/entity"
	"mayfly-go/server/sys/domain/repository"
)

type msgRepo struct{}

var MsgDao repository.Msg = &msgRepo{}

func (m *msgRepo) GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity)
}

func (m *msgRepo) Insert(account *entity.Msg) {
	biz.ErrIsNil(model.Insert(account), "新增消息记录失败")
}
