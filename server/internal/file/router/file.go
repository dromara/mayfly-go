package router

import (
	"mayfly-go/internal/file/api"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/req"

	"github.com/gin-gonic/gin"
)

func InitFileRouter(router *gin.RouterGroup) {
	file := router.Group("sys/files")
	f := new(api.File)
	biz.ErrIsNil(ioc.Inject(f))

	reqs := [...]*req.Conf{

		req.NewGet("/detail/:keys", f.GetFileByKeys).DontNeedToken(),

		req.NewGet("/:key", f.GetFileContent).DontNeedToken().NoRes(),

		req.NewPost("/upload", f.Upload).Log(req.NewLogSave("file-文件上传")),

		req.NewDelete("/:key", f.Remove).Log(req.NewLogSave("file-文件删除")),
	}

	req.BatchSetGroup(file, reqs[:])
}
