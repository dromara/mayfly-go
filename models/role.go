package models

import (
	"mayfly-go/base/model"

	"github.com/beego/beego/v2/client/orm"
)

type Role struct {
	model.Model

	Name string `orm:"column(name)" json:"username"`
	//AccountId int64  `orm:"column(account_id)`

	Account *Account `orm:"rel(fk);index"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(Role))
}
