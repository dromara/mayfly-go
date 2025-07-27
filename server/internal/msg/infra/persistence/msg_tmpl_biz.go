package persistence

import (
	"mayfly-go/internal/msg/domain/entity"
	"mayfly-go/internal/msg/domain/repository"
	"mayfly-go/pkg/base"
)

type msgTmplBizRepoImpl struct {
	base.RepoImpl[*entity.MsgTmplBiz]
}

func newMsgTmplBizRepo() repository.MsgTmplBiz {
	return &msgTmplBizRepoImpl{}
}
