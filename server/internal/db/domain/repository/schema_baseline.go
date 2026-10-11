package repository

import (
	"context"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
)

// SchemaBaseline 表结构基线仓库接口
type SchemaBaseline interface {
	base.Repo[*entity.DbSchemaBaseline]

	// ListByDb 取某个库的基线行（按表名小写存）
	ListByDb(dbId uint64, dbName string) ([]*entity.DbSchemaBaseline, error)

	// ListDbKeys 列出所有已建立基线的库（定时比对遍历用）
	ListDbKeys() ([]*entity.DbSchemaBaseline, error)

	// DriftStat 返回该实例下的基线行数与其中处于漂移状态的行数。
	//
	// 两个数一起给而不是只给漂移数：只给漂移数就分不开「有基线且都没变」与
	// 「根本没建过基线」，后者在告警指标上必须是「无从判断」而不是 0
	DriftStat(instanceId uint64) (baselineRows int64, driftedRows int64, err error)

	// DeleteByDb 删除某个库的全部基线
	DeleteByDb(ctx context.Context, dbId uint64, dbName string) error
}
