package init

import (
	"mayfly-go/internal/sys/api"
	"mayfly-go/internal/sys/application"
	"mayfly-go/internal/sys/infra/persistence"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/starter"
	"mayfly-go/pkg/validatorx"
)

func init() {
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.InitIoc()
		api.InitIoc()
	})

	// 权限码缓存回源：进程内缓存未命中时从 DB 加载并回填，
	// 避免后端重启后旧 token 仍有效但缓存为空，首个受保护请求被误判为无权限
	req.SetPermissionCodeLoader(application.LoadPermissionCodes)

	// 账号用户名校验
	validatorx.RegisterPattern("account_username", "^[a-zA-Z0-9_]{5,20}$", "只允许输入5-20位大小写字母、数字、下划线")
}
