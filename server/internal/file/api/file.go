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
	FileApp application.File `inject:""`
}

func (f *File) GetFileByKeys(rc *req.Ctx) {
	keysStr := rc.PathParam("keys")
	biz.NotEmpty(keysStr, "keys cannot be empty")

	var files []vo.SimpleFile
	err := f.FileApp.ListByCondToAny(model.NewCond().In("file_key", strings.Split(keysStr, ",")), &files)
	biz.ErrIsNil(err)
	rc.ResData = files
}

func (f *File) GetFileContent(rc *req.Ctx) {
	key := rc.PathParam("key")
	biz.NotEmpty(key, "key cannot be empty")

	filename, reader, err := f.FileApp.GetReader(rc.MetaCtx, key)
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

	fileKey, err := f.FileApp.Upload(rc.MetaCtx, rc.Query("fileKey"), file.FileName(), file)
	biz.ErrIsNil(err)
	rc.ResData = fileKey
}

func (f *File) Remove(rc *req.Ctx) {
	biz.ErrIsNil(f.FileApp.Remove(rc.MetaCtx, rc.PathParam("key")))
}
