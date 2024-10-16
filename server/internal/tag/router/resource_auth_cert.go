package router

import (
	"mayfly-go/internal/tag/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitResourceAuthCertRouter(router *gin.RouterGroup) {
	m := new(api.ResourceAuthCert)
	biz.ErrIsNil(ioc.Inject(m))

	resourceAuthCert := router.Group("/auth-certs")
	{
		reqs := [...]*req.Conf{
			req.NewGet("", m.ListByQuery),

			req.NewGet("/simple", m.SimpleAc),

			req.NewGet("/detail", m.GetCompleteAuthCert).Log(req.NewLogSave("授权凭证-查看密文")).RequiredPermissionCode("authcert:showciphertext"),

			req.NewPost("", m.SaveAuthCert).Log(req.NewLogSave("授权凭证-保存")).RequiredPermissionCode("authcert:save"),

			req.NewDelete(":id", m.Delete).Log(req.NewLogSave("授权凭证-删除")).RequiredPermissionCode("authcert:del"),
		}

		req.BatchSetGroup(resourceAuthCert, reqs[:])
	}
}
