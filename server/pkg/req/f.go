package req

import (
	"io"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mime/multipart"
	"net/http"
	"strconv"
)

// http请求常用的通用方法（目前系统使用了以下那些方法），F算是framework简称
type F interface {
	GetRequest() *http.Request

	GetWriter() http.ResponseWriter

	Redirect(code int, location string)

	ClientIP() string

	BindJSON(obj any) error

	BindQuery(obj any) error

	Query(qm string) string

	PathParam(pm string) string

	PostForm(key string) string

	FormFile(name string) (*multipart.FileHeader, error)

	MultipartForm() (*multipart.Form, error)

	JSONRes(code int, data any)
}

// wrapper F，提供更多基于F接口方法的封装方法
type WrapperF struct {
	F F
}

func NewWrapperF(f F) *WrapperF {
	return &WrapperF{F: f}
}

// Header is an intelligent shortcut for c.Writer.Header().Set(key, value).
// It writes a header in the response.
// If value == "", this method removes the header `c.Writer.Header().Del(key)`
func (wf *WrapperF) Header(key, value string) {
	if value == "" {
		wf.GetWriter().Header().Del(key)
		return
	}
	wf.GetWriter().Header().Set(key, value)
}

// get request header value
func (wf *WrapperF) GetHeader(key string) string {
	return wf.GetRequest().Header.Get(key)
}

// 获取查询参数，不存在则返回默认值
func (wf *WrapperF) QueryDefault(qm string, defaultStr string) string {
	qv := wf.Query(qm)
	if qv == "" {
		return defaultStr
	}
	return qv
}

// 获取查询参数中指定参数值，并转为int
func (wf *WrapperF) QueryInt(qm string) int {
	return wf.QueryIntDefault(qm, 0)
}

// 获取查询参数中指定参数值，并转为int， 不存在则返回默认值
func (wf *WrapperF) QueryIntDefault(qm string, defaultInt int) int {
	qv := wf.Query(qm)
	if qv == "" {
		return defaultInt
	}
	qvi, err := strconv.Atoi(qv)
	biz.ErrIsNil(err, "query param not int")
	return qvi
}

// 获取分页参数
func (wf *WrapperF) GetPageParam() *model.PageParam {
	return &model.PageParam{PageNum: wf.QueryIntDefault("pageNum", 1), PageSize: wf.QueryIntDefault("pageSize", 10)}
}

// 获取路径参数
func (wf *WrapperF) PathParamInt(pm string) int {
	value, err := strconv.Atoi(wf.PathParam(pm))
	biz.ErrIsNilAppendErr(err, "string类型转换int异常: %s")
	return value
}

func (wf *WrapperF) Download(reader io.Reader, filename string) {
	wf.Header("Content-Type", "application/octet-stream")
	wf.Header("Content-Disposition", "attachment; filename="+filename)
	io.Copy(wf.GetWriter(), reader)
}

/************************************/
/************ wrapper F ************/
/************************************/

func (wf *WrapperF) GetRequest() *http.Request {
	return wf.F.GetRequest()
}

func (wf *WrapperF) GetWriter() http.ResponseWriter {
	return wf.F.GetWriter()
}

func (wf *WrapperF) Redirect(code int, location string) {
	wf.F.Redirect(code, location)
}

func (wf *WrapperF) ClientIP() string {
	return wf.F.ClientIP()
}

func (wf *WrapperF) BindJSON(data any) error {
	return wf.F.BindJSON(data)
}

func (wf *WrapperF) BindQuery(data any) error {
	return wf.F.BindQuery(data)
}

func (wf *WrapperF) Query(qm string) string {
	return wf.F.Query(qm)
}

func (wf *WrapperF) PathParam(pm string) string {
	return wf.F.PathParam(pm)
}

func (wf *WrapperF) PostForm(key string) string {
	return wf.F.PostForm(key)
}

func (wf *WrapperF) FormFile(name string) (*multipart.FileHeader, error) {
	return wf.F.FormFile(name)
}

func (wf *WrapperF) MultipartForm() (*multipart.Form, error) {
	return wf.F.MultipartForm()
}

func (wf *WrapperF) JSONRes(code int, data any) {
	wf.F.JSONRes(200, data)
}
