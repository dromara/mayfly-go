package init

import (
	"mayfly-go/initialize"
	"mayfly-go/internal/common/api"
)

func init() {
	initialize.AddInitIocFunc(func() {
		api.InitIoc()
	})
}
