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
type wrapperF struct {
	f F
}

func NewWrapperF(f F) *wrapperF {
	return &wrapperF{f: f}
}

// Header is an intelligent shortcut for c.Writer.Header().Set(key, value).
// It writes a header in the response.
// If value == "", this method removes the header `c.Writer.Header().Del(key)`
func (wf *wrapperF) Header(key, value string) {
	if value == "" {
		wf.GetWriter().Header().Del(key)
		return
	}
	wf.GetWriter().Header().Set(key, value)
}

// get request header value
func (wf *wrapperF) GetHeader(key string) string {
	return wf.GetRequest().Header.Get(key)
}

// 获取查询参数，不存在则返回默认值
func (wf *wrapperF) QueryDefault(qm string, defaultStr string) string {
	qv := wf.Query(qm)
	if qv == "" {
		return defaultStr
	}
	return qv
}

// 获取查询参数中指定参数值，并转为int
func (wf *wrapperF) QueryInt(qm string) int {
	return wf.QueryIntDefault(qm, 0)
}

// 获取查询参数中指定参数值，并转为int， 不存在则返回默认值
func (wf *wrapperF) QueryIntDefault(qm string, defaultInt int) int {
	qv := wf.Query(qm)
	if qv == "" {
		return defaultInt
	}
	qvi, err := strconv.Atoi(qv)
	biz.ErrIsNil(err, "query param not int")
	return qvi
}

// 获取分页参数
func (wf *wrapperF) GetPageParam() model.PageParam {
	return model.PageParam{PageNum: wf.QueryIntDefault("pageNum", 1), PageSize: wf.QueryIntDefault("pageSize", 100)}
}

// 获取路径参数
func (wf *wrapperF) PathParamInt(pm string) int {
	value, err := strconv.Atoi(wf.PathParam(pm))
	biz.ErrIsNilAppendErr(err, "string类型转换int异常: %s")
	return value
}

func (wf *wrapperF) Download(reader io.Reader, filename string) {
	wf.Header("Content-Type", "application/octet-stream")
	wf.Header("Content-Disposition", "attachment; filename="+filename)
	io.Copy(wf.GetWriter(), reader)
}

/************************************/
/************ wrapper F ************/
/************************************/

func (wf *wrapperF) GetRequest() *http.Request {
	return wf.f.GetRequest()
}

func (wf *wrapperF) GetWriter() http.ResponseWriter {
	return wf.f.GetWriter()
}

func (wf *wrapperF) Redirect(code int, location string) {
	wf.f.Redirect(code, location)
}

func (wf *wrapperF) ClientIP() string {
	return wf.f.ClientIP()
}

func (wf *wrapperF) BindJSON(data any) error {
	return wf.f.BindJSON(data)
}

func (wf *wrapperF) BindQuery(data any) error {
	return wf.f.BindQuery(data)
}

func (wf *wrapperF) Query(qm string) string {
	return wf.f.Query(qm)
}

func (wf *wrapperF) PathParam(pm string) string {
	return wf.f.PathParam(pm)
}

func (wf *wrapperF) PostForm(key string) string {
	return wf.f.PostForm(key)
}

func (wf *wrapperF) FormFile(name string) (*multipart.FileHeader, error) {
	return wf.f.FormFile(name)
}

func (wf *wrapperF) MultipartForm() (*multipart.Form, error) {
	return wf.f.MultipartForm()
}

func (wf *wrapperF) JSONRes(code int, data any) {
	wf.f.JSONRes(code, data)
}
