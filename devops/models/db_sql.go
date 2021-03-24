package models

import (
	"mayfly-go/base/model"

	"github.com/beego/beego/v2/client/orm"
)

type DbSql struct {
	model.Model `orm:"-"`

	DbId uint64 `orm:"column(db_id)" json:"db_id"`
	Type int    `orm:"column(type)" json:"type"` // 类型
	Sql  string `orm:"column(sql)" json:"sql"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(DbSql))
}

func GetDbSqlByUser(userId uint64, dbId uint64, sqlType int) *Db {
	db := new(Db)
	query := model.QuerySetter(db).Filter("CreatorId", userId).Filter("DbId", dbId).Filter("Type", sqlType)
	model.GetOne(query, db, db)
	return db
}
