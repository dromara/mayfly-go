package models

import (
	"mayfly-go/base/model"

	"github.com/beego/beego/v2/client/orm"
)

type Db struct {
	model.Model

	Name     string `orm:"column(name)" json:"name"`
	Type     string `orm:"column(type)" json:"type"` // 类型，mysql oracle等
	Host     string `orm:"column(host)" json:"host"`
	Port     int    `orm:"column(port)" json:"port"`
	Network  string `orm:"column(network)" json:"network"`
	Username string `orm:"column(username)" json:"username"`
	Passowrd string `orm:"column(password)" json:"-"`
	Database string `orm:"column(database)" json:"database"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(Db))
}

func GetDbById(id uint64) *Db {
	db := new(Db)
	db.Id = id
	err := model.GetBy(db)
	if err != nil {
		return nil
	}
	return db
}
