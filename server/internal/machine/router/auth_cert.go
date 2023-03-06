package router

import (
	"mayfly-go/internal/machine/api"
	"mayfly-go/internal/machine/application"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitAuthCertRouter(router *gin.RouterGroup) {
	r := &api.AuthCert{AuthCertApp: application.GetAuthCertApp()}
	db := router.Group("sys/authcerts")
	{
		listAcP := req.NewPermission("authcert")
		db.GET("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithRequiredPermission(listAcP).
				Handle(r.AuthCerts)
		})

		// 基础授权凭证信息，不包含密码等
		db.GET("base", func(c *gin.Context) {
			req.NewCtxWithGin(c).Handle(r.BaseAuthCerts)
		})

		saveAc := req.NewLogInfo("保存授权凭证").WithSave(true)
		saveAcP := req.NewPermission("authcert:save")
		db.POST("", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(saveAc).
				WithRequiredPermission(saveAcP).
				Handle(r.SaveAuthCert)
		})

		deleteAc := req.NewLogInfo("删除授权凭证").WithSave(true)
		deleteAcP := req.NewPermission("authcert:del")
		db.DELETE(":id", func(c *gin.Context) {
			req.NewCtxWithGin(c).
				WithLog(deleteAc).
				WithRequiredPermission(deleteAcP).
				Handle(r.Delete)
		})
	}
}
