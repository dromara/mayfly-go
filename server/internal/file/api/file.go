package api

import (
	"mayfly-go/internal/file/api/vo"
	"mayfly-go/internal/file/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"strings"
)

type File struct {
	fileApp application.File `inject:"T"`
}

func (f *File) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("/detail/:keys", f.GetFileByKeys).DontNeedToken(),

		req.NewGet("/:key", f.GetFileContent).DontNeedToken().NoRes(),

		req.NewPost("/upload", f.Upload).Log(req.NewLogSave("file-文件上传")),

		req.NewDelete("/:key", f.Remove).Log(req.NewLogSave("file-文件删除")),
	}

	return req.NewConfs("/sys/files", reqs[:]...)
}

func (f *File) GetFileByKeys(rc *req.Ctx) {
	keysStr := rc.PathParam("keys")
	biz.NotEmpty(keysStr, "keys cannot be empty")

	var files []vo.SimpleFile
	err := f.fileApp.ListByCondToAny(model.NewCond().In("file_key", strings.Split(keysStr, ",")), &files)
	biz.ErrIsNil(err)
	rc.ResData = files
}

func (f *File) GetFileContent(rc *req.Ctx) {
	key := rc.PathParam("key")
	biz.NotEmpty(key, "key cannot be empty")

	filename, reader, err := f.fileApp.GetReader(rc.MetaCtx, key)
	if err != nil {
		rc.GetWriter().Write([]byte(err.Error()))
		return
	}
	defer reader.Close()
	rc.Download(reader, filename)
}

func (f *File) Upload(rc *req.Ctx) {
	multipart, err := rc.GetRequest().MultipartReader()
	biz.ErrIsNilAppendErr(err, "read file error: %s")
	file, err := multipart.NextPart()
	biz.ErrIsNilAppendErr(err, "read file error: %s")
	defer file.Close()

	fileKey, err := f.fileApp.Upload(rc.MetaCtx, rc.Query("fileKey"), file.FileName(), file)
	biz.ErrIsNil(err)
	rc.ResData = fileKey
}

func (f *File) Remove(rc *req.Ctx) {
	biz.ErrIsNil(f.fileApp.Remove(rc.MetaCtx, rc.PathParam("key")))
}
