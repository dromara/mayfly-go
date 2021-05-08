package service

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/models"
)

type dbService struct {
}

func (d *dbService) GetDbById(id uint64) *models.Db {
	db := new(models.Db)
	db.Id = id
	err := model.GetBy(db)
	if err != nil {
		return nil
	}
	return db
}
