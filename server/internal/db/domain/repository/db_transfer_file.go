package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbTransferFile interface {
	base.Repo[*entity.DbTransferFile]

	// 分页获取数据库实例信息列表
	GetPageList(condition *entity.DbTransferFileQuery, orderBy ...string) (*model.PageResult[*entity.DbTransferFile], error)
}
