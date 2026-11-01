package repository

import (
	"context"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbSQLExec interface {
	base.Repo[*entity.DbSQLExec]

	// 分页获取
	GetPageList(condition *entity.DbSQLExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSQLExec], error)

	// DeleteBefore 物理删除早于 cutoff 的执行记录，返回删除行数。
	//
	// 必须物理删除：软删只是把行标记为不可见，表照样无限增长，保留期形同虚设。
	DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error)
}
