package req

import (
	"github.com/gin-gonic/gin"
)

// 请求配置，如是否需要权限，日志信息等配置
type Conf struct {
	method  string
	path    string
	handler HandlerFunc // 请求处理函数

	requiredPermission *Permission // 需要的权限信息，默认为nil，需要校验token
	logInfo            *LogInfo    // 日志相关信息
	noRes              bool        // 无需返回结果，即文件下载等
}

type Confs struct {
	Group string
	Confs []*Conf
}

func NewConfs(group string, confs ...*Conf) *Confs {
	return &Confs{group, confs}
}

func New(method, path string, handler HandlerFunc) *Conf {
	return &Conf{method: method, path: path, handler: handler, noRes: false}
}

func NewPost(path string, handler HandlerFunc) *Conf {
	return New("POST", path, handler)
}

func NewGet(path string, handler HandlerFunc) *Conf {
	return New("GET", path, handler)
}

func NewPut(path string, handler HandlerFunc) *Conf {
	return New("PUT", path, handler)
}

func NewDelete(path string, handler HandlerFunc) *Conf {
	return New("DELETE", path, handler)
}
func NewAny(path string, handler HandlerFunc) *Conf {
	return New("any", path, handler)
}

func (r *Conf) ToGinHFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		NewCtxWithGin(c).WithConf(r).Handle(r.handler)
	}
}

// 调用该方法设置请求描述，则默认记录日志，并不记录响应结果
func (r *Conf) Log(li *LogInfo) *Conf {
	r.logInfo = li
	return r
}

// 设置请求上下文需要的权限信息
func (r *Conf) RequiredPermission(permission *Permission) *Conf {
	r.requiredPermission = permission
	return r
}

// 设置请求上下文需要的权限信息
func (r *Conf) RequiredPermissionCode(code string) *Conf {
	r.RequiredPermission(NewPermission(code))
	return r
}

// 不需要token校验
func (r *Conf) DontNeedToken() *Conf {
	r.requiredPermission = &Permission{NeedToken: false}
	return r
}

// 没有响应结果，即文件下载等
func (r *Conf) NoRes() *Conf {
	r.noRes = true
	return r
}

// 注册至group
func (r *Conf) Group(gr *gin.RouterGroup) *Conf {
	if r.method == "any" {
		gr.Any(r.path, r.ToGinHFunc())
	} else {
		gr.Handle(r.method, r.path, r.ToGinHFunc())
	}
	return r
}

// 批量注册至group
func BatchSetGroup(gr *gin.RouterGroup, reqs []*Conf) {
	for _, req := range reqs {
		req.Group(gr)
	}
}
