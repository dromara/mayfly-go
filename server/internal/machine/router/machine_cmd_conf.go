package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitMachineCmdConfRouter(router *gin.RouterGroup) {
	mccs := router.Group("machine/security/cmd-confs")

	mcc := new(api.MachineCmdConf)
	biz.ErrIsNil(ioc.Inject(mcc))

	reqs := [...]*req.Conf{
		req.NewGet("", mcc.MachineCmdConfs),

		req.NewPost("", mcc.Save).Log(req.NewLogSave("机器命令配置-保存")).RequiredPermissionCode("cmdconf:save"),

		req.NewDelete(":id", mcc.Delete).Log(req.NewLogSave("机器命令配置-删除")).RequiredPermissionCode("cmdconf:del"),
	}

	req.BatchSetGroup(mccs, reqs[:])
}
