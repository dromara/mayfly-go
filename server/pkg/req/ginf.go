package req

import (
	"mayfly-go/pkg/contextx"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewCtxWithGin(g *gin.Context) *Ctx {
	return &Ctx{F: NewWrapperF(&GinF{ginCtx: g}), MetaCtx: contextx.WithTraceId(g.Request.Context())}
}

type GinF struct {
	ginCtx *gin.Context
}

func (gf *GinF) GetRequest() *http.Request {
	return gf.ginCtx.Request
}

func (gf *GinF) GetWriter() http.ResponseWriter {
	return gf.ginCtx.Writer
}

func (gf GinF) Redirect(code int, location string) {
	gf.ginCtx.Redirect(code, location)
}

func (gf *GinF) ClientIP() string {
	return gf.ginCtx.ClientIP()
}

func (gf *GinF) BindJSON(data any) error {
	return gf.ginCtx.ShouldBindJSON(data)
}

func (gf *GinF) BindQuery(data any) error {
	return gf.ginCtx.BindQuery(data)
}

func (gf *GinF) Query(qm string) string {
	return gf.ginCtx.Query(qm)
}

func (gf *GinF) PathParam(pm string) string {
	return gf.ginCtx.Param(pm)
}

func (gf *GinF) PostForm(key string) string {
	return gf.ginCtx.PostForm(key)
}

func (gf *GinF) FormFile(name string) (*multipart.FileHeader, error) {
	return gf.ginCtx.FormFile(name)
}

func (gf *GinF) MultipartForm() (*multipart.Form, error) {
	return gf.ginCtx.MultipartForm()
}

func (gf *GinF) JSONRes(code int, data any) {
	gf.ginCtx.JSON(200, data)
}
