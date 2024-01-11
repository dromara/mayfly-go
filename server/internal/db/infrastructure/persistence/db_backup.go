package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/gormx"
	"slices"
)

var _ repository.DbBackup = (*dbBackupRepoImpl)(nil)

type dbBackupRepoImpl struct {
	dbJobBase[*entity.DbBackup]
}

func NewDbBackupRepo() repository.DbBackup {
	return &dbBackupRepoImpl{}
}

func (d *dbBackupRepoImpl) GetDbNamesWithoutBackup(instanceId uint64, dbNames []string) ([]string, error) {
	var dbNamesWithBackup []string
	query := gormx.NewQuery(d.GetModel()).
		Eq("db_instance_id", instanceId).
		Eq("repeated", true)
	if err := query.GenGdb().Pluck("db_name", &dbNamesWithBackup).Error; err != nil {
		return nil, err
	}
	result := make([]string, 0, len(dbNames))
	for _, name := range dbNames {
		if !slices.Contains(dbNamesWithBackup, name) {
			result = append(result, name)
		}
	}
	return result, nil
}
