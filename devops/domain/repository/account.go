package repository

import "mayfly-go/devops/domain/entity"

type Account interface {
	// 根据条件获取账号信息
	GetAccount(condition *entity.Account, cols ...string) error
}
