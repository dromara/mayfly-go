package init

import (
	"mayfly-go/internal/common/api"
	"mayfly-go/pkg/starter"
)

func init() {
	starter.AddInitIocFunc(func() {
		api.InitIoc()
	})
}
