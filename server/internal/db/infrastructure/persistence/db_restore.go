package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/gormx"
	"slices"
)

var _ repository.DbRestore = (*dbRestoreRepoImpl)(nil)

type dbRestoreRepoImpl struct {
	dbJobBase[*entity.DbRestore]
}

func NewDbRestoreRepo() repository.DbRestore {
	return &dbRestoreRepoImpl{}
}

func (d *dbRestoreRepoImpl) GetDbNamesWithoutRestore(instanceId uint64, dbNames []string) ([]string, error) {
	var dbNamesWithRestore []string
	query := gormx.NewQuery(d.GetModel()).
		Eq("db_instance_id", instanceId).
		Eq("repeated", true)
	if err := query.GenGdb().Pluck("db_name", &dbNamesWithRestore).Error; err != nil {
		return nil, err
	}
	result := make([]string, 0, len(dbNames))
	for _, name := range dbNames {
		if !slices.Contains(dbNamesWithRestore, name) {
			result = append(result, name)
		}
	}
	return result, nil
}
