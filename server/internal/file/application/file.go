package application

import (
	"context"
	"io"

	"mayfly-go/internal/file/config"
	"mayfly-go/internal/file/domain/entity"
	"mayfly-go/internal/file/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/utils/stringx"
	"mayfly-go/pkg/utils/writerx"
	"os"
	"path/filepath"
	"time"

	"github.com/may-fly/cast"
)

type File interface {
	base.App[*entity.File]

	// Upload 上传文件
	//
	// @param fileKey 文件key，若存在则使用存在的文件key，否则生成新的文件key。
	//
	// @param filename 文件名，带文件后缀
	//
	// @return fileKey 文件key
	Upload(ctx context.Context, fileKey string, filename string, r io.Reader) (string, error)

	// NewWriter 创建文件writer
	//
	// @param canEmptyFileKey 文件key，若不为空则使用该文件key，否则生成新的文件key。
	//
	// @param filename 文件名，带文件后缀
	//
	// @return fileKey 文件key
	//
	// @return writer 文件writer
	//
	// @return saveFunc(*error) 保存文件信息的回调函数 (必须要defer中调用才会入库保存该文件信息)，若*error不为nil则表示业务逻辑处理失败，不需要保存文件信息并将创建的文件删除
	NewWriter(ctx context.Context, canEmptyFileKey string, filename string) (fileKey string, writer *writerx.CountingWriteCloser, saveFunc func(*error) error, err error)

	// GetReader 获取文件reader
	//
	// @return filename 文件名
	//
	// @return reader 文件reader
	//
	// @return err 错误
	GetReader(ctx context.Context, fileKey string) (string, io.ReadCloser, error)

	// Remove 删除文件
	Remove(ctx context.Context, fileKey string) error
}

type fileAppImpl struct {
	base.AppImpl[*entity.File, repository.File]
}

func (f *fileAppImpl) Upload(ctx context.Context, fileKey string, filename string, r io.Reader) (string, error) {
	var err error
	fileKey, writer, saveFileFunc, err := f.NewWriter(ctx, fileKey, filename)
	if err != nil {
		return fileKey, err
	}
	defer saveFileFunc(&err)

	if _, err = io.Copy(writer, r); err != nil {
		return fileKey, err
	}
	return fileKey, nil
}

func (f *fileAppImpl) NewWriter(ctx context.Context, canEmptyFileKey string, filename string) (fileKey string, writer *writerx.CountingWriteCloser, saveFunc func(*error) error, err error) {
	isNewFile := true
	file := &entity.File{}

	if canEmptyFileKey == "" {
		canEmptyFileKey = stringx.RandUUID()
		file.FileKey = canEmptyFileKey
	} else {
		file.FileKey = canEmptyFileKey
		if err := f.GetByCond(file); err == nil {
			isNewFile = false
		}
	}
	file.Filename = filename

	if !isNewFile {
		// 先删除旧文件
		f.remove(ctx, file)
	}

	// 生产新的文件名
	newFilename := canEmptyFileKey + filepath.Ext(filename)
	filepath, w, err := f.newWriter(newFilename)
	if err != nil {
		return "", nil, nil, err
	}
	file.Path = filepath

	fileKey = canEmptyFileKey
	writer = writerx.NewCountingWriteCloser(w)
	// 创建回调函数
	saveFunc = func(e *error) error {
		if e != nil {
			err := *e
			if err != nil {
				logx.Errorf("the write file business logic failed: %s", err.Error())
				// 删除已经创建的文件
				f.remove(ctx, file)
				return err
			}
		}
		// 获取已写入的字节数
		file.Size = writer.BytesWritten()
		writer.Close()
		// 保存文件信息
		return f.Save(ctx, file)
	}

	return fileKey, writer, saveFunc, nil
}

func (f *fileAppImpl) GetReader(ctx context.Context, fileKey string) (string, io.ReadCloser, error) {
	file := &entity.File{FileKey: fileKey}
	if err := f.GetByCond(file); err != nil {
		return "", nil, errorx.NewBiz("file not found")
	}
	r, err := os.Open(filepath.Join(config.GetFileConfig().BasePath, file.Path))
	return file.Filename, r, err
}

func (f *fileAppImpl) Remove(ctx context.Context, fileKey string) error {
	file := &entity.File{FileKey: fileKey}
	if err := f.GetByCond(file); err != nil {
		return errorx.NewBiz("file not found")
	}
	f.DeleteById(ctx, file.Id)
	return f.remove(ctx, file)
}

func (f *fileAppImpl) newWriter(filename string) (string, io.WriteCloser, error) {
	now := time.Now()
	filePath := filepath.Join(cast.ToString(now.Year()), cast.ToString(int(now.Month())), cast.ToString(now.Day()), cast.ToString(now.Hour()), filename)
	fileAbsPath := filepath.Join(config.GetFileConfig().BasePath, filePath)

	// 目录不存在则创建
	fileDir := filepath.Dir(fileAbsPath)
	if _, err := os.Stat(fileDir); os.IsNotExist(err) {
		err = os.MkdirAll(fileDir, os.ModePerm)
		if err != nil {
			return "", nil, err
		}
	}

	// 创建目标文件
	out, err := os.OpenFile(fileAbsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0766)
	if err != nil {
		return "", nil, err
	}
	return filePath, out, nil
}

func (f *fileAppImpl) remove(ctx context.Context, file *entity.File) error {
	if err := os.Remove(filepath.Join(config.GetFileConfig().BasePath, file.Path)); err != nil {
		logx.ErrorfContext(ctx, "failed to delete old file [%s]: %s", file.Path, err.Error())
		return err
	}
	return nil
}
