package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type Db interface {
	base.Repo[*entity.Db]

	// GetPageList 分页获取数据信息列表
	GetPageList(condition *entity.DbQuery, orderBy ...string) (*model.PageResult[*entity.DbListPO], error)
}
