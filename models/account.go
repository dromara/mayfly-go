package models

import (
	"github.com/astaxie/beego/orm"
	"mayfly-go/base"
	"mayfly-go/controllers/vo"
)

type Account struct {
	base.Model

	Username string `orm:"column(username)" json:"username"`
	Password string `orm:"column(password)" json:"-"`
	Status   int8   `json:"status"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(Account))
}

func ListAccount(param *base.PageParam, args ...interface{}) base.PageResult {
	sql := "SELECT a.id, a.username, a.create_time, a.creator_id, a.creator, r.Id AS 'Role.Id', r.Name AS 'Role.Name'" +
		" FROM t_account a LEFT JOIN t_role r ON a.id = r.account_id"
	return base.GetPageBySql(sql, new([]vo.AccountVO), param, args)
}
