package init

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/infra/persistence"
	"mayfly-go/pkg/starter"
	"mayfly-go/pkg/validatorx"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	// 账号用户名校验
	validatorx.RegisterPattern("account_username", "^[a-zA-Z0-9_]{5,20}$", "只允许输入5-20位大小写字母、数字、下划线")
}
