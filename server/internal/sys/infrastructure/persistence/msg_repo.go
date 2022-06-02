package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type msgRepo struct{}

var MsgDao repository.Msg = &msgRepo{}

func (m *msgRepo) GetPageList(condition *entity.Msg, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity)
}

func (m *msgRepo) Insert(account *entity.Msg) {
	biz.ErrIsNil(model.Insert(account), "新增消息记录失败")
}
