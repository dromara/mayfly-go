package api

import (
	"mayfly-go/internal/file/api/vo"
	"mayfly-go/internal/file/application"
	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strings"
)

type File struct {
	fileApp application.File `inject:"T"`
}

func (f *File) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 元数据端点需登录：它会按 key 回显文件名与大小，若允许匿名查则成为探测他人数据类产物的旁路
		// （SPA 内的调用方走 axios 封装，本就会附带 Authorization 头，不因此改变行为）
		req.NewGet("/detail/:keys", f.GetFileByKeys),

		// 文件内容端点保持路由级不校 token：头像、AI 附件等展示类资产由 <img src>/<a href> 直接引用，
		// 浏览器无法为其附加 Authorization 头。访问控制下沉到按文件访问模式判定（见 GetFileContent）：
		// public 匿名可读，private 一律拒读并提示改走所属模块带权限码的下载入口
		req.NewGet("/:key", f.GetFileContent).DontNeedToken().NoRes(),

		// 上传端点服务于展示类资产（头像、AI 附件、终端分发源文件），访问模式固定 public；
		// 不接受客户端自报访问模式，否则上传方可把数据产物声明成公开文件绕过收口
		req.NewPost("/upload", f.Upload).Log(req.NewLogSave("file-文件上传")),

		req.NewDelete("/:key", f.Remove).Log(req.NewLogSave("file-文件删除")),
	}

	return req.NewConfs("/sys/files", reqs[:]...)
}

func (f *File) GetFileByKeys(rc *req.Ctx) {
	keysStr := rc.PathParam("keys")
	biz.NotEmpty(keysStr, "keys cannot be empty")

	var files []vo.SimpleFile
	biz.ErrIsNil(f.fileApp.ListPublicByKeys(strings.Split(keysStr, ","), &files))
	rc.ResData = files
}

func (f *File) GetFileContent(rc *req.Ctx) {
	key := rc.PathParam("key")
	biz.NotEmpty(key, "key cannot be empty")

	// 先过访问模式判定，再开文件流：私有产物不能因知道 key 就被匿名拉走
	biz.ErrIsNil(f.fileApp.CanAnonymousRead(rc.MetaCtx, key))

	filename, reader, err := f.fileApp.GetReader(rc.MetaCtx, key)
	biz.ErrIsNil(err)
	defer reader.Close()
	rc.Download(reader, filename)
}

func (f *File) Upload(rc *req.Ctx) {
	multipart, err := rc.GetRequest().MultipartReader()
	biz.ErrIsNilAppendErr(err, "read file error: %s")
	file, err := multipart.NextPart()
	biz.ErrIsNilAppendErr(err, "read file error: %s")
	defer file.Close()

	fileKey, err := f.fileApp.Upload(rc.MetaCtx, rc.Query("fileKey"), file.FileName(), file, entity.FileAccessModePublic)
	biz.ErrIsNil(err)
	rc.ResData = fileKey
}

func (f *File) Remove(rc *req.Ctx) {
	biz.ErrIsNil(f.fileApp.Remove(rc.MetaCtx, rc.PathParam("key")))
}
