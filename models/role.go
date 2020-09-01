package models

import (
	"github.com/astaxie/beego/orm"
	"mayfly-go/base"
)

type Role struct {
	base.Model

	Name string `orm:"column(name)" json:"username"`
	//AccountId int64  `orm:"column(account_id)`

	Account *Account `orm:"rel(fk);index"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(Role))
}
