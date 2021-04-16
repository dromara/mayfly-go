package models

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/controllers/vo"
)

type Account struct {
	model.Model

	Username string `json:"username"`
	Password string `json:"-"`
	Status   int8   `json:"status"`
}

func ListAccount(param *model.PageParam, args ...interface{}) model.PageResult {
	sql := "SELECT a.id, a.username, a.create_time, a.creator_id, a.creator, r.Id AS 'Role.Id', r.Name AS 'Role.Name'" +
		" FROM t_account a LEFT JOIN t_role r ON a.id = r.account_id"
	return model.GetPageBySql(sql, param, new([]vo.AccountVO), args)
}
