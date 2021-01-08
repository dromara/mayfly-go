package models

import (
	"mayfly-go/base/model"
	"mayfly-go/controllers/vo"

	"github.com/beego/beego/v2/client/orm"
)

type Account struct {
	model.Model

	Username string `orm:"column(username)" json:"username"`
	Password string `orm:"column(password)" json:"-"`
	Status   int8   `json:"status"`
}

func init() {
	orm.RegisterModelWithPrefix("t_", new(Account))
}

func ListAccount(param *model.PageParam, args ...interface{}) model.PageResult {
	sql := "SELECT a.id, a.username, a.create_time, a.creator_id, a.creator, r.Id AS 'Role.Id', r.Name AS 'Role.Name'" +
		" FROM t_account a LEFT JOIN t_role r ON a.id = r.account_id"
	return model.GetPageBySql(sql, new([]vo.AccountVO), param, args)
}
